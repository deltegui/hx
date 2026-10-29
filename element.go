package hx

import (
	"fmt"
	"strings"
)

// Renderer decides when and how the changes of the nodes reach the output.
// DiffRenderer updates the browser DOM and StringRenderer writes HTML.
type Renderer interface {
	ScheduleRender()
	Mark(element *VNode)
}

// EventContext is what an event handler receives: the node that has the listener and the browser event.
type EventContext struct {
	Target INode
	// Event is the browser event (a dom.Event from honnef.co/go/js/dom/v2).
	// It is only available when the code runs in the browser.
	Event domEvent
}

// Event is the name of a DOM event.
type Event string

const (
	EventClick  Event = "click"
	EventInput  Event = "input"
	EventChange Event = "change"
	EventKeyUp  Event = "keyup"

	EventKeyDown    Event = "keydown"
	EventSubmit     Event = "submit"
	EventFocus      Event = "focus"
	EventBlur       Event = "blur"
	EventDblClick   Event = "dblclick"
	EventMouseEnter Event = "mouseenter"
	EventMouseLeave Event = "mouseleave"
)

type changeStatus int

const (
	unchanged changeStatus = iota
	changeNew
	changeModified
	changeDeleted
)

// INode is a node of the tree. Every method of VNode returns *VNode, so calls
// can be chained in any order, and a function that returns INode can be chained
// too. The old per-element types are aliases of VNode.
type INode = *VNode

// Deprecated aliases kept for compatibility: they are all VNode.
type (
	InputVNode   = VNode
	TextAreaNode = VNode
	OptionNode   = VNode
	AVNode       = VNode
	NoopNode     = VNode
)

type dirtyFlag int

const flagNumber = 5

const (
	flagStyles dirtyFlag = iota
	flagClasses
	flagAttributes
	flagEventListeners
	flagChildren
)

// eventListener is the handler of an event. The handler can be replaced at
// any moment. The DOM listener (wrapper) is only registered once per DOM element.
type eventListener struct {
	handler  func(EventContext)
	attached bool
	domListener
}

// binding is an Effect owned by a node. It is disposed when the node leaves the
// tree and created again if the node comes back.
type binding struct {
	run    func()
	stop   func() // called when the binding is deactivated, may be nil
	effect *Effect
}

const svgNamespace = "http://www.w3.org/2000/svg"

// VNode is a node of the UI tree. It keeps the state you want (text, attributes, classes, styles, listeners and children) and the renderer applies the difference to the DOM.
// Every method returns *VNode so calls can be chained.
type VNode struct {
	domState    // the DOM element and what goes with it, only in the browser
	bindings    []*binding
	deactivated bool

	father *VNode
	status changeStatus

	renderer     Renderer
	haveRenderer bool

	tag       string
	namespace string
	id        diffValue[string]
	text      diffValue[string]
	value     diffValue[string]
	checked   diffValue[bool]

	styles         map[string]diffValue[string]
	classes        map[string]changeStatus
	attributes     map[string]diffValue[string]
	eventListeners map[Event]*eventListener
	children       []*VNode

	dirtyFlags [flagNumber]bool
}

// NewWithoutMount creates a node that is not attached to the DOM, for example to render HTML with a StringRenderer.
func NewWithoutMount(tag string, renderer Renderer) *VNode {
	VNode := newVNode(tag)
	VNode.setRenderer(renderer, true)
	return VNode
}

func newVNode(tag string) *VNode {
	vnode := &VNode{
		status: changeNew,

		father: nil,

		tag:            tag,
		id:             diffValue[string]{},
		text:           diffValue[string]{},
		value:          diffValue[string]{},
		styles:         map[string]diffValue[string]{},
		classes:        map[string]changeStatus{},
		attributes:     map[string]diffValue[string]{},
		eventListeners: map[Event]*eventListener{},
		children:       []*VNode{},

		dirtyFlags: [flagNumber]bool{false},
	}
	return vnode
}

func newVNodeNS(tag, namespace string) *VNode {
	vnode := newVNode(tag)
	vnode.namespace = namespace
	return vnode
}

// bind runs fn in an Effect that belongs to the node, so it stops when the node
// is removed from the tree.
func (element *VNode) bind(fn func()) {
	element.bindWith(fn, nil)
}

