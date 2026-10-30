package wasm

import (
	"strings"
	"testing"

	"github.com/deltegui/hx"
)

func render(node hx.INode) string {
	r := &StringRenderer{}
	NewWithoutMount("DIV", r) // make sure the constructor works without DOM
	r.Render(node.(*VNode))
	return r.String()
}

func TestStringRendererAttributesAndStyles(t *testing.T) {
	got := render(hx.Div().Id("x").Class("b", "a").Style("color", "red").Attribute("title", "hi").Text("a<b"))
	want := `<div id="x" class="a b" style="color:red;" title="hi">a&lt;b</div>`
	if got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
}

func TestStringRendererVoidAndInput(t *testing.T) {
	got := render(hx.Div().Body(hx.Input().Type("text").Value("v"), hx.Br()))
	want := `<div><input value="v" type="text"/><br/></div>`
	if got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
}

func TestStringRendererNoopIsTransparent(t *testing.T) {
	got := render(hx.Ul().Body(hx.Noop().Body(hx.Li().Text("1"), hx.Li().Text("2"))))
	want := `<ul><li>1</li><li>2</li></ul>`
	if got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
}

func TestStringRendererSkipsRemoved(t *testing.T) {
	got := render(hx.Div().Class("a", "b").RemoveClass("a").Style("x", "1").RemoveStyle("x").Attribute("k", "v").RemoveAttribute("k"))
	want := `<div class="b"></div>`
	if got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
}

func TestBodyReplacedDropsOldChildren(t *testing.T) {
	d := hx.Div()
	a, b := hx.Span().Text("a"), hx.Span().Text("b")
	d.Body(a, b)
	d.Body(b)
	if got := render(d); got != `<div><span>b</span></div>` {
		t.Fatalf("got %s", got)
	}
	if len(d.(*VNode).children) != 2 { // a (deleted, pending removal) and b
		t.Fatalf("children=%d", len(d.(*VNode).children))
	}
}

func TestClassRemovedAndAddedAgain(t *testing.T) {
	d := hx.Div().Class("a")
	d.(*VNode).classes["a"] = unchanged // as if rendered
	d.RemoveClass("a").Class("a")
	if !d.HaveClass("a") {
		t.Fatal("class lost")
	}
}

func TestAttributeEdgeCases(t *testing.T) {
	d := hx.Div().Attribute("disabled", "")
	if !d.HaveAttribute("disabled", "") {
		t.Fatal("empty attribute ignored")
	}
	v := d.(*VNode)
	e := v.attributes["disabled"]
	e.tick()
	v.attributes["disabled"] = e // rendered

	d.RemoveAttribute("disabled")
	if d.GetAttribute("disabled") != "" || d.HaveAttribute("disabled", "") {
		t.Fatal("still present")
	}
	d.Attribute("disabled", "")
	if !d.HaveAttribute("disabled", "") {
		t.Fatal("re-adding a removed attribute with the same value was ignored")
	}

	d.Attribute("title", "1")
	e = v.attributes["title"]
	e.tick()
	v.attributes["title"] = e
	d.Attribute("title", "2").Attribute("title", "1")
	if v.attributes["title"].status != unchanged {
		t.Fatal("going back to the rendered value should not be a change")
	}
}

func TestStyleRemovedAndAddedAgain(t *testing.T) {
	d := hx.Div().Style("color", "red")
	v := d.(*VNode)
	e := v.styles["color"]
	e.tick()
	v.styles["color"] = e
	d.RemoveStyle("color")
	if d.GetStyle("color") != "" {
		t.Fatal("still present")
	}
	d.Style("color", "red")
	if d.GetStyle("color") != "red" {
		t.Fatal("re-adding was ignored")
	}
}

func TestOnReplacesHandler(t *testing.T) {
	d := hx.Div()
	called := ""
	d.OnClick(func(hx.EventContext) { called = "first" })
	d.OnClick(func(hx.EventContext) { called = "second" })
	d.(*VNode).eventListeners[hx.EventClick].handler(hx.EventContext{})
	if called != "second" {
		t.Fatalf("called=%s", called)
	}
}

