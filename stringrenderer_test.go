package hx

import "testing"

func render(node INode) string {
	r := &StringRenderer{}
	NewWithoutMount("DIV", r) // make sure the constructor works without DOM
	r.Render(node.AsVNode())
	return r.String()
}

func TestStringRendererAttributesAndStyles(t *testing.T) {
	got := render(Div().Id("x").Class("b", "a").Style("color", "red").Attribute("title", "hi").Text("a<b"))
	want := `<DIV id="x" class="a b" style="color:red;" title="hi">a&lt;b</DIV>`
	if got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
}

func TestStringRendererVoidAndInput(t *testing.T) {
	got := render(Div().Body(Input().Type("text").Value("v"), Br()))
	want := `<DIV><INPUT value="v" type="text"/><BR/></DIV>`
	if got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
}

func TestStringRendererNoopIsTransparent(t *testing.T) {
	got := render(Ul().Body(Noop().Body(Li().Text("1"), Li().Text("2"))))
	want := `<UL><LI>1</LI><LI>2</LI></UL>`
	if got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
}

func TestStringRendererSkipsRemoved(t *testing.T) {
	got := render(Div().Class("a", "b").RemoveClass("a").Style("x", "1").RemoveStyle("x").Attribute("k", "v").RemoveAttribute("k"))
	want := `<DIV class="b"></DIV>`
	if got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
}

func TestBodyReplacedDropsOldChildren(t *testing.T) {
	d := Div()
	a, b := Span().Text("a"), Span().Text("b")
	d.Body(a, b)
	d.Body(b)
	if got := render(d); got != `<DIV><SPAN>b</SPAN></DIV>` {
		t.Fatalf("got %s", got)
	}
	if len(d.AsVNode().children) != 2 { // a (deleted, pending removal) and b
		t.Fatalf("children=%d", len(d.AsVNode().children))
	}
}

func TestClassRemovedAndAddedAgain(t *testing.T) {
	d := Div().Class("a")
	d.AsVNode().classes["a"] = unchanged // as if rendered
	d.RemoveClass("a").Class("a")
	if !d.HaveClass("a") {
		t.Fatal("class lost")
	}
}

func TestAttributeEdgeCases(t *testing.T) {
	d := Div().Attribute("disabled", "")
	if !d.HaveAttribute("disabled", "") {
		t.Fatal("empty attribute ignored")
	}
	v := d.AsVNode()
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
	d := Div().Style("color", "red")
	v := d.AsVNode()
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
	d := Div()
	called := ""
	d.OnClick(func(EventContext) { called = "first" })
	d.OnClick(func(EventContext) { called = "second" })
	d.AsVNode().eventListeners[EventClick].handler(EventContext{})
	if called != "second" {
		t.Fatalf("called=%s", called)
	}
}

func TestBindingStopsWhenDeactivated(t *testing.T) {
	s := Signal("a")
	node := Span().BindText(s)
	if got := render(node); got != `<SPAN>a</SPAN>` {
		t.Fatalf("got %s", got)
	}
	s.Set("b")
	if got := render(node); got != `<SPAN>b</SPAN>` {
		t.Fatalf("got %s", got)
	}

	node.AsVNode().deactivate()
	s.Set("c")
	if got := render(node); got != `<SPAN>b</SPAN>` {
		t.Fatalf("binding still active after deactivate: %s", got)
	}
	if len(s.subscribers) != 0 {
		t.Fatalf("subscribers=%d", len(s.subscribers))
	}

	node.AsVNode().reactivate()
	if got := render(node); got != `<SPAN>c</SPAN>` {
		t.Fatalf("binding not restored with the current value: %s", got)
	}
	s.Set("d")
	if got := render(node); got != `<SPAN>d</SPAN>` {
		t.Fatalf("binding not live after reactivate: %s", got)
	}
}

func TestDeactivateReachesDescendants(t *testing.T) {
	s := Signal("a")
	child := Span().BindText(s)
	parent := Div().Body(Noop().Body(child))
	parent.AsVNode().deactivate()
	if len(s.subscribers) != 0 {
		t.Fatalf("descendant binding still subscribed")
	}
	parent.AsVNode().reactivate()
	if len(s.subscribers) != 1 {
		t.Fatalf("descendant binding not restored")
	}
}

func TestBindingInRootIsDisposedByRoot(t *testing.T) {
	s := Signal("a")
	dispose := CreateRoot(func(dispose func()) func() {
		Span().BindText(s)
		return dispose
	})
	dispose()
	if len(s.subscribers) != 0 {
		t.Fatalf("subscribers=%d", len(s.subscribers))
	}
}

func TestNodeEffectFollowsTheNode(t *testing.T) {
	s := Signal(1)
	runs := 0
	node := Div().Effect(func() {
		runs++
		s.Get()
	})
	s.Set(2)
	if runs != 2 {
		t.Fatalf("runs=%d", runs)
	}

	node.AsVNode().deactivate()
	s.Set(3)
	if runs != 2 {
		t.Fatalf("effect ran after deactivate, runs=%d", runs)
	}

	node.AsVNode().reactivate()
	if runs != 3 {
		t.Fatalf("effect did not run again on reactivate, runs=%d", runs)
	}
	s.Set(4)
	if runs != 4 {
		t.Fatalf("effect not live after reactivate, runs=%d", runs)
	}
}

