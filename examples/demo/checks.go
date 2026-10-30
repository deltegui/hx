//go:build js && wasm

package main

import (
	"fmt"
	"strings"
	"syscall/js"

	"github.com/deltegui/hx"
	"github.com/deltegui/hx/wasm"
	"honnef.co/go/js/dom/v2"
)

// autoRunChecks is set when the page is opened with #checks-auto.
var autoRunChecks bool

// sandbox is a throwaway mount point in the real DOM.
type sandbox struct {
	el   dom.Element
	root *wasm.VNode
}

func newSandbox() *sandbox {
	doc := dom.GetWindow().Document()
	el := doc.CreateElement("div")
	doc.GetElementByID("sandbox").AppendChild(el)
	return &sandbox{el: el, root: wasm.New(el)}
}

func (s *sandbox) html() string { return s.el.InnerHTML() }

func (s *sandbox) click(selector string) {
	s.el.QuerySelector(selector).Underlying().Call("click")
}

func (s *sandbox) close() { s.el.ParentNode().RemoveChild(s.el) }

func (s *sandbox) expect(want string) error {
	if got := s.html(); got != want {
		return fmt.Errorf("html mismatch\n  got:  %s\n  want: %s", got, want)
	}
	return nil
}

func (s *sandbox) expectContains(sub string) error {
	if got := s.html(); !strings.Contains(got, sub) {
		return fmt.Errorf("html does not contain %q\n  got: %s", sub, got)
	}
	return nil
}

func (s *sandbox) expectMissing(sub string) error {
	if got := s.html(); strings.Contains(got, sub) {
		return fmt.Errorf("html contains %q\n  got: %s", sub, got)
	}
	return nil
}

// countMutations observes the childList mutations in the sandbox. The returned
// function gives how many were delivered since the last call.
func (s *sandbox) countMutations() func() int {
	count := 0
	observer := js.Global().Get("MutationObserver").New(js.FuncOf(func(_ js.Value, args []js.Value) any {
		count += args[0].Length()
		return nil
	}))
	observer.Call("observe", s.el.Underlying(), map[string]any{"childList": true, "subtree": true})
	return func() int {
		n := count
		count = 0
		return n
	}
}

func eq[T comparable](what string, got, want T) error {
	if got != want {
		return fmt.Errorf("%s: got %v, want %v", what, got, want)
	}
	return nil
}

func firstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

type check struct {
	name string
	run  func(sb *sandbox) error
}