// bindWith is bind with a function that runs when the node leaves the tree,
// after the effect has been stopped.
func (element *VNode) bindWith(fn, stop func()) {
	b := &binding{run: fn, stop: stop}
	b.effect = EffectFunc(fn)
	element.bindings = append(element.bindings, b)
}

// Effect runs fn in an Effect that belongs to the node, like the Bind methods do:
// it is disposed when the node is removed from the tree, and it runs again if
// the node is added back. Prefer it to EffectFunc for effects that only make
// sense while the node is alive.
func (element *VNode) Effect(fn func()) INode {
	element.bind(fn)
	return element
}

// deactivate frees what a node that leaves the tree holds: its DOM listeners and
// its bindings. It applies to the descendants too.
func (element *VNode) deactivate() {
	element.deactivated = true
	element.releaseListeners()
	for _, b := range element.bindings {
		if b.effect != nil {
			b.effect.Dispose()
			b.effect = nil
			if b.stop != nil {
				b.stop()
			}
		}
	}
	for _, child := range element.children {
		if child != nil {
			child.deactivate()
		}
	}
}

// reactivate undoes deactivate for a node that is added to the tree again.
// It does nothing for a node that is active.
func (element *VNode) reactivate() {
	if !element.deactivated {
		return
	}
	element.deactivated = false
	element.reviveListeners()
	for _, b := range element.bindings {
		if b.effect == nil || b.effect.disposed {
			b.effect = EffectFunc(b.run)
		}
	}
	for _, child := range element.children {
		if child != nil {
			child.reactivate()
		}
	}
}

// AsVNode returns the node itself. It exists for compatibility with code that used the INode interface.
func (element *VNode) AsVNode() *VNode {
	return element
}

func (element *VNode) mark() {
	if element.haveRenderer {
		element.renderer.Mark(element)
		element.renderer.ScheduleRender()
	}
}

func (element *VNode) clearDirty() {
	element.dirtyFlags = [flagNumber]bool{false}
}

func (element *VNode) setDirty(flag dirtyFlag) {
	element.dirtyFlags[flag] = true
	element.mark()
}

func (element *VNode) isDirty(flag dirtyFlag) bool {
	return element.dirtyFlags[flag]
}

func (element *VNode) setRenderer(renderer Renderer, haveRenderer bool) {
	element.renderer = renderer
	element.haveRenderer = haveRenderer
	for index := range element.children {
		element.children[index].setRenderer(renderer, haveRenderer)
		element.children[index].haveRenderer = haveRenderer
	}
	element.setDirty(flagChildren)
}

func (element *VNode) scheludeRender() {
	if element.haveRenderer {
		element.renderer.ScheduleRender()
	}
}

// Body replaces the children of the node. Nil children are skipped.
// Nodes that were already children and are passed again are reused (moved), not recreated.
func (element *VNode) Body(childs ...INode) *VNode {
	return element.BodyList(childs)
}

// BodyList is Body with a slice.
func (element *VNode) BodyList(childs []INode) *VNode {
	incoming := make(map[*VNode]struct{}, len(childs))
	for _, child := range childs {
		if child != nil {
			incoming[asVNode(child)] = struct{}{}
		}
	}

	// Nodes that are given again are re-added below, so they must not be duplicated.
	kept := make([]*VNode, 0, len(element.children))
	for _, child := range element.children {
		if child == nil {
			continue
		}
		if _, again := incoming[child]; again {
			continue
		}
		child.status = changeDeleted
		kept = append(kept, child)
	}
	element.children = kept

	for _, child := range childs {
		if child == nil {
			continue
		}
		realNode := asVNode(child)
		realNode.status = changeNew
		realNode.setRenderer(element.renderer, element.haveRenderer)
		realNode.father = element
		element.children = append(element.children, realNode)
	}

	element.setDirty(flagChildren)
	element.scheludeRender()

	return element
}

// On sets the handler of an event. Calling it again for the same event replaces the handler.
// The handler runs untracked, and a render is scheduled after it.
func (element *VNode) On(event Event, handler func(ctx EventContext)) *VNode {
	listener, ok := element.eventListeners[event]
	if ok {
		listener.handler = handler
		return element
	}
	element.eventListeners[event] = &eventListener{handler: handler}
	element.setDirty(flagEventListeners)
	return element
}

