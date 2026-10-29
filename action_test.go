package hx

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"
)

func waitFor(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

func TestActionSuccess(t *testing.T) {
	done := make(chan struct{})
	var gotArg, gotResult int
	release := make(chan struct{})
	save := Action(func(ctx context.Context, n int) (int, error) { <-release; return n * 2, nil }).
		OnSuccess(func(result, arg int) { gotResult, gotArg = result, arg; close(done) })

	save.Run(21)
	if !UntrackGet(save.Pending()) {
		t.Fatal("must be pending as soon as it runs")
	}
	close(release)
	waitFor(t, done, "the action")

	if UntrackGet(save.Pending()) || UntrackGet(save.Error()) != nil || UntrackGet(save.Value()) != 42 {
		t.Fatalf("pending=%v err=%v value=%v", UntrackGet(save.Pending()), UntrackGet(save.Error()), UntrackGet(save.Value()))
	}
	if gotArg != 21 || gotResult != 42 {
		t.Fatalf("OnSuccess got arg=%d result=%d", gotArg, gotResult)
	}
}

func TestActionErrorKeepsTheLastValue(t *testing.T) {
	boom := errors.New("boom")
	fail := false
	done := make(chan struct{}, 2)
	action := Action(func(ctx context.Context, _ struct{}) (string, error) {
		if fail {
			return "", boom
		}
		return "ok", nil
	}).OnSuccess(func(string, struct{}) { done <- struct{}{} }).
		OnError(func(error, struct{}) { done <- struct{}{} })

	action.Run(struct{}{})
	waitFor(t, done, "the first run")
	fail = true
	action.Run(struct{}{})
	waitFor(t, done, "the second run")

	if UntrackGet(action.Error()) != boom || UntrackGet(action.Value()) != "ok" || UntrackGet(action.Pending()) {
		t.Fatalf("err=%v value=%q pending=%v", UntrackGet(action.Error()), UntrackGet(action.Value()), UntrackGet(action.Pending()))
	}

	fail = false
	action.Run(struct{}{})
	if UntrackGet(action.Error()) != nil {
		t.Fatal("the error must be cleared when it runs again")
	}
	waitFor(t, done, "the third run")
}

func TestActionLastCallWins(t *testing.T) {
	release := make(chan struct{})
	canceled := make(chan struct{})
	started := make(chan struct{}, 2)
	done := make(chan struct{})
	action := Action(func(ctx context.Context, n int) (int, error) {
		started <- struct{}{}
		if n == 1 {
			select {
			case <-ctx.Done(): // the first run must be canceled by the second
				close(canceled)
				<-release
				return 1, nil
			case <-time.After(2 * time.Second):
				return 1, nil
			}
		}
		return n, nil
	}).OnSuccess(func(int, int) { close(done) })

	action.Run(1)
	waitFor(t, started, "the first run to start")
	action.Run(2)
	waitFor(t, canceled, "the first context to be canceled")
	waitFor(t, done, "the second run")
	close(release) // the first one returns late
	time.Sleep(20 * time.Millisecond)

	if UntrackGet(action.Value()) != 2 {
		t.Fatalf("the result of a superseded run must be ignored, value=%d", UntrackGet(action.Value()))
	}
	if UntrackGet(action.Pending()) {
		t.Fatal("must not be pending")
	}
}

func TestActionCancel(t *testing.T) {
	started := make(chan struct{})
	finished := make(chan struct{})
	errorCalled := false
	action := Action(func(ctx context.Context, _ struct{}) (int, error) {
		close(started)
		<-ctx.Done()
		defer close(finished)
		return 0, ctx.Err()
	}).OnError(func(error, struct{}) { errorCalled = true })

	action.Run(struct{}{})
	waitFor(t, started, "the run")
	action.Cancel()
	waitFor(t, finished, "the run to notice")
	time.Sleep(20 * time.Millisecond)

	if UntrackGet(action.Pending()) || UntrackGet(action.Error()) != nil || errorCalled {
		t.Fatalf("pending=%v err=%v errorCalled=%v", UntrackGet(action.Pending()), UntrackGet(action.Error()), errorCalled)
	}
	action.Cancel() // nothing running: must be harmless
}

func TestActionReset(t *testing.T) {
	done := make(chan struct{})
	action := Action(func(ctx context.Context, _ struct{}) (int, error) { return 7, nil }).
		OnSuccess(func(int, struct{}) { close(done) })
	action.Run(struct{}{})
	waitFor(t, done, "the run")
	action.Reset()
	if UntrackGet(action.Value()) != 0 || UntrackGet(action.Error()) != nil {
		t.Fatal("Reset must forget the result")
	}
}

func TestActionDrivesTheUI(t *testing.T) {
	release := make(chan struct{})
	done := make(chan struct{})
	action := Action(func(ctx context.Context, _ struct{}) (string, error) {
		<-release
		return "saved", nil
	}).OnSuccess(func(string, struct{}) { close(done) })

	var texts []string
	dispose := CreateRoot(func(dispose func()) func() {
		EffectFunc(func() {
			if action.Pending().Get() {
				texts = append(texts, "saving")
			} else {
				texts = append(texts, "idle:"+action.Value().Get())
			}
		})
		return dispose
	})

	action.Run(struct{}{})
	close(release)
	waitFor(t, done, "the run")
	dispose()

	if texts[0] != "idle:" || texts[len(texts)-1] != "idle:saved" {
		t.Fatalf("got %v", texts)
	}
	for _, text := range texts[1 : len(texts)-1] {
		if text != "saving" {
			t.Fatalf("the UI saw %q between the start and the answer: %v", text, texts)
		}
	}
}

// settledChannel makes the resource tell, on a channel, each time a load ends.
func settledChannel[T any](r *ResourceT[T]) chan struct{} {
	ch := make(chan struct{}, 10)
	r.settled(func() { ch <- struct{}{} })
	return ch
}

func TestResourceLoadsWhenMountedAndRefetches(t *testing.T) {
	n := 0
	started := make(chan struct{}, 10)
	var r *ResourceT[int]
	// the load starts as soon as the resource is created (outside of a hydration)
	// so the hook is set before it can end: the fetch waits for it
	hookSet := make(chan struct{})
	r = Resource(func(ctx context.Context) (int, error) {
		<-hookSet
		n++
		started <- struct{}{}
		return n * 10, nil
	})
	settled := settledChannel(r)
	close(hookSet)
	waitFor(t, settled, "the first load")

	if UntrackGet(r.Value()) != 10 || UntrackGet(r.Loading()) || UntrackGet(r.Error()) != nil {
		t.Fatalf("value=%d loading=%v err=%v", UntrackGet(r.Value()), UntrackGet(r.Loading()), UntrackGet(r.Error()))
	}
	r.Refetch()
	waitFor(t, settled, "the refetch")
	if UntrackGet(r.Value()) != 20 {
		t.Fatalf("value=%d", UntrackGet(r.Value()))
	}
	r.Set(99)
	if UntrackGet(r.Value()) != 99 {
		t.Fatal("Set must replace the value")
	}
}

func TestResourceWithStartsFromTheInitialValueAndDoesNotLoad(t *testing.T) {
	called := make(chan struct{}, 1)
	r := ResourceWith(5, func(ctx context.Context) (int, error) { called <- struct{}{}; return 6, nil })
	settled := settledChannel(r)
	select {
	case <-called:
		t.Fatal("ResourceWith must not load by itself")
	case <-time.After(30 * time.Millisecond):
	}
	if UntrackGet(r.Value()) != 5 {
		t.Fatalf("value=%d", UntrackGet(r.Value()))
	}
	r.Refetch()
	waitFor(t, settled, "the refetch")
	if UntrackGet(r.Value()) != 6 {
		t.Fatalf("value=%d", UntrackGet(r.Value()))
	}
}

func TestResourceIsNotLoadedWhileRenderingOnAServer(t *testing.T) {
	called := make(chan struct{}, 1)
	RenderToString(func() INode {
		Resource(func(ctx context.Context) (int, error) { called <- struct{}{}; return 1, nil })
		return Div()
	})
	select {
	case <-called:
		t.Fatal("a server render must not load resources")
	case <-time.After(30 * time.Millisecond):
	}
}

func TestResourceForReloadsWhenTheSourceChanges(t *testing.T) {
	query := Signal("a")
	seen := make(chan string, 10)
	hookSet := make(chan struct{})
	var r *ResourceT[string]
	dispose := CreateRoot(func(dispose func()) func() {
		r = ResourceFor[string, string](query, func(ctx context.Context, q string) (string, error) {
			<-hookSet
			seen <- q
			return strings.ToUpper(q), nil
		})
		return dispose
	})
	defer dispose()
	settled := settledChannel(r)
	close(hookSet)

	if got := <-seen; got != "a" {
		t.Fatalf("first load used %q", got)
	}
	waitFor(t, settled, "the first load")
	if UntrackGet(r.Value()) != "A" {
		t.Fatalf("value=%q", UntrackGet(r.Value()))
	}

	query.Set("b")
	if got := <-seen; got != "b" {
		t.Fatalf("second load used %q", got)
	}
	waitFor(t, settled, "the second load")
	if UntrackGet(r.Value()) != "B" {
		t.Fatalf("value=%q", UntrackGet(r.Value()))
	}

	r.Refetch()
	if got := <-seen; got != "b" {
		t.Fatalf("Refetch must use the current source, got %q", got)
	}
	waitFor(t, settled, "the refetch")
}

func TestBodyEncoding(t *testing.T) {
	ct, data, _ := TextBody("text/csv", "a,b").encode()
	if ct != "text/csv" || string(data) != "a,b" {
		t.Fatalf("text: %q %q", ct, data)
	}
	// a url.Values can be passed as it is
	ct, data, _ = FormValues(url.Values{"b": {"2"}, "a": {"1", "x y&z=é"}}).encode()
	if ct != "application/x-www-form-urlencoded" || string(data) != "a=1&a=x+y%26z%3D%C3%A9&b=2" {
		t.Fatalf("values: %q %q", ct, data)
	}
	// the same as the standard library writes
	values := url.Values{"name": {"Ada Lovelace"}, "k~v-_.": {"/?#[]@!$'()*+,;="}, "ü": {"ñ"}}
	if got := encodeValues(values); got != values.Encode() {
		t.Fatalf("encodeValues differs from url.Values.Encode:\n got %s\nwant %s", got, values.Encode())
	}
	if _, _, err := FormOf(Form()).encode(); err == nil {
		t.Fatal("a form is built by the browser: encode must say so")
	}
}

func TestMultipartBuilder(t *testing.T) {
	m := NewMultipart().Field("name", "Ada").Bytes("avatar", "a.png", "image/png", []byte{1, 2}).File("doc", File{Name: "d.txt", Size: 3})
	if len(m.parts) != 3 || m.parts[0].kind != partField || m.parts[1].kind != partBytes || m.parts[2].kind != partFile {
		t.Fatalf("parts=%+v", m.parts)
	}
	if m.Body().kind != bodyMultipart {
		t.Fatal("wrong body kind")
	}
}

func TestResponse(t *testing.T) {
	ok := Response{Status: 200, body: []byte(`{"n":3}`)}
	if !ok.OK() || ok.Err() != nil || ok.Text() != `{"n":3}` || string(ok.Bytes()) != `{"n":3}` {
		t.Fatalf("ok response: %v", ok.Err())
	}
	bad := Response{Status: 422, StatusText: "Unprocessable", body: []byte("  the email is not valid\n")}
	var httpErr *HTTPError
	if bad.OK() || !errors.As(bad.Err(), &httpErr) || httpErr.Status != 422 || httpErr.Body == "" {
		t.Fatalf("bad response: %v", bad.Err())
	}
	if httpErr.Error() != "422 Unprocessable: the email is not valid" {
		t.Fatalf("message: %q", httpErr.Error())
	}
	if (&HTTPError{Status: 500, StatusText: "Oops"}).Error() != "500 Oops" {
		t.Fatal("an error without body")
	}
	long := &HTTPError{Status: 500, StatusText: "x", Body: strings.Repeat("a", 500)}
	if len(long.Error()) > 230 {
		t.Fatalf("a long body must be cut, got %d", len(long.Error()))
	}
}

func TestProgressFraction(t *testing.T) {
	if (Progress{Loaded: 50, Total: 200}).Fraction() != 0.25 {
		t.Fatal("0.25 expected")
	}
	if (Progress{Loaded: 50}).Fraction() != 0 {
		t.Fatal("unknown total must be 0")
	}
}

func TestIsCanceledAndIsTimeout(t *testing.T) {
	if !IsCanceled(context.Canceled) || IsCanceled(context.DeadlineExceeded) || !IsTimeout(context.DeadlineExceeded) {
		t.Fatal("wrong classification")
	}
	wrapped := errors.Join(errors.New("x"), context.Canceled)
	if !IsCanceled(wrapped) {
		t.Fatal("must see through wrapping")
	}
}

func TestCheckedRendersAndBinds(t *testing.T) {
	on := Signal(true)
	got := RenderToString(func() INode {
		return Div().Body(
			Input().Type("checkbox").Attribute("name", "a").BindChecked(on),
			Input().Type("checkbox").Attribute("name", "b"),
			Input().Type("checkbox").Attribute("name", "c").Checked(true),
		)
	})
	want := `<div><input checked="" name="a" type="checkbox"/><input name="b" type="checkbox"/><input checked="" name="c" type="checkbox"/></div>`
	if got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
}

func TestBindOnCheckedRegistersAChangeHandler(t *testing.T) {
	box := Input().Type("checkbox").BindOnChecked(Signal(false))
	if _, ok := box.eventListeners[EventChange]; !ok {
		t.Fatal("BindOnChecked must listen to change")
	}
}