func allChecks() []check {
	return []check{
		{"renders a node", func(sb *sandbox) error {
			sb.root.Body(hx.Div().Class("a").Text("hi"))
			settle()
			return sb.expect(`<div class="a">hi</div>`)
		}},

		{"text change keeps the children", func(sb *sandbox) error {
			d := hx.Div().Text("one")
			d.Body(hx.Span().Text("kid"))
			sb.root.Body(d)
			settle()
			if err := sb.expect(`<div>one<span>kid</span></div>`); err != nil {
				return err
			}
			d.Text("two")
			settle()
			return sb.expect(`<div>two<span>kid</span></div>`)
		}},

		{"BindText on a node with children", func(sb *sandbox) error {
			label := hx.Signal("a")
			d := hx.Div()
			d.BindText(label)
			d.Body(hx.Span().Text("kid"))
			sb.root.Body(d)
			settle()
			label.Set("b")
			settle()
			return sb.expect(`<div>b<span>kid</span></div>`)
		}},

		{"Noop keeps its position between siblings", func(sb *sandbox) error {
			mid := hx.Noop()
			mid.Body(hx.Li().Text("x"))
			sb.root.Body(hx.Ul().Body(hx.Li().Text("first"), mid, hx.Li().Text("last")))
			settle()
			if err := sb.expect(`<ul><li>first</li><li>x</li><li>last</li></ul>`); err != nil {
				return err
			}
			mid.BodyList([]hx.INode{hx.Li().Text("y"), hx.Li().Text("z")})
			settle()
			if err := sb.expect(`<ul><li>first</li><li>y</li><li>z</li><li>last</li></ul>`); err != nil {
				return err
			}
			mid.Body()
			settle()
			return sb.expect(`<ul><li>first</li><li>last</li></ul>`)
		}},

		{"children of a removed Noop leave the DOM", func(sb *sandbox) error {
			mid := hx.Noop()
			mid.Body(hx.Li().Text("x"), hx.Li().Text("y"))
			parent := hx.Ul().Body(mid)
			sb.root.Body(parent)
			settle()
			parent.Body()
			settle()
			return sb.expect(`<ul></ul>`)
		}},

		{"Each: removed items leave the DOM", func(sb *sandbox) error {
			items := hx.Signal([]string{"a", "b", "c"})
			ul := hx.Ul()
			ul.Effect(func() {
				ul.Body(hx.Each(items, func(i int, v string) hx.INode { return hx.Li().Text(v) }))
			})
			sb.root.Body(ul)
			settle()
			if err := sb.expect(`<ul><li>a</li><li>b</li><li>c</li></ul>`); err != nil {
				return err
			}
			items.Set([]string{"a"})
			settle()
			return sb.expect(`<ul><li>a</li></ul>`)
		}},

		{"EachMapSorted renders in key order", func(sb *sandbox) error {
			m := hx.Signal(map[string]int{"b": 1, "a": 2, "c": 3})
			d := hx.Div()
			d.Effect(func() {
				d.Body(hx.EachMapSorted(m, func(k string, v int) hx.INode {
					return hx.P().Text(fmt.Sprintf("%s%d", k, v))
				}))
			})
			sb.root.Body(d)
			settle()
			return sb.expect(`<div><p>a2</p><p>b1</p><p>c3</p></div>`)
		}},

		{"parent removes a child while another child changes", func(sb *sandbox) error {
			// Both nodes are marked in the same frame. The renderer must sync from the parent.
			for i := 0; i < 25; i++ {
				c := hx.Span().Text("x")
				d := hx.Span().Text("gone")
				p := hx.Div().Body(c, d)
				sb.root.Body(p)
				settle()
				c.Text("y")
				p.Body(c)
				settle()
				if err := sb.expect(`<div><span>y</span></div>`); err != nil {
					return fmt.Errorf("round %d: %w", i, err)
				}
			}
			return nil
		}},

		{"Body with a reused node does not duplicate it", func(sb *sandbox) error {
			a, b := hx.Span().Text("a"), hx.Span().Text("b")
			d := hx.Div().Body(a, b)
			sb.root.Body(d)
			settle()
			d.BodyList([]hx.INode{b, a})
			settle()
			if err := sb.expect(`<div><span>b</span><span>a</span></div>`); err != nil {
				return err
			}
			d.BodyList([]hx.INode{a})
			settle()
			return sb.expect(`<div><span>a</span></div>`)
		}},

		{"class removed and added in the same tick stays", func(sb *sandbox) error {
			d := hx.Div().Class("hl")
			sb.root.Body(d)
			settle()
			d.RemoveClass("hl")
			d.Class("hl")
			settle()
			if err := sb.expectContains(`class="hl"`); err != nil {
				return err
			}
			d.RemoveClass("hl")
			settle()
			return sb.expectMissing("hl")
		}},

		{"attribute with empty value, remove and add again", func(sb *sandbox) error {
			b := hx.Button().Attribute("disabled", "")
			sb.root.Body(b)
			settle()
			if err := sb.expectContains(`disabled=""`); err != nil {
				return err
			}
			b.RemoveAttribute("disabled")
			settle()
			if err := sb.expectMissing("disabled"); err != nil {
				return err
			}
			b.Attribute("disabled", "")
			settle()
			return sb.expectContains(`disabled=""`)
		}},

		{"attribute that goes back to its rendered value", func(sb *sandbox) error {
			d := hx.Div().Attribute("title", "1")
			sb.root.Body(d)
			settle()
			d.Attribute("title", "2")
			d.Attribute("title", "1")
			settle()
			return sb.expectContains(`title="1"`)
		}},

		{"style add, remove and add again", func(sb *sandbox) error {
			d := hx.Div().Style("color", "red")
			sb.root.Body(d)
			settle()
			if err := sb.expectContains("color:red"); err != nil {
				return err
			}
			d.RemoveStyle("color")
			settle()
			if err := sb.expectMissing("style="); err != nil {
				return err
			}
			d.Style("color", "red")
			settle()
			return sb.expectContains("color:red")
		}},

		{"id changes", func(sb *sandbox) error {
			d := hx.Div().Id("one")
			sb.root.Body(d)
			settle()
			d.Id("two")
			d.Id("one")
			settle()
			return sb.expectContains(`id="one"`)
		}},

		{"On replaces the handler: one call per click", func(sb *sandbox) error {
			count := 0
			b := hx.Button()
			b.OnClick(func(hx.EventContext) { count += 100 })
			sb.root.Body(b)
			settle()
			b.OnClick(func(hx.EventContext) { count += 10 })
			b.OnClick(func(hx.EventContext) { count++ })
			settle()
			b.OnClick(func(hx.EventContext) { count++ })
			settle()
			sb.click("button")
			settle()
			return eq("count after one click", count, 1)
		}},

		{"listeners survive a reorder", func(sb *sandbox) error {
			count := 0
			a := hx.Button().Text("a")
			a.OnClick(func(hx.EventContext) { count++ })
			b := hx.Button().Text("b")
			d := hx.Div().Body(a, b)
			sb.root.Body(d)
			settle()
			d.BodyList([]hx.INode{b, a})
			settle()
			if err := sb.expect(`<div><button>b</button><button>a</button></div>`); err != nil {
				return err
			}
			sb.click("button:last-child")
			settle()
			return eq("count", count, 1)
		}},

		{"removed node: listener released, re-added: listener works", func(sb *sandbox) error {
			count := 0
			b := hx.Button().Text("b")
			b.OnClick(func(hx.EventContext) { count++ })
			sb.root.Body(b)
			settle()
			sb.root.Body()
			settle()
			sb.root.Body(b)
			settle()
			sb.click("button")
			settle()
			return eq("count", count, 1)
		}},

		{"node Effect stops when the node is removed and returns with it", func(sb *sandbox) error {
			s := hx.Signal(0)
			runs := 0
			d := hx.Div().Effect(func() { runs++; s.Get() })
			sb.root.Body(d)
			settle()
			sb.root.Body()
			settle()
			before := runs
			s.Set(1)
			if err := eq("runs while removed", runs, before); err != nil {
				return err
			}
			sb.root.Body(d)
			settle()
			if err := eq("runs after re-adding", runs, before+1); err != nil {
				return err
			}
			s.Set(2)
			return eq("runs when live again", runs, before+2)
		}},

		{"If discards the unused child", func(sb *sandbox) error {
			s := hx.Signal(0)
			runs := 0
			dispose := hx.CreateRoot(func(dispose func()) func() {
				hx.EffectFunc(func() {
					hx.If(hx.Value(false), hx.Div().Effect(func() { runs++; s.Get() }))
				})
				return dispose
			})
			defer dispose()
			before := runs
			s.Set(1)
			return eq("runs after the child was discarded", runs, before)
		}},

		{"EachKeyed keeps the DOM elements of existing keys", func(sb *sandbox) error {
			type row struct {
				id   int
				name string
			}
			items := hx.Signal([]row{{1, "a"}, {2, "b"}, {3, "c"}})
			sb.root.Body(hx.Ul().Body(hx.EachKeyed(items,
				func(r row) int { return r.id },
				func(id int, item hx.Gettable[row], _ hx.Gettable[int]) hx.INode {
					return hx.Li().BindText(hx.Computed(func() string { return item.Get().name }))
				})))
			settle()
			for _, li := range sb.el.QuerySelectorAll("li") {
				li.Underlying().Set("marker", li.TextContent())
			}
			items.Set([]row{{3, "c"}, {1, "a"}, {4, "d"}, {2, "b"}})
			settle()
			if err := sb.expect(`<ul><li>c</li><li>a</li><li>d</li><li>b</li></ul>`); err != nil {
				return err
			}
			var kept []string
			for _, li := range sb.el.QuerySelectorAll("li") {
				marker := li.Underlying().Get("marker")
				if marker.IsUndefined() {
					kept = append(kept, "new")
				} else {
					kept = append(kept, marker.String())
				}
			}
			return eq("elements", strings.Join(kept, ","), "c,a,new,b")
		}},

		{"EachKeyed: a reused row keeps its input value and focus", func(sb *sandbox) error {
			type row struct{ id int }
			items := hx.Signal([]row{{1}, {2}, {3}})
			sb.root.Body(hx.Ul().Body(hx.EachKeyed(items,
				func(r row) int { return r.id },
				func(id int, _ hx.Gettable[row], _ hx.Gettable[int]) hx.INode {
					return hx.Li().Body(hx.Input().Id(fmt.Sprint("in", id)))
				})))
			settle()
			input := sb.el.QuerySelector("#in2").Underlying()
			input.Set("value", "typed by the user")
			input.Call("focus")
			// add a row at the start and remove the first one
			items.Set([]row{{4}, {2}, {3}})
			settle()
			same := sb.el.QuerySelector("#in2").Underlying()
			return firstError(
				eq("value", same.Get("value").String(), "typed by the user"),
				eq("is the same element", same.Equal(input), true),
				eq("has focus", js.Global().Get("document").Get("activeElement").Equal(input), true),
			)
		}},

		{"EachKeyed: moving one row moves one element", func(sb *sandbox) error {
			type row struct{ name string }
			items := hx.Signal([]row{{"a"}, {"b"}, {"c"}, {"d"}, {"e"}})
			sb.root.Body(hx.Ul().Body(hx.EachKeyed(items,
				func(r row) string { return r.name },
				func(k string, _ hx.Gettable[row], _ hx.Gettable[int]) hx.INode { return hx.Li().Text(k) })))
			settle()

			mutations := sb.countMutations()

			items.Set([]row{{"e"}, {"a"}, {"b"}, {"c"}, {"d"}}) // the last goes first
			settle()
			if err := sb.expect(`<ul><li>e</li><li>a</li><li>b</li><li>c</li><li>d</li></ul>`); err != nil {
				return err
			}
			records := mutations()
			if records < 1 || records > 2 { // moving a node is a removal and an addition
				return fmt.Errorf("%d DOM mutation records, want 1 or 2", records)
			}

			items.Set([]row{{"e"}, {"a"}, {"b"}, {"c"}, {"d"}}) // same order again
			settle()
			if n := mutations(); n != 0 {
				return fmt.Errorf("%d DOM mutations when nothing moved", n)
			}
			return nil
		}},

		{"EachKeyed: new values update the rows without touching the DOM structure", func(sb *sandbox) error {
			type row struct {
				id   int
				name string
			}
			items := hx.Signal([]row{{1, "a"}, {2, "b"}})
			sb.root.Body(hx.Ul().Body(hx.EachKeyed(items,
				func(r row) int { return r.id },
				func(id int, item hx.Gettable[row], _ hx.Gettable[int]) hx.INode {
					return hx.Li().BindText(hx.Computed(func() string { return item.Get().name }))
				})))
			settle()
			mutations := sb.countMutations()
			items.Set([]row{{1, "A"}, {2, "B"}})
			settle()
			if err := sb.expect(`<ul><li>A</li><li>B</li></ul>`); err != nil {
				return err
			}
			return eq("childList mutations", mutations(), 0)
		}},

		{"IfFn and ShowFn build only the path they use", func(sb *sandbox) error {
			built := 0
			build := func() hx.INode { built++; return hx.Span().Text("x") }
			shown := hx.Signal(false)
			box := hx.Div()
			box.Effect(func() {
				box.Body(hx.ShowFn(shown, build, func() hx.INode { return hx.P().Text("off") }))
			})
			sb.root.Body(box)
			settle()
			if err := firstError(eq("built while hidden", built, 0), sb.expect(`<div><p>off</p></div>`)); err != nil {
				return err
			}
			shown.Set(true)
			settle()
			return firstError(eq("built when shown", built, 1), sb.expect(`<div><span>x</span></div>`))
		}},

		{"CreateRoot: dispose stops effects and runs OnCleanup", func(sb *sandbox) error {
			s := hx.Signal(0)
			runs, cleanups := 0, 0
			dispose := hx.CreateRoot(func(dispose func()) func() {
				hx.EffectFunc(func() { runs++; s.Get() })
				hx.OnCleanup(func() { cleanups++ })
				return dispose
			})
			s.Set(1)
			if err := eq("runs while alive", runs, 2); err != nil {
				return err
			}
			dispose()
			dispose()
			s.Set(2)
			return firstError(eq("runs after dispose", runs, 2), eq("cleanups", cleanups, 1))
		}},

		{"inputs keep their specific dom type", func(sb *sandbox) error {
			in := hx.Input()
			sb.root.Body(in)
			settle()
			if _, ok := wasm.Underlying(in).(*dom.HTMLInputElement); !ok {
				return fmt.Errorf("Underlying() is %T, want *dom.HTMLInputElement", wasm.Underlying(in))
			}
			return nil
		}},

		{"SVG elements use the SVG namespace", func(sb *sandbox) error {
			sb.root.Body(hx.Svg().Body(hx.Path()))
			settle()
			ns := sb.el.QuerySelector("path").Underlying().Get("namespaceURI").String()
			return eq("namespace", ns, "http://www.w3.org/2000/svg")
		}},

		{"textarea BindValue sets text and value", func(sb *sandbox) error {
			s := hx.Signal("hello")
			t := hx.TextArea()
			t.BindValue(s)
			sb.root.Body(t)
			settle()
			if err := eq("value", sb.el.QuerySelector("textarea").Underlying().Get("value").String(), "hello"); err != nil {
				return err
			}
			s.Set("bye")
			settle()
			return eq("value", sb.el.QuerySelector("textarea").Underlying().Get("value").String(), "bye")
		}},

		{"select BindValue picks its option when it is created, and when the options change", func(sb *sandbox) error {
			s := hx.Signal("b")
			opts := hx.Signal([]string{"a", "b", "c"})
			sel := hx.Select().BindValue(s)
			sel.Effect(func() {
				nodes := []hx.INode{}
				for _, o := range opts.Get() {
					nodes = append(nodes, hx.Option().Value(o).Text(o))
				}
				sel.BodyList(nodes)
			})
			sb.root.Body(sel)
			settle()
			el := sb.el.QuerySelector("select").Underlying()
			if err := eq("value at creation", el.Get("value").String(), "b"); err != nil {
				return err
			}
			s.Set("c")
			settle()
			if err := eq("value after the signal changed", el.Get("value").String(), "c"); err != nil {
				return err
			}
			opts.Set([]string{"z", "c", "y"})
			settle()
			return eq("value after the options changed", el.Get("value").String(), "c")
		}},

		{"OnRendered runs once the element is in the document", func(sb *sandbox) error {
			calls, connected := 0, false
			canvas := hx.Canvas()
			canvas.OnRendered(func() {
				calls++
				connected = wasm.Underlying(canvas) != nil && wasm.Underlying(canvas).Underlying().Get("isConnected").Bool()
			})
			sb.root.Body(hx.Div().Body(canvas))
			if err := eq("calls before the render", calls, 0); err != nil {
				return err
			}
			settle()
			canvas.Attribute("width", "10")
			settle()
			return firstError(eq("calls", calls, 1), eq("connected", connected, true))
		}},

		{"OnRemoved runs when the node leaves, OnRendered again when it comes back", func(sb *sandbox) error {
			rendered, removed := 0, 0
			child := hx.Span().Text("kid")
			child.OnRendered(func() { rendered++ }).OnRemoved(func() { removed++ })
			parent := hx.Div()
			sb.root.Body(parent)
			parent.Body(child)
			settle()
			if err := firstError(eq("rendered", rendered, 1), eq("removed", removed, 0)); err != nil {
				return err
			}
			parent.Body()
			settle()
			if err := firstError(eq("rendered", rendered, 1), eq("removed", removed, 1)); err != nil {
				return err
			}
			parent.Body(child)
			settle()
			return firstError(eq("rendered", rendered, 2), eq("removed", removed, 1))
		}},

		{"BindValue resets an input whose DOM value drifted", func(sb *sandbox) error {
			s := hx.Signal("")
			in := hx.Input().BindValue(s)
			sb.root.Body(in)
			settle()
			el := sb.el.QuerySelector("input").Underlying()
			el.Set("value", "typed without an input event")
			s.Set("")
			settle()
			return eq("value", el.Get("value").String(), "")
		}},

		{"BindOnInput works with the event target value", func(sb *sandbox) error {
			s := hx.Signal("")
			in := hx.Input()
			in.BindOnInput(s)
			sb.root.Body(in)
			settle()
			input := sb.el.QuerySelector("input").Underlying()
			input.Set("value", "typed")
			input.Call("dispatchEvent", js.Global().Get("Event").New("input"))
			settle()
			return eq("signal", hx.UntrackGet[string](s), "typed")
		}},
	}
}