// OnClick is On(EventClick, handler).
func (element *VNode) OnClick(handler func(ctx EventContext)) INode {
	return element.On(EventClick, handler)
}

// Text sets the text of the node. It is kept in its own text node, before the children.
func (element *VNode) Text(t string) *VNode {
	if element.text.assign(t, changeModified) {
		element.mark()
	}
	return element
}

// BindText keeps the text in sync with the signal. The binding belongs to the node: it stops when the node leaves the tree and starts again if the node comes back.
func (element *VNode) BindText(signal Gettable[string]) *VNode {
	element.bind(func() {
		element.Text(signal.Get())
	})
	return element
}

// Class adds classes. Empty classes are ignored, and classes with spaces are ignored with a warning.
func (element *VNode) Class(classes ...string) *VNode {
	for _, create := range classes {
		if len(create) <= 0 {
			continue
		}
		status, ok := element.classes[create]
		if !ok {
			element.classes[create] = changeNew
			element.setDirty(flagClasses)
		} else if status == changeDeleted {
			// Removed and added again before rendering: it is still in the DOM.
			element.classes[create] = unchanged
			element.setDirty(flagClasses)
		}
	}
	return element
}

// RemoveClass removes classes.
func (element *VNode) RemoveClass(classes ...string) *VNode {
	for _, remove := range classes {
		if len(remove) <= 0 {
			continue
		}
		status, ok := element.classes[remove]
		if !ok || status == changeDeleted {
			continue
		}
		if status == changeNew {
			// It was never rendered, so there is nothing to remove from the DOM.
			delete(element.classes, remove)
		} else {
			element.classes[remove] = changeDeleted
		}
		element.setDirty(flagClasses)
	}
	return element
}

// HaveClass tells if the node will have the class after the next render.
func (element *VNode) HaveClass(class string) bool {
	changeStatus, ok := element.classes[class]
	return ok && changeStatus != changeDeleted
}

// Attribute sets an attribute. An empty value is valid (for example disabled).
func (element *VNode) Attribute(key, value string) *VNode {
	if len(key) <= 0 {
		return element
	}
	if setEntry(element.attributes, key, value) {
		element.setDirty(flagAttributes)
	}
	return element
}

// RemoveAttribute removes an attribute.
func (element *VNode) RemoveAttribute(key string) *VNode {
	if removeEntry(element.attributes, key) {
		element.setDirty(flagAttributes)
	}
	return element
}

// GetAttribute returns the value the attribute will have after the next render, or an empty string.
func (element *VNode) GetAttribute(key string) string {
	value, ok := element.attributes[key]
	if !ok {
		return ""
	}
	if value.status == changeDeleted {
		return ""
	}
	return value.Value()
}

// HaveAttribute tells if the attribute will have that value after the next render.
func (element *VNode) HaveAttribute(key, value string) bool {
	v, ok := element.attributes[key]
	return ok && v.status != changeDeleted && v.Value() == value
}

// Style sets a CSS property. Properties are written together in the style attribute.
func (element *VNode) Style(key, value string) *VNode {
	if setEntry(element.styles, key, value) {
		element.setDirty(flagStyles)
	}
	return element
}

// RemoveStyle removes a CSS property.
func (element *VNode) RemoveStyle(key string) *VNode {
	if removeEntry(element.styles, key) {
		element.setDirty(flagStyles)
	}
	return element
}

// GetStyle returns the value the property will have after the next render, or an empty string.
func (element *VNode) GetStyle(key string) string {
	value, ok := element.styles[key]
	if !ok {
		return ""
	}
	if value.status == changeDeleted {
		return ""
	}
	return value.Value()
}

// HaveStyle tells if the property will have that value after the next render.
func (element *VNode) HaveStyle(key, value string) bool {
	v, ok := element.styles[key]
	return ok && v.status != changeDeleted && v.Value() == value
}

// Id sets the id attribute of the node.
func (element *VNode) Id(id string) *VNode {
	if element.id.assign(id, changeModified) {
		element.mark()
	}
	return element
}

// Src sets the src attribute.
func (element *VNode) Src(v string) *VNode { return element.Attribute("src", v) }

// Href sets the href attribute.
func (element *VNode) Href(v string) *VNode { return element.Attribute("href", v) }

// Placeholder sets the placeholder attribute.
func (element *VNode) Placeholder(v string) *VNode { return element.Attribute("placeholder", v) }

