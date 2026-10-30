package hx

import (
	"context"
	"errors"
	"fmt"
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

func TestResourceForWithSkipsTheFirstLoadAndFollowsTheSource(t *testing.T) {
	query := Signal("a")
	seen := make(chan string, 10)
	var r *ResourceT[string]
	dispose := CreateRoot(func(dispose func()) func() {
		r = ResourceForWith("from the server", query, func(ctx context.Context, q string) (string, error) {
			seen <- q
			return strings.ToUpper(q), nil
		})
		return dispose
	})
	defer dispose()
	settled := settledChannel(r)

	select {
	case q := <-seen:
		t.Fatalf("the first load must be skipped, loaded %q", q)
	case <-time.After(30 * time.Millisecond):
	}
	if got := UntrackGet(r.Value()); got != "from the server" {
		t.Fatalf("value=%q", got)
	}

	query.Set("b")
	if got := <-seen; got != "b" {
		t.Fatalf("a change of the source must load, got %q", got)
	}
	waitFor(t, settled, "the load")
	if got := UntrackGet(r.Value()); got != "B" {
		t.Fatalf("value=%q", got)
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

// The end of a run is one update for the UI: an effect that reads the result and
// pending never sees one without the other, and runs once.
func TestTheEndOfAnActionIsOneUpdate(t *testing.T) {
	save := Action(func(ctx context.Context, n int) (int, error) { return n * 2, nil })
	settled := make(chan struct{}, 10)
	save.setSettled(func() { settled <- struct{}{} }) // after the batch of the end of the run
	var seen []string
	banner := Signal("")
	save.OnSuccess(func(result int, _ int) { banner.Set(fmt.Sprintf("saved %d", result)) })
	dispose := CreateRoot(func(dispose func()) func() {
		EffectFunc(func() {
			seen = append(seen, fmt.Sprintf("pending=%v value=%d banner=%q", save.Pending().Get(), save.Value().Get(), banner.Get()))
		})
		return dispose
	})
	defer dispose()

	save.Run(21)
	waitFor(t, settled, "the action")

	// The state of the end is one update (never "not running" without the answer),
	// and what the callback sets is the next one.
	want := []string{
		`pending=false value=0 banner=""`,
		`pending=true value=0 banner=""`,
		`pending=false value=42 banner=""`,
		`pending=false value=42 banner="saved 42"`,
	}
	if strings.Join(seen, " | ") != strings.Join(want, " | ") {
		t.Fatalf("the effect saw:\n  %s\nwant:\n  %s", strings.Join(seen, "\n  "), strings.Join(want, "\n  "))
	}
}
