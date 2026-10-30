package hx

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestRenderToString(t *testing.T) {
	count := Signal(3)
	got := RenderToString(func() INode {
		return Div().Class("card").Body(
			H1().Text("Hello"),
			P().BindText(Computed(func() string { return fmt.Sprintf("count %d", count.Get()) })),
			Input().Placeholder("name").Value("ada"),
			Br(),
		)
	})
	want := `<div class="card"><h1>Hello</h1><p>count 3</p><input value="ada" placeholder="name"/><br/></div>`
	if got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
}

func TestRenderToStringRunsEffectsOnceAndReleasesThem(t *testing.T) {
	s := Signal(1)
	runs := 0
	RenderToString(func() INode {
		return Div().Effect(func() { runs++; s.Get() })
	})
	if runs != 1 {
		t.Fatalf("runs=%d, want 1", runs)
	}
	s.Set(2)
	if runs != 1 {
		t.Fatalf("an effect of a server render kept running: runs=%d", runs)
	}
	if len(s.subscribers) != 0 {
		t.Fatalf("subscribers=%d", len(s.subscribers))
	}
}

func TestRenderToStringIgnoresOnMount(t *testing.T) {
	called := false
	RenderToString(func() INode {
		OnMount(func() { called = true })
		return Div()
	})
	if called {
		t.Fatal("OnMount must not run while rendering to a string")
	}
	if currentMountMode != mountNow {
		t.Fatal("the mount mode was not restored")
	}
}

func TestOnMountOutsideARenderRunsRightAway(t *testing.T) {
	called := false
	OnMount(func() { called = true })
	if !called {
		t.Fatal("OnMount must run right away outside of a render")
	}
}

func TestRenderToStringIsSafeToCallFromManyGoroutines(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			name := Signal("x")
			got := RenderToString(func() INode {
				return Span().BindText(name)
			})
			if got != `<span>x</span>` {
				t.Errorf("got %s", got)
			}
		}()
	}
	wg.Wait()
}

func TestRenderToStringOfEachAndConditionals(t *testing.T) {
	items := Signal([]string{"a", "b"})
	got := RenderToString(func() INode {
		list := Ul()
		list.Effect(func() {
			list.Body(Each(items, func(i int, v string) INode { return Li().Textf("%d:%s", i, v) }))
		})
		return Div().Body(list, IfFn(Value(false), func() INode { return P().Text("hidden") }), If(Value(true), P().Text("shown")))
	})
	want := `<div><ul><li>0:a</li><li>1:b</li></ul><p>shown</p></div>`
	if got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
}

func TestRenderToStringKeyedList(t *testing.T) {
	items := Signal([]int{1, 2, 3})
	got := RenderToString(func() INode {
		return Ul().Body(EachKeyed(items, func(i int) int { return i },
			func(k int, _ Gettable[int], idx Gettable[int]) INode { return Li().Textf("%d@%d", k, idx.Get()) }))
	})
	if got != `<ul><li>1@0</li><li>2@1</li><li>3@2</li></ul>` {
		t.Fatalf("got %s", got)
	}
}

func TestStringRendererTextareaAndPreKeepALeadingNewline(t *testing.T) {
	got := RenderToString(func() INode { return Div().Body(TextArea().Text("\nline"), Pre().Text("\nx")) })
	want := "<div><textarea>\n\nline</textarea><pre>\n\nx</pre></div>"
	if got != want {
		t.Fatalf("\n got %q\nwant %q", got, want)
	}
}

func TestStateScriptCannotCloseTheScript(t *testing.T) {
	got := StateScript("state", `{"a":"</script><b>"}`)
	if strings.Contains(got[len(`<script type="application/json" id="state">`):len(got)-len("</script>")], "<") {
		t.Fatalf("unescaped <: %s", got)
	}
	if !strings.HasPrefix(got, `<script type="application/json" id="state">`) || !strings.HasSuffix(got, "</script>") {
		t.Fatalf("got %s", got)
	}
}

func TestSvgRendersWithItsOwnCase(t *testing.T) {
	got := RenderToString(func() INode {
		return Svg().Attribute("viewBox", "0 0 1 1").Body(Path().Attribute("d", "M0 0"))
	})
	if got != `<svg viewBox="0 0 1 1"><path d="M0 0"></path></svg>` {
		t.Fatalf("got %s", got)
	}
}

func TestRenderedHooksDoNotRunOnTheServer(t *testing.T) {
	calls := 0
	html := RenderToString(func() INode {
		return Div().Body(Span().Text("x").OnRendered(func() { calls++ }).OnRemoved(func() { calls++ }))
	})
	if html != "<div><span>x</span></div>" {
		t.Fatalf("html: %s", html)
	}
	if calls != 0 {
		t.Fatalf("the hooks ran %d times on the server", calls)
	}
}