// Type sets the type attribute, for example of an input.
func (element *VNode) Type(v string) *VNode { return element.Attribute("type", v) }

// Selected marks an Option as selected.
func (element *VNode) Selected() *VNode { return element.Attribute("selected", "selected") }

// Textf is Text with a format.
func (element *VNode) Textf(format string, args ...any) *VNode {
	return element.Text(fmt.Sprintf(format, args...))
}

// ClassIf adds the class when on is true and removes it otherwise.
func (element *VNode) ClassIf(class string, on bool) *VNode {
	if on {
		return element.Class(class)
	}
	return element.RemoveClass(class)
}

// Value sets the value property of an input, textarea, select or option.
func (element *VNode) Value(t string) *VNode {
	// Not deduplicated: the user can change the value in the DOM at any time,
	// so the renderer compares with the real value when it paints.
	element.value.nextValue = t
	element.value.status = changeModified
	element.mark()
	return element
}

// Checked sets whether a checkbox or radio button is checked.
func (element *VNode) Checked(on bool) *VNode {
	// Not deduplicated, like Value: the user can toggle it at any time, so the
	// renderer compares with the real state when it paints.
	element.checked.nextValue = on
	element.checked.status = changeModified
	element.mark()
	return element
}

// BindChecked keeps a checkbox or radio button in sync with the signal.
func (element *VNode) BindChecked(signal Gettable[bool]) *VNode {
	element.bind(func() { element.Checked(signal.Get()) })
	return element
}

// BindOnChecked writes the checked state of the element to the signal on every change event.
func (element *VNode) BindOnChecked(signal Settable[bool]) *VNode {
	return element.On(EventChange, func(ctx EventContext) { signal.Set(targetChecked(ctx)) })
}

// BindValue keeps the value of an input, textarea or option in sync with the
// signal. A textarea also gets it as its text.
func (element *VNode) BindValue(signal Gettable[string]) *VNode {
	element.bind(func() {
		v := signal.Get()
		element.Value(v)
		if element.tag == "TEXTAREA" {
			element.Text(v)
		}
	})
	return element
}

// BindOnInput writes the value of the element to the signal on every input event.
func (element *VNode) BindOnInput(signal Settable[string]) *VNode {
	return element.On(EventInput, func(ctx EventContext) { signal.Set(targetValue(ctx)) })
}

// BindOnChange writes the value of the element to the signal on every change event.
func (element *VNode) BindOnChange(signal Settable[string]) *VNode {
	return element.On(EventChange, func(ctx EventContext) { signal.Set(targetValue(ctx)) })
}

// OnInput is On(EventInput, handler).
func (element *VNode) OnInput(handler func(ctx EventContext)) *VNode {
	return element.On(EventInput, handler)
}

// OnChange is On(EventChange, handler).
func (element *VNode) OnChange(handler func(ctx EventContext)) *VNode {
	return element.On(EventChange, handler)
}

// OnKeyUp is On(EventKeyUp, handler).
func (element *VNode) OnKeyUp(handler func(ctx EventContext)) *VNode {
	return element.On(EventKeyUp, handler)
}

// OnKeyDown is On(EventKeyDown, handler).
func (element *VNode) OnKeyDown(handler func(ctx EventContext)) *VNode {
	return element.On(EventKeyDown, handler)
}

// OnSubmit is On(EventSubmit, handler). Call ctx.Event.PreventDefault() to stop the page from reloading.
func (element *VNode) OnSubmit(handler func(ctx EventContext)) *VNode {
	return element.On(EventSubmit, handler)
}

// OnFocus is On(EventFocus, handler).
func (element *VNode) OnFocus(handler func(ctx EventContext)) *VNode {
	return element.On(EventFocus, handler)
}

// OnBlur is On(EventBlur, handler).
func (element *VNode) OnBlur(handler func(ctx EventContext)) *VNode {
	return element.On(EventBlur, handler)
}

const noopIdNode string = "noop"

func asVNode(i INode) *VNode {
	return i.AsVNode()
}

// H1 creates a <h1> element.
func H1() *VNode { return newVNode("H1") }

// H2 creates a <h2> element.
func H2() *VNode { return newVNode("H2") }

// H3 creates a <h3> element.
func H3() *VNode { return newVNode("H3") }