func TestBindingStopsWhenDeactivated(t *testing.T) {
	s := hx.Signal("a")
	node := hx.Span().BindText(s)
	if got := render(node); got != `<span>a</span>` {
		t.Fatalf("got %s", got)
	}
	s.Set("b")
	if got := render(node); got != `<span>b</span>` {
		t.Fatalf("got %s", got)
	}

	node.(*VNode).deactivate()
	s.Set("c")
	if got := render(node); got != `<span>b</span>` {
		t.Fatalf("binding still active after deactivate: %s", got)
	}
	if s.Subscribers() != 0 {
		t.Fatalf("subscribers=%d", s.Subscribers())
	}

	node.(*VNode).reactivate()
	if got := render(node); got != `<span>c</span>` {
		t.Fatalf("binding not restored with the current value: %s", got)
	}
	s.Set("d")
	if got := render(node); got != `<span>d</span>` {
		t.Fatalf("binding not live after reactivate: %s", got)
	}
}

func TestDeactivateReachesDescendants(t *testing.T) {
	s := hx.Signal("a")
	child := hx.Span().BindText(s)
	parent := hx.Div().Body(hx.Noop().Body(child))
	parent.(*VNode).deactivate()
	if s.Subscribers() != 0 {
		t.Fatalf("descendant binding still subscribed")
	}
	parent.(*VNode).reactivate()
	if s.Subscribers() != 1 {
		t.Fatalf("descendant binding not restored")
	}
}

func TestBindingInRootIsDisposedByRoot(t *testing.T) {
	s := hx.Signal("a")
	dispose := hx.CreateRoot(func(dispose func()) func() {
		hx.Span().BindText(s)
		return dispose
	})
	dispose()
	if s.Subscribers() != 0 {
		t.Fatalf("subscribers=%d", s.Subscribers())
	}
}

func TestNodeEffectFollowsTheNode(t *testing.T) {
	s := hx.Signal(1)
	runs := 0
	node := hx.Div().Effect(func() {
		runs++
		s.Get()
	})
	s.Set(2)
	if runs != 2 {
		t.Fatalf("runs=%d", runs)
	}

	node.(*VNode).deactivate()
	s.Set(3)
	if runs != 2 {
		t.Fatalf("effect ran after deactivate, runs=%d", runs)
	}

	node.(*VNode).reactivate()
	if runs != 3 {
		t.Fatalf("effect did not run again on reactivate, runs=%d", runs)
	}
	s.Set(4)
	if runs != 4 {
		t.Fatalf("effect not live after reactivate, runs=%d", runs)
	}
}

func TestNodeEffectCanRebuildItsOwnBody(t *testing.T) {
	items := hx.Signal([]string{"a"})
	list := hx.Ul()
	list.Effect(func() {
		list.Body(hx.Each(items, func(i int, v string) hx.INode { return hx.Li().Text(v) }))
	})
	items.Set([]string{"a", "b"})
	if got := render(list); got != `<ul><li>a</li><li>b</li></ul>` {
		t.Fatalf("got %s", got)
	}
}

func TestInputInheritsEffect(t *testing.T) {
	s := hx.Signal(0)
	runs := 0
	hx.Input().Effect(func() { runs++; s.Get() })
	s.Set(1)
	if runs != 2 {
		t.Fatalf("runs=%d", runs)
	}
}

func TestIfDiscardsUnusedChild(t *testing.T) {
	cond := hx.Signal(false)
	text := hx.Signal("a")
	var got hx.INode
	hx.EffectFunc(func() {
		got = hx.If(cond, hx.Span().BindText(text))
	})
	if got != nil {
		t.Fatal("expected nil when the condition is false")
	}
	if text.Subscribers() != 0 {
		t.Fatalf("discarded child is still subscribed: %d", text.Subscribers())
	}
}

func TestIfKeepsUsedChildActive(t *testing.T) {
	text := hx.Signal("a")
	var got hx.INode
	hx.EffectFunc(func() {
		got = hx.If(hx.Value(true), hx.Span().BindText(text))
	})
	if got == nil || text.Subscribers() != 1 {
		t.Fatalf("used child must stay active, subscribers=%d", text.Subscribers())
	}
}

func TestShowDiscardsOnlyTheOtherPath(t *testing.T) {
	a, b := hx.Signal("a"), hx.Signal("b")
	var got hx.INode
	hx.EffectFunc(func() {
		got = hx.Show(hx.Value(true), hx.Span().BindText(a), hx.Span().BindText(b))
	})
	if a.Subscribers() != 1 || b.Subscribers() != 0 {
		t.Fatalf("a=%d b=%d", a.Subscribers(), b.Subscribers())
	}
	if got == nil {
		t.Fatal("nil result")
	}
}

func TestDiscardedNodeComesBack(t *testing.T) {
	text := hx.Signal("a")
	node := hx.Span().BindText(text)
	hx.If(hx.Value(false), node)
	text.Set("b")
	node.(*VNode).reactivate()
	if got := render(node); got != `<span>b</span>` {
		t.Fatalf("got %s", got)
	}
	if text.Subscribers() != 1 {
		t.Fatalf("subscribers=%d", text.Subscribers())
	}
}