func TestNodeEffectCanRebuildItsOwnBody(t *testing.T) {
	items := Signal([]string{"a"})
	list := Ul()
	list.Effect(func() {
		list.Body(Each(items, func(i int, v string) INode { return Li().Text(v) }))
	})
	items.Set([]string{"a", "b"})
	if got := render(list); got != `<UL><LI>a</LI><LI>b</LI></UL>` {
		t.Fatalf("got %s", got)
	}
}

func TestInputInheritsEffect(t *testing.T) {
	s := Signal(0)
	runs := 0
	Input().Effect(func() { runs++; s.Get() })
	s.Set(1)
	if runs != 2 {
		t.Fatalf("runs=%d", runs)
	}
}

func TestIfDiscardsUnusedChild(t *testing.T) {
	cond := Signal(false)
	text := Signal("a")
	var got INode
	EffectFunc(func() {
		got = If(cond, Span().BindText(text))
	})
	if got != nil {
		t.Fatal("expected nil when the condition is false")
	}
	if len(text.subscribers) != 0 {
		t.Fatalf("discarded child is still subscribed: %d", len(text.subscribers))
	}
}

func TestIfKeepsUsedChildActive(t *testing.T) {
	text := Signal("a")
	var got INode
	EffectFunc(func() {
		got = If(Value(true), Span().BindText(text))
	})
	if got == nil || len(text.subscribers) != 1 {
		t.Fatalf("used child must stay active, subscribers=%d", len(text.subscribers))
	}
}

func TestShowDiscardsOnlyTheOtherPath(t *testing.T) {
	a, b := Signal("a"), Signal("b")
	var got INode
	EffectFunc(func() {
		got = Show(Value(true), Span().BindText(a), Span().BindText(b))
	})
	if len(a.subscribers) != 1 || len(b.subscribers) != 0 {
		t.Fatalf("a=%d b=%d", len(a.subscribers), len(b.subscribers))
	}
	if got == nil {
		t.Fatal("nil result")
	}
}

func TestDiscardedNodeComesBack(t *testing.T) {
	text := Signal("a")
	node := Span().BindText(text)
	If(Value(false), node)
	text.Set("b")
	node.AsVNode().reactivate()
	if got := render(node); got != `<SPAN>b</SPAN>` {
		t.Fatalf("got %s", got)
	}
	if len(text.subscribers) != 1 {
		t.Fatalf("subscribers=%d", len(text.subscribers))
	}
}

func TestFluentChainingInAnyOrder(t *testing.T) {
	// Input-specific and generic methods can be mixed, because everything returns *VNode.
	got := render(Input().Placeholder("p").Class("c").Type("text").Id("x").Style("a", "b").Value("v").Attribute("k", "v"))
	want := `<INPUT id="x" class="c" style="a:b;" value="v" k="v" placeholder="p" type="text"/>`
	if got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
}

func TestFluentHelpers(t *testing.T) {
	got := render(A().Href("/x").ClassIf("on", true).ClassIf("off", false).Textf("%d items", 3))
	want := `<A class="on" href="/x">3 items</A>`
	if got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
	if got := render(El("details").Body(El("summary").Text("s"))); got != `<DETAILS><SUMMARY>s</SUMMARY></DETAILS>` {
		t.Fatalf("got %s", got)
	}
}

// A function that returns INode can still be chained.
func fluentCard() INode { return Div().Class("card") }

func TestINodeReturnCanBeChained(t *testing.T) {
	if got := render(fluentCard().Class("wide").Text("x")); got != `<DIV class="card wide">x</DIV>` {
		t.Fatalf("got %s", got)
	}
}

func TestTextAreaBindValueSetsText(t *testing.T) {
	s := Signal("hi")
	if got := render(TextArea().BindValue(s)); got != `<TEXTAREA>hi</TEXTAREA>` {
		t.Fatalf("got %s", got)
	}
}

func TestIfFnBuildsOnlyWhenTrue(t *testing.T) {
	built := 0
	build := func() INode { built++; return Span().Text("x") }

	if IfFn(Value(false), build) != nil || built != 0 {
		t.Fatalf("built=%d when the condition is false", built)
	}
	if got := IfFn(Value(true), build); got == nil || built != 1 {
		t.Fatalf("built=%d when the condition is true", built)
	}
}

func TestShowFnBuildsOnlyTheUsedPath(t *testing.T) {
	a, b := 0, 0
	buildA := func() INode { a++; return Span().Text("a") }
	buildB := func() INode { b++; return Span().Text("b") }

	if got := render(ShowFn(Value(true), buildA, buildB)); got != `<SPAN>a</SPAN>` || a != 1 || b != 0 {
		t.Fatalf("got %s a=%d b=%d", got, a, b)
	}
	if got := render(ShowFn(Value(false), buildA, buildB)); got != `<SPAN>b</SPAN>` || a != 1 || b != 1 {
		t.Fatalf("got %s a=%d b=%d", got, a, b)
	}
	if ShowFn(Value(false), buildA, nil) != nil {
		t.Fatal("a nil else path must give nil")
	}
}

func TestIfFnBindingStaysActive(t *testing.T) {
	text := Signal("a")
	var got INode
	EffectFunc(func() {
		got = IfFn(Value(true), func() INode { return Span().BindText(text) })
	})
	if got == nil || len(text.subscribers) != 1 {
		t.Fatalf("subscribers=%d", len(text.subscribers))
	}
}
