package ssr

import (
	"context"
	"fmt"
	"github.com/deltegui/hx"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRender(t *testing.T) {
	count := hx.Signal(3)
	got := Render(func() hx.INode {
		return hx.Div().Class("card").Body(
			hx.H1().Text("Hello"),
			hx.P().BindText(hx.Computed(func() string { return fmt.Sprintf("count %d", count.Get()) })),
			hx.Input().Placeholder("name").Value("ada"),
			hx.Br(),
		)
	})
	want := `<div class="card"><h1>Hello</h1><p>count 3</p><input value="ada" placeholder="name"/><br/></div>`
	if got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
}

func TestRenderRunsEffectsOnce(t *testing.T) {
	s := hx.Signal(1)
	runs := 0
	Render(func() hx.INode {
		return hx.Div().Effect(func() { runs++; s.Get() })
	})
	if runs != 1 {
		t.Fatalf("runs=%d, want 1", runs)
	}
	s.Set(2)
	if runs != 1 {
		t.Fatalf("an effect of a server render kept running: runs=%d", runs)
	}
}

func TestRenderIgnoresOnMount(t *testing.T) {
	called := false
	Render(func() hx.INode {
		hx.OnMount(func() { called = true })
		return hx.Div()
	})
	if called {
		t.Fatal("OnMount must not run while rendering to a string")
	}
}

func TestOnMountNeverRunsOnTheServer(t *testing.T) {
	called := false
	hx.OnMount(func() { called = true })
	if called {
		t.Fatal("OnMount must not run on a server, even outside of a render")
	}
}

func TestRenderIsSafeToCallFromManyGoroutines(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			name := hx.Signal("x")
			got := Render(func() hx.INode {
				return hx.Span().BindText(name)
			})
			if got != `<span>x</span>` {
				t.Errorf("got %s", got)
			}
		}()
	}
	wg.Wait()
}

func TestRenderOfEachAndConditionals(t *testing.T) {
	items := hx.Signal([]string{"a", "b"})
	got := Render(func() hx.INode {
		list := hx.Ul()
		list.Effect(func() {
			list.Body(hx.Each(items, func(i int, v string) hx.INode { return hx.Li().Textf("%d:%s", i, v) }))
		})
		return hx.Div().Body(list, hx.IfFn(hx.Value(false), func() hx.INode { return hx.P().Text("hidden") }), hx.If(hx.Value(true), hx.P().Text("shown")))
	})
	want := `<div><ul><li>0:a</li><li>1:b</li></ul><p>shown</p></div>`
	if got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
}

func TestRenderKeyedList(t *testing.T) {
	items := hx.Signal([]int{1, 2, 3})
	got := Render(func() hx.INode {
		return hx.Ul().Body(hx.EachKeyed(items, func(i int) int { return i },
			func(k int, _ hx.Gettable[int], idx hx.Gettable[int]) hx.INode {
				return hx.Li().Textf("%d@%d", k, idx.Get())
			}))
	})
	if got != `<ul><li>1@0</li><li>2@1</li><li>3@2</li></ul>` {
		t.Fatalf("got %s", got)
	}
}

func TestStringRendererTextareaAndPreKeepALeadingNewline(t *testing.T) {
	got := Render(func() hx.INode { return hx.Div().Body(hx.TextArea().Text("\nline"), hx.Pre().Text("\nx")) })
	want := "<div><textarea>\n\nline</textarea><pre>\n\nx</pre></div>"
	if got != want {
		t.Fatalf("\n got %q\nwant %q", got, want)
	}
}

func TestStateScriptCannotCloseTheScript(t *testing.T) {
	got := hx.StateScript("state", `{"a":"</script><b>"}`)
	if strings.Contains(got[len(`<script type="application/json" id="state">`):len(got)-len("</script>")], "<") {
		t.Fatalf("unescaped <: %s", got)
	}
	if !strings.HasPrefix(got, `<script type="application/json" id="state">`) || !strings.HasSuffix(got, "</script>") {
		t.Fatalf("got %s", got)
	}
}

func TestSvgRendersWithItsOwnCase(t *testing.T) {
	got := Render(func() hx.INode {
		return hx.Svg().Attribute("viewBox", "0 0 1 1").Body(hx.Path().Attribute("d", "M0 0"))
	})
	if got != `<svg viewBox="0 0 1 1"><path d="M0 0"></path></svg>` {
		t.Fatalf("got %s", got)
	}
}

func TestRenderedHooksDoNotRunOnTheServer(t *testing.T) {
	calls := 0
	html := Render(func() hx.INode {
		return hx.Div().Body(hx.Span().Text("x").OnRendered(func() { calls++ }).OnRemoved(func() { calls++ }))
	})
	if html != "<div><span>x</span></div>" {
		t.Fatalf("html: %s", html)
	}
	if calls != 0 {
		t.Fatalf("the hooks ran %d times on the server", calls)
	}
}

// A signal that many renders read (a counter, the settings of the site) must be
// safe to read from every goroutine: nothing writes to global state.
func TestRenderReadsASharedSignalFromManyGoroutines(t *testing.T) {
	shared := hx.Signal("site")
	double := hx.Computed(func() string { return shared.Get() + shared.Get() })
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got := Render(func() hx.INode {
				return hx.Div().Body(hx.P().BindText(shared), hx.P().BindText(double))
			})
			if got != "<div><p>site</p><p>sitesite</p></div>" {
				t.Errorf("got %s", got)
			}
		}()
	}
	wg.Wait()
}

func TestComputedIsFreshOnTheServer(t *testing.T) {
	n := hx.Signal(1)
	twice := hx.Computed(func() int { return n.Get() * 2 })
	n.Set(5)
	if twice.Get() != 10 {
		t.Fatalf("twice=%d, want 10", twice.Get())
	}
}

func BenchmarkRenderParallel(b *testing.B) {
	rows := make([]string, 50)
	for i := range rows {
		rows[i] = "row"
	}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Render(func() hx.INode {
				list := hx.Ul()
				items := make([]hx.INode, len(rows))
				for i, r := range rows {
					items[i] = hx.Li().Class("item").Text(r).Attribute("data-i", "x")
				}
				return hx.Div().Class("card").Body(hx.H1().Text("Title"), list.Body(items...))
			})
		}
	})
}

func TestResourceIsNotLoadedWhileRenderingOnAServer(t *testing.T) {
	called := make(chan struct{}, 1)
	Render(func() hx.INode {
		hx.Resource(func(ctx context.Context) (int, error) { called <- struct{}{}; return 1, nil })
		return hx.Div()
	})
	select {
	case <-called:
		t.Fatal("a server render must not load resources")
	case <-time.After(30 * time.Millisecond):
	}
}

func TestCheckedRendersAndBinds(t *testing.T) {
	on := hx.Signal(true)
	got := Render(func() hx.INode {
		return hx.Div().Body(
			hx.Input().Type("checkbox").Attribute("name", "a").BindChecked(on),
			hx.Input().Type("checkbox").Attribute("name", "b"),
			hx.Input().Type("checkbox").Attribute("name", "c").Checked(true),
		)
	})
	want := `<div><input checked="" name="a" type="checkbox"/><input name="b" type="checkbox"/><input checked="" name="c" type="checkbox"/></div>`
	if got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
}