func TestFluentChainingInAnyOrder(t *testing.T) {
	// Input-specific and generic methods can be mixed, because everything returns *VNode.
	got := render(hx.Input().Placeholder("p").Class("c").Type("text").Id("x").Style("a", "b").Value("v").Attribute("k", "v"))
	want := `<input id="x" class="c" style="a:b;" value="v" k="v" placeholder="p" type="text"/>`
	if got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
}

func TestFluentHelpers(t *testing.T) {
	got := render(hx.A().Href("/x").ClassIf("on", true).ClassIf("off", false).Textf("%d items", 3))
	want := `<a class="on" href="/x">3 items</a>`
	if got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
	if got := render(hx.El("details").Body(hx.El("summary").Text("s"))); got != `<details><summary>s</summary></details>` {
		t.Fatalf("got %s", got)
	}
}

// hx.A function that returns hx.INode can still be chained.
func fluentCard() hx.INode { return hx.Div().Class("card") }

func TestINodeReturnCanBeChained(t *testing.T) {
	if got := render(fluentCard().Class("wide").Text("x")); got != `<div class="card wide">x</div>` {
		t.Fatalf("got %s", got)
	}
}

func TestTextAreaBindValueSetsText(t *testing.T) {
	s := hx.Signal("hi")
	if got := render(hx.TextArea().BindValue(s)); got != `<textarea>hi</textarea>` {
		t.Fatalf("got %s", got)
	}
}

func TestIfFnBuildsOnlyWhenTrue(t *testing.T) {
	built := 0
	build := func() hx.INode { built++; return hx.Span().Text("x") }

	if hx.IfFn(hx.Value(false), build) != nil || built != 0 {
		t.Fatalf("built=%d when the condition is false", built)
	}
	if got := hx.IfFn(hx.Value(true), build); got == nil || built != 1 {
		t.Fatalf("built=%d when the condition is true", built)
	}
}

func TestShowFnBuildsOnlyTheUsedPath(t *testing.T) {
	a, b := 0, 0
	buildA := func() hx.INode { a++; return hx.Span().Text("a") }
	buildB := func() hx.INode { b++; return hx.Span().Text("b") }

	if got := render(hx.ShowFn(hx.Value(true), buildA, buildB)); got != `<span>a</span>` || a != 1 || b != 0 {
		t.Fatalf("got %s a=%d b=%d", got, a, b)
	}
	if got := render(hx.ShowFn(hx.Value(false), buildA, buildB)); got != `<span>b</span>` || a != 1 || b != 1 {
		t.Fatalf("got %s a=%d b=%d", got, a, b)
	}
	if hx.ShowFn(hx.Value(false), buildA, nil) != nil {
		t.Fatal("a nil else path must give nil")
	}
}

func TestIfFnBindingStaysActive(t *testing.T) {
	text := hx.Signal("a")
	var got hx.INode
	hx.EffectFunc(func() {
		got = hx.IfFn(hx.Value(true), func() hx.INode { return hx.Span().BindText(text) })
	})
	if got == nil || text.Subscribers() != 1 {
		t.Fatalf("subscribers=%d", text.Subscribers())
	}
}

func TestBindOnCheckedRegistersAChangeHandler(t *testing.T) {
	box := hx.Input().Type("checkbox").BindOnChecked(hx.Signal(false))
	if _, ok := box.(*VNode).eventListeners[hx.EventChange]; !ok {
		t.Fatal("BindOnChecked must listen to change")
	}
}

func TestAClassWithASpaceIsIgnored(t *testing.T) {
	d := hx.Div().Class("a b", "c")
	if got := render(d); got != `<div class="c"></div>` {
		t.Fatalf("got %s", got)
	}
	if d.HaveClass("a b") {
		t.Fatal("the class with a space must not be kept")
	}
}

