package wasm

import (
	"fmt"

	"github.com/deltegui/hx"
)

// Renderer decides when and how the changes of the nodes reach the output.
// DiffRenderer updates the browser DOM and StringRenderer writes HTML.
type Renderer interface {
	ScheduleRender()
	Mark(element *VNode)
}

type changeStatus int

const (
	unchanged changeStatus = iota
	changeNew
	changeModified
	changeDeleted
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
	handler  func(hx.EventContext)
	attached bool
	domListener
}

// binding is an Effect owned by a node. It is disposed when the node leaves the
// tree and created again if the node comes back.
type binding struct {
	run    func()
	stop   func() // called when the binding is deactivated, may be nil
	effect *hx.Effect
}

// VNode is the node of the browser engine. It keeps the state you want (text, attributes, classes, styles, listeners and children) and the renderer applies the difference to the DOM.
// It implements hx.INode.
type VNode struct {
	domState    // the DOM element and what goes with it, only in the browser
	bindings    []*binding
	deactivated bool

	onRendered []func()
	onRemoved  []func()
	attached   bool // OnRendered has run and OnRemoved has not

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
	eventListeners map[hx.Event]*eventListener
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
		eventListeners: map[hx.Event]*eventListener{},
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
// BindWith is Effect with a function that runs when the node leaves the tree,
// after the effect has been stopped. hx.EachKeyed uses it.
func (element *VNode) BindWith(fn, stop func()) {
	element.bindWith(fn, stop)
}

func (element *VNode) bindWith(fn, stop func()) {
	b := &binding{run: fn, stop: stop}
	b.effect = hx.EffectFunc(fn)
	element.bindings = append(element.bindings, b)
}

// Effect runs fn in an Effect that belongs to the node, like the Bind methods do:
// it is disposed when the node is removed from the tree, and it runs again if
// the node is added back. Prefer it to hx.EffectFunc for effects that only make
// sense while the node is alive.
func (element *VNode) Effect(fn func()) hx.INode {
	element.bind(fn)
	return element
}

// deactivate frees what a node that leaves the tree holds: its DOM listeners and
// its bindings. It applies to the descendants too.
func (element *VNode) deactivate() {
	element.deactivated = true
	element.fireRemoved()
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
	element.queueRendered()
	element.reviveListeners()
	for _, b := range element.bindings {
		if b.effect == nil || b.effect.Disposed() {
			b.effect = hx.EffectFunc(b.run)
		}
	}
	for _, child := range element.children {
		if child != nil {
			child.reactivate()
		}
	}
}

// OnRendered runs fn when the node is in the page and has been rendered, so
// Underlying() returns its DOM element and the element is in the document. It
// runs after the renderer has finished the frame, so the rest of the page is
// rendered too. Use it to hand the element to something that needs it, such as
// a chart library.
//
// It runs the first time, and again each time the node is put back after being
// removed (see OnRemoved). It never runs while rendering on a server.
func (element *VNode) OnRendered(fn func()) hx.INode {
	element.onRendered = append(element.onRendered, fn)
	element.queueRendered()
	return element
}

// OnRemoved runs fn when a node that OnRendered has run for leaves the tree: a
// parent replaced its children, or If, Show or a list dropped it. Use it to
// undo what OnRendered did. It does not run when the node is only disposed
// with its root; use hx.OnCleanup for that.
func (element *VNode) OnRemoved(fn func()) hx.INode {
	element.onRemoved = append(element.onRemoved, fn)
	return element
}

func (element *VNode) fireRemoved() {
	if !element.attached {
		return
	}
	element.attached = false
	for _, fn := range element.onRemoved {
		hx.Untrack(fn)
	}
}

// Discard frees a node that was built but is not going to be used (see hx.If).
func (element *VNode) Discard() { element.deactivate() }

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
func (element *VNode) Body(childs ...hx.INode) hx.INode {
	return element.BodyList(childs)
}

// BodyList is Body with a slice.
func (element *VNode) BodyList(childs []hx.INode) hx.INode {
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
func (element *VNode) On(event hx.Event, handler func(ctx hx.EventContext)) hx.INode {
	listener, ok := element.eventListeners[event]
	if ok {
		listener.handler = handler
		return element
	}
	element.eventListeners[event] = &eventListener{handler: handler}
	element.setDirty(flagEventListeners)
	return element
}

// OnClick is On(hx.EventClick, handler).
func (element *VNode) OnClick(handler func(ctx hx.EventContext)) hx.INode {
	return element.On(hx.EventClick, handler)
}

// Text sets the text of the node. It is kept in its own text node, before the children.
func (element *VNode) Text(t string) hx.INode {
	if element.text.assign(t, changeModified) {
		element.mark()
	}
	return element
}

// BindText keeps the text in sync with the signal. The binding belongs to the node: it stops when the node leaves the tree and starts again if the node comes back.
func (element *VNode) BindText(signal hx.Gettable[string]) hx.INode {
	element.bind(func() {
		element.Text(signal.Get())
	})
	return element
}

// Class adds classes. Empty classes are ignored, and classes with spaces are ignored with a warning.
func (element *VNode) Class(classes ...string) hx.INode {
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
func (element *VNode) RemoveClass(classes ...string) hx.INode {
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
func (element *VNode) Attribute(key, value string) hx.INode {
	if len(key) <= 0 {
		return element
	}
	if setEntry(element.attributes, key, value) {
		element.setDirty(flagAttributes)
	}
	return element
}

// RemoveAttribute removes an attribute.
func (element *VNode) RemoveAttribute(key string) hx.INode {
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
func (element *VNode) Style(key, value string) hx.INode {
	if setEntry(element.styles, key, value) {
		element.setDirty(flagStyles)
	}
	return element
}

// RemoveStyle removes a CSS property.
func (element *VNode) RemoveStyle(key string) hx.INode {
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
func (element *VNode) Id(id string) hx.INode {
	if element.id.assign(id, changeModified) {
		element.mark()
	}
	return element
}

// Src sets the src attribute.
func (element *VNode) Src(v string) hx.INode { return element.Attribute("src", v) }

// Href sets the href attribute.
func (element *VNode) Href(v string) hx.INode { return element.Attribute("href", v) }

// Placeholder sets the placeholder attribute.
func (element *VNode) Placeholder(v string) hx.INode { return element.Attribute("placeholder", v) }

// Type sets the type attribute, for example of an input.
func (element *VNode) Type(v string) hx.INode { return element.Attribute("type", v) }

// Selected marks an Option as selected.
func (element *VNode) Selected() hx.INode { return element.Attribute("selected", "selected") }

// Textf is Text with a format.
func (element *VNode) Textf(format string, args ...any) hx.INode {
	return element.Text(fmt.Sprintf(format, args...))
}

// ClassIf adds the class when on is true and removes it otherwise.
func (element *VNode) ClassIf(class string, on bool) hx.INode {
	if on {
		return element.Class(class)
	}
	return element.RemoveClass(class)
}

// Value sets the value property of an input, textarea, select or option.
func (element *VNode) Value(t string) hx.INode {
	// Not deduplicated: the user can change the value in the DOM at any time,
	// so the renderer compares with the real value when it paints.
	element.value.nextValue = t
	element.value.status = changeModified
	element.mark()
	return element
}

// Checked sets whether a checkbox or radio button is checked.
func (element *VNode) Checked(on bool) hx.INode {
	// Not deduplicated, like Value: the user can toggle it at any time, so the
	// renderer compares with the real state when it paints.
	element.checked.nextValue = on
	element.checked.status = changeModified
	element.mark()
	return element
}

// BindChecked keeps a checkbox or radio button in sync with the signal.
func (element *VNode) BindChecked(signal hx.Gettable[bool]) hx.INode {
	element.bind(func() { element.Checked(signal.Get()) })
	return element
}

// BindOnChecked writes the checked state of the element to the signal on every change event.
func (element *VNode) BindOnChecked(signal hx.Settable[bool]) hx.INode {
	return element.On(hx.EventChange, func(ctx hx.EventContext) { signal.Set(targetChecked(ctx)) })
}

// BindValue keeps the value of an input, textarea or option in sync with the
// signal. A textarea also gets it as its text.
func (element *VNode) BindValue(signal hx.Gettable[string]) hx.INode {
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
func (element *VNode) BindOnInput(signal hx.Settable[string]) hx.INode {
	return element.On(hx.EventInput, func(ctx hx.EventContext) { signal.Set(targetValue(ctx)) })
}

// BindOnChange writes the value of the element to the signal on every change event.
func (element *VNode) BindOnChange(signal hx.Settable[string]) hx.INode {
	return element.On(hx.EventChange, func(ctx hx.EventContext) { signal.Set(targetValue(ctx)) })
}

// OnInput is On(hx.EventInput, handler).
func (element *VNode) OnInput(handler func(ctx hx.EventContext)) hx.INode {
	return element.On(hx.EventInput, handler)
}

// OnChange is On(hx.EventChange, handler).
func (element *VNode) OnChange(handler func(ctx hx.EventContext)) hx.INode {
	return element.On(hx.EventChange, handler)
}

// OnKeyUp is On(hx.EventKeyUp, handler).
func (element *VNode) OnKeyUp(handler func(ctx hx.EventContext)) hx.INode {
	return element.On(hx.EventKeyUp, handler)
}

// OnKeyDown is On(hx.EventKeyDown, handler).
func (element *VNode) OnKeyDown(handler func(ctx hx.EventContext)) hx.INode {
	return element.On(hx.EventKeyDown, handler)
}

// OnSubmit is On(hx.EventSubmit, handler). Call ctx.Event.PreventDefault() to stop the page from reloading.
func (element *VNode) OnSubmit(handler func(ctx hx.EventContext)) hx.INode {
	return element.On(hx.EventSubmit, handler)
}

// OnFocus is On(hx.EventFocus, handler).
func (element *VNode) OnFocus(handler func(ctx hx.EventContext)) hx.INode {
	return element.On(hx.EventFocus, handler)
}

// OnBlur is On(hx.EventBlur, handler).
func (element *VNode) OnBlur(handler func(ctx hx.EventContext)) hx.INode {
	return element.On(hx.EventBlur, handler)
}

const noopIdNode string = "noop"

func asVNode(i hx.INode) *VNode {
	if v, ok := i.(*VNode); ok {
		return v
	}
	panic("wasm: the node was not made by the browser engine; a program cannot mix engines")
}