func checksPage() hx.INode {
	type result struct {
		name   string
		ok     bool
		detail string
	}
	results := hx.Signal([]result{})
	summary := hx.Signal("not run yet")
	running := false

	run := func() {
		if running {
			return
		}
		running = true
		results.Set([]result{})
		summary.Set("running...")
		go func() {
			defer func() { running = false }()
			passed := 0
			list := allChecks()
			for _, c := range list {
				res := result{name: c.name, ok: true}
				func() {
					sb := newSandbox()
					defer sb.close()
					defer func() {
						if r := recover(); r != nil {
							res.ok = false
							res.detail = fmt.Sprintf("panic: %v", r)
						}
					}()
					if err := c.run(sb); err != nil {
						res.ok = false
						res.detail = err.Error()
					}
				}()
				if res.ok {
					passed++
				}
				results.Update(func(r []result) []result { return append(r[:len(r):len(r)], res) })
			}
			text := fmt.Sprintf("%d/%d checks passed", passed, len(list))
			summary.Set(text)
			dom.GetWindow().Document().QuerySelector("body").SetAttribute("data-checks", text)
		}()
	}

	ul := hx.Ul().Class("checks")
	ul.Effect(func() {
		ul.Body(hx.Each(results, func(i int, r result) hx.INode {
			status := hx.Strong().Class("ok").Text("PASS ")
			if !r.ok {
				status = hx.Strong().Class("bad").Text("FAIL ")
			}
			parts := []hx.INode{status, hx.Span().Text(r.name)}
			if r.detail != "" {
				parts = append(parts, hx.Small().Text(r.detail))
			}
			return hx.Li().Body(parts...)
		}))
	})

	page := wideCard("Checks against the real DOM",
		note("Each check builds nodes in a scratch element, waits for the renderer and compares the resulting HTML or behaviour."),
		row(
			btn("Run checks", run),
			live(summary.Get),
		),
		ul,
	)
	if autoRunChecks {
		go func() {
			settle()
			run()
		}()
	}
	return hx.Div().Class("content").Body(page)
}