// A binding to a signal and a handler of the user on the same event must both
// run. Before, the second one replaced the first and the signal stopped following
// the element without any warning.
func TestBindingAndHandlerOfTheSameEventBothRun(t *testing.T) {
	var order []string
	sel := hx.Select().(*VNode)
	sel.onBound(hx.EventChange, func(hx.EventContext) { order = append(order, "binding") })
	sel.OnChange(func(hx.EventContext) { order = append(order, "handler") })
	sel.eventListeners[hx.EventChange].fire(hx.EventContext{})
	if got := strings.Join(order, ","); got != "binding,handler" {
		t.Fatalf("ran %q, want the binding first and then the handler", got)
	}

	// the other order of calls gives the same result
	order = nil
	other := hx.Select().(*VNode)
	other.OnChange(func(hx.EventContext) { order = append(order, "handler") })
	other.onBound(hx.EventChange, func(hx.EventContext) { order = append(order, "binding") })
	other.eventListeners[hx.EventChange].fire(hx.EventContext{})
	if got := strings.Join(order, ","); got != "binding,handler" {
		t.Fatalf("other order of calls: ran %q", got)
	}
}

func TestEachSlotOfAListenerIsReplacedOnItsOwn(t *testing.T) {
	var order []string
	d := hx.Input().(*VNode)
	d.onBound(hx.EventInput, func(hx.EventContext) { order = append(order, "binding 1") })
	d.onBound(hx.EventInput, func(hx.EventContext) { order = append(order, "binding 2") })
	d.OnInput(func(hx.EventContext) { order = append(order, "handler 1") })
	d.OnInput(func(hx.EventContext) { order = append(order, "handler 2") })
	d.eventListeners[hx.EventInput].fire(hx.EventContext{})
	if got := strings.Join(order, ","); got != "binding 2,handler 2" {
		t.Fatalf("ran %q, want the last binding and the last handler", got)
	}
}

func TestTheBindMethodsAndOnShareOneDOMListener(t *testing.T) {
	sig := hx.Signal("")
	d := hx.Input().BindOnInput(sig).OnInput(func(hx.EventContext) {}).(*VNode)
	if len(d.eventListeners) != 1 {
		t.Fatalf("%d listeners for one event", len(d.eventListeners))
	}
	l := d.eventListeners[hx.EventInput]
	if l.bound == nil || l.handler == nil {
		t.Fatal("the two slots must be set")
	}
}

// A handler that sets several signals makes each effect that depends on them
// run once, when the handler ends, and the binding of the same event goes in the
// same batch.
func TestAHandlerRunsInABatch(t *testing.T) {
	a, b, typed := hx.Signal(0), hx.Signal(0), hx.Signal("")
	runs := 0
	hx.EffectFunc(func() { runs++; a.Get(); b.Get(); typed.Get() })

	input := hx.Input().OnInput(func(hx.EventContext) {
		a.Set(1)
		b.Set(1)
		if runs != 1 {
			t.Fatalf("an effect ran in the middle of the handler: runs=%d", runs)
		}
	}).(*VNode)
	input.onBound(hx.EventInput, func(hx.EventContext) { typed.Set("x") })
	input.eventListeners[hx.EventInput].fire(hx.EventContext{})

	if runs != 2 {
		t.Fatalf("runs=%d, want 2: one at creation and one for the whole event", runs)
	}
}

func TestDynamicIsBuiltAgainWhenItsSignalsChange(t *testing.T) {
	items := hx.Signal([]string{"a"})
	inner := hx.Signal("x")
	ul := hx.Ul().Body(hx.Dynamic(func() hx.INode {
		return hx.Each(items, func(i int, v string) hx.INode {
			return hx.Li().BindText(hx.Computed(func() string { return v + inner.Get() }))
		})
	}))
	if got := render(ul); got != `<ul><li>ax</li></ul>` {
		t.Fatalf("got %s", got)
	}
	items.Set([]string{"a", "b"})
	if got := render(ul); got != `<ul><li>ax</li><li>bx</li></ul>` {
		t.Fatalf("after the change: %s", got)
	}
	// the bindings of the rows that were replaced do not stay subscribed
	before := inner.Subscribers()
	items.Set([]string{"a", "b", "c"})
	items.Set([]string{"a"})
	if inner.Subscribers() >= before+3 || inner.Subscribers() > 1 {
		t.Fatalf("subscribers of inner: %d before, %d after; the rows that left must be released", before, inner.Subscribers())
	}
}

func TestDynamicCanShowNothing(t *testing.T) {
	show := hx.Signal(false)
	d := hx.Div().Body(hx.Dynamic(func() hx.INode {
		return hx.IfFn(show, func() hx.INode { return hx.P().Text("hi") })
	}))
	if got := render(d); got != `<div></div>` {
		t.Fatalf("got %s", got)
	}
	show.Set(true)
	if got := render(d); got != `<div><p>hi</p></div>` {
		t.Fatalf("got %s", got)
	}
	show.Set(false)
	if got := render(d); got != `<div></div>` {
		t.Fatalf("got %s", got)
	}
}