// H4 creates a <h4> element.
func H4() *VNode { return newVNode("H4") }

// H5 creates a <h5> element.
func H5() *VNode { return newVNode("H5") }

// H6 creates a <h6> element.
func H6() *VNode { return newVNode("H6") }

// Div creates a <div> element.
func Div() *VNode { return newVNode("DIV") }

// P creates a <p> element.
func P() *VNode { return newVNode("P") }

// Button creates a <button> element.
func Button() *VNode { return newVNode("BUTTON") }

// I creates a <i> element.
func I() *VNode { return newVNode("I") }

// A creates a <a> element.
func A() *VNode { return newVNode("A") }

// Span creates a <span> element.
func Span() *VNode { return newVNode("SPAN") }

// Strong creates a <strong> element.
func Strong() *VNode { return newVNode("STRONG") }

// Em creates a <em> element.
func Em() *VNode { return newVNode("EM") }

// Small creates a <small> element.
func Small() *VNode { return newVNode("SMALL") }

// Img creates a <img> element.
func Img() *VNode { return newVNode("IMG") }

// Input creates a <input> element.
func Input() *VNode { return newVNode("INPUT") }

// Label creates a <label> element.
func Label() *VNode { return newVNode("LABEL") }

// Form creates a <form> element.
func Form() *VNode { return newVNode("FORM") }

// Select creates a <select> element.
func Select() *VNode { return newVNode("SELECT") }

// Option creates a <option> element.
func Option() *VNode { return newVNode("OPTION") }

// TextArea creates a <textarea> element.
func TextArea() *VNode { return newVNode("TEXTAREA") }

// Ul creates a <ul> element.
func Ul() *VNode { return newVNode("UL") }

// Ol creates a <ol> element.
func Ol() *VNode { return newVNode("OL") }

// Li creates a <li> element.
func Li() *VNode { return newVNode("LI") }

// Table creates a <table> element.
func Table() *VNode { return newVNode("TABLE") }

// THead creates a <thead> element.
func THead() *VNode { return newVNode("THEAD") }

// TBody creates a <tbody> element.
func TBody() *VNode { return newVNode("TBODY") }

// TFoot creates a <tfoot> element.
func TFoot() *VNode { return newVNode("TFOOT") }

// Tr creates a <tr> element.
func Tr() *VNode { return newVNode("TR") }

// Th creates a <th> element.
func Th() *VNode { return newVNode("TH") }

// Td creates a <td> element.
func Td() *VNode { return newVNode("TD") }

// Nav creates a <nav> element.
func Nav() *VNode { return newVNode("NAV") }

// Header creates a <header> element.
func Header() *VNode { return newVNode("HEADER") }

// Footer creates a <footer> element.
func Footer() *VNode { return newVNode("FOOTER") }

// Main creates a <main> element.
func Main() *VNode { return newVNode("MAIN") }

// Section creates a <section> element.
func Section() *VNode { return newVNode("SECTION") }

// Article creates a <article> element.
func Article() *VNode { return newVNode("ARTICLE") }

// Aside creates a <aside> element.
func Aside() *VNode { return newVNode("ASIDE") }

// Video creates a <video> element.
func Video() *VNode { return newVNode("VIDEO") }

// Audio creates a <audio> element.
func Audio() *VNode { return newVNode("AUDIO") }

// Source creates a <source> element.
func Source() *VNode { return newVNode("SOURCE") }

// Canvas creates a <canvas> element.
func Canvas() *VNode { return newVNode("CANVAS") }

// Svg creates an <svg> element in the SVG namespace.
func Svg() *VNode { return newVNodeNS("svg", svgNamespace) }

// Path creates an SVG <path> element.
func Path() *VNode { return newVNodeNS("path", svgNamespace) }

// Br creates a <br> element.
func Br() *VNode { return newVNode("BR") }

// Hr creates a <hr> element.
func Hr() *VNode { return newVNode("HR") }

// Code creates a <code> element.
func Code() *VNode { return newVNode("CODE") }

// Pre creates a <pre> element.
func Pre() *VNode { return newVNode("PRE") }

// Noop is a fragment: it renders its children but no element of its own.
func Noop() *VNode { return newVNode(noopIdNode) }

// El creates a node with any tag, for the ones that have no constructor.
func El(tag string) *VNode { return newVNode(strings.ToUpper(tag)) }
