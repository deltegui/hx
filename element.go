package hx

import (
	"syscall/js"

	"honnef.co/go/js/dom/v2"
)

type Renderer interface {
	ScheduleRender()
	Mark(element *VNode)
}

type EventContext struct {
	Target INode
	Event  dom.Event
}

type Event string

const (
	EventClick  Event = "click"
	EventInput  Event = "input"
	EventChange Event = "change"
	EventKeyUp  Event = "keyup"
)

type changeStatus int

const (
	unchanged changeStatus = iota
	changeNew
	changeModified
	changeDeleted
)

type INode interface {
	AsVNode() *VNode

	Id(id string) INode

	Text(t string) INode
	BindText(signal Gettable[string]) INode

	Body(childs ...INode) INode
	BodyList(childs []INode) INode

	Attribute(key, value string) INode
	RemoveAttribute(key string) INode
	GetAttribute(key string) string
	HaveAttribute(key, value string) bool

	Style(key, value string) INode
	RemoveStyle(key string) INode
	GetStyle(key string) string
	HaveStyle(key, value string) bool

	Class(classes ...string) INode
	RemoveClass(classes ...string) INode
	HaveClass(class string) bool

	On(event Event, handler func(ctx EventContext)) INode
	OnClick(handler func(ctx EventContext)) INode

	Effect(fn func()) INode
}

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
	wrapper  js.Func
}

// binding is an Effect owned by a node. It is disposed when the node leaves the
// tree and created again if the node comes back.
type binding struct {
	run    func()
	effect *Effect
}

const svgNamespace = "http://www.w3.org/2000/svg"

type VNode struct {
	domElement     dom.Element
	haveDomElement bool
	textNode       *dom.Text
	bindings       []*binding
	deactivated    bool

	father *VNode
	Owner  INode
	status changeStatus

	renderer     Renderer
	haveRenderer bool

	tag       string
	namespace string
	id        diffValue[string]
	text      diffValue[string]
	value     diffValue[string]

	styles         map[string]diffValue[string]
	classes        map[string]changeStatus
	attributes     map[string]diffValue[string]
	eventListeners map[Event]*eventListener
	children       []*VNode

	dirtyFlags [flagNumber]bool
}

func NewWithRenderer(element dom.Element, renderer Renderer) *VNode {
	VNode := newVNode(element.NodeName())
	VNode.status = unchanged
	VNode.domElement = element
	VNode.haveDomElement = true
	VNode.renderer = renderer
	VNode.haveRenderer = true
	return VNode
}

func New(element dom.Element) *VNode {
	return NewWithRenderer(element, newRenderer(element))
}

func NewFromId(id string) *VNode {
	mountPoint := dom.GetWindow().Document().GetElementByID(id)
	return New(mountPoint)
}

func NewFromIdWithRenderer(id string, renderer Renderer) *VNode {
	mountPoint := dom.GetWindow().Document().GetElementByID(id)
	return NewWithRenderer(mountPoint, renderer)
}

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
	vnode.Owner = vnode
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
	b := &binding{run: fn}
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

func (element *VNode) Underlying() dom.Element {
	return element.domElement
}

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

func (element *VNode) Body(childs ...INode) INode {
	return element.BodyList(childs)
}

func (element *VNode) BodyList(childs []INode) INode {
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

func (element *VNode) On(event Event, handler func(ctx EventContext)) INode {
	listener, ok := element.eventListeners[event]
	if ok {
		listener.handler = handler
		return element
	}
	element.eventListeners[event] = &eventListener{handler: handler}
	element.setDirty(flagEventListeners)
	return element
}

func (element *VNode) OnClick(handler func(ctx EventContext)) INode {
	return element.On(EventClick, handler)
}

func (element *VNode) Text(t string) INode {
	if element.text.assign(t, changeModified) {
		element.mark()
	}
	return element
}

func (element *VNode) BindText(signal Gettable[string]) INode {
	element.bind(func() {
		element.Text(signal.Get())
	})
	return element
}

func (element *VNode) Class(classes ...string) INode {
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

func (element *VNode) RemoveClass(classes ...string) INode {
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

func (element *VNode) HaveClass(class string) bool {
	changeStatus, ok := element.classes[class]
	return ok && changeStatus != changeDeleted
}

func (element *VNode) Attribute(key, value string) INode {
	if len(key) <= 0 {
		return element
	}
	if setEntry(element.attributes, key, value) {
		element.setDirty(flagAttributes)
	}
	return element
}

func (element *VNode) RemoveAttribute(key string) INode {
	if removeEntry(element.attributes, key) {
		element.setDirty(flagAttributes)
	}
	return element
}

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

func (element *VNode) HaveAttribute(key, value string) bool {
	v, ok := element.attributes[key]
	return ok && v.status != changeDeleted && v.Value() == value
}

func (element *VNode) Style(key, value string) INode {
	if setEntry(element.styles, key, value) {
		element.setDirty(flagStyles)
	}
	return element
}

func (element *VNode) RemoveStyle(key string) INode {
	if removeEntry(element.styles, key) {
		element.setDirty(flagStyles)
	}
	return element
}

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

func (element *VNode) HaveStyle(key, value string) bool {
	v, ok := element.styles[key]
	return ok && v.status != changeDeleted && v.Value() == value
}

func (element *VNode) Id(id string) INode {
	if element.id.assign(id, changeModified) {
		element.mark()
	}
	return element
}

func (element *InputVNode) BindOnChange(signal Settable[string]) *InputVNode {
	element.On(EventChange, func(ctx EventContext) {
		signal.Set(ctx.Event.Target().Underlying().Get("value").String())
	})
	return element
}

func (element *InputVNode) OnKeyUp(handler func(ctx EventContext)) *InputVNode {
	element.On(EventKeyUp, handler)
	return element
}

func (e *VNode) Src(v string) INode { return e.Attribute("src", v) }

type AVNode struct {
	VNode
}

func newA(element *VNode) *AVNode {
	a := &AVNode{
		*element,
	}
	a.Owner = a
	return a
}

func (e *AVNode) Href(v string) *AVNode {
	e.Attribute("href", v)
	return e
}

type InputVNode struct {
	VNode
}

func asInput(node *VNode) *InputVNode {
	input := &InputVNode{*node}
	input.Owner = input
	return input
}

func (element *InputVNode) Value(t string) INode {
	if element.value.assign(t, changeModified) {
		element.mark()
	}
	return element
}

func (element *InputVNode) BindOnInput(signal Settable[string]) *InputVNode {
	element.On(EventInput, func(ctx EventContext) {
		signal.Set(ctx.Event.Target().Underlying().Get("value").String())
		element.scheludeRender()
	})
	return element
}

func (element *InputVNode) BindValue(signal Gettable[string]) *InputVNode {
	element.bind(func() {
		element.Value(signal.Get())
	})
	return element
}

func (e *InputVNode) Placeholder(v string) *InputVNode {
	e.Attribute("placeholder", v)
	return e
}

func (e *InputVNode) Type(v string) *InputVNode {
	e.Attribute("type", v)
	return e
}

type TextAreaNode struct {
	VNode
}

func asTextArea(node *VNode) *TextAreaNode {
	textArea := &TextAreaNode{
		*node,
	}
	textArea.Owner = textArea
	return textArea
}

func (element *TextAreaNode) Value(t string) INode {
	if element.value.assign(t, changeModified) {
		element.mark()
	}
	return element
}

func (element *TextAreaNode) BindValue(signal Gettable[string]) *TextAreaNode {
	element.bind(func() {
		v := signal.Get()
		element.Value(v)
		element.Text(v)
	})
	return element
}

type OptionNode struct {
	VNode
}

func asOption(node *VNode) *OptionNode {
	option := &OptionNode{
		*node,
	}
	option.Owner = option
	return option
}

func (element *OptionNode) Value(t string) *OptionNode {
	if element.value.assign(t, changeModified) {
		element.mark()
	}
	return element
}

func (element *OptionNode) Selected() *OptionNode {
	element.Attribute("selected", "selected")
	element.mark()
	return element
}

const noopIdNode string = "noop"

type NoopNode struct {
	VNode
}

func asNoop(node *VNode) *NoopNode {
	noop := &NoopNode{
		*node,
	}
	noop.Owner = noop
	return noop
}

func (nop *NoopNode) Id(id string) INode {
	return nop
}
func (nop *NoopNode) Text(t string) INode {
	return nop
}
func (nop *NoopNode) BindText(signal Gettable[string]) INode {
	return nop
}
func (nop *NoopNode) Body(childs ...INode) INode {
	nop.VNode.BodyList(childs)
	return nop
}
func (nop *NoopNode) BodyList(childs []INode) INode {
	nop.VNode.BodyList(childs)
	return nop
}
func (nop *NoopNode) Attribute(key, value string) INode {
	return nop
}
func (nop *NoopNode) RemoveAttribute(key string) INode {
	return nop
}
func (nop *NoopNode) Style(key, value string) INode {
	return nop
}
func (nop *NoopNode) RemoveStyle(key string) INode {
	return nop
}
func (nop *NoopNode) Class(classes ...string) INode {
	return nop
}
func (nop *NoopNode) RemoveClass(classes ...string) INode {
	return nop
}
func (nop *NoopNode) On(event Event, handler func(ctx EventContext)) INode {
	return nop
}
func (nop *NoopNode) OnClick(handler func(ctx EventContext)) INode {
	return nop
}

func asVNode(i INode) *VNode {
	return i.AsVNode()
}

func H1() *VNode { return newVNode("H1") }
func H2() *VNode { return newVNode("H2") }
func H3() *VNode { return newVNode("H3") }
func H4() *VNode { return newVNode("H4") }
func H5() *VNode { return newVNode("H5") }
func H6() *VNode { return newVNode("H6") }

func Div() *VNode { return newVNode("DIV") }

func P() *VNode { return newVNode("P") }

func Button() *VNode { return newVNode("BUTTON") }

func I() *VNode { return newVNode("I") }

func A() *AVNode              { return newA(newVNode("A")) }
func Span() *VNode            { return newVNode("SPAN") }
func Strong() *VNode          { return newVNode("STRONG") }
func Em() *VNode              { return newVNode("EM") }
func Small() *VNode           { return newVNode("SMALL") }
func Img() *VNode             { return newVNode("IMG") }
func Input() *InputVNode      { return asInput(newVNode("INPUT")) }
func Label() *VNode           { return newVNode("LABEL") }
func Form() *VNode            { return newVNode("FORM") }
func Select() *VNode          { return newVNode("SELECT") }
func Option() *OptionNode     { return asOption(newVNode("OPTION")) }
func TextArea() *TextAreaNode { return asTextArea(newVNode("TEXTAREA")) }

func Ul() *VNode { return newVNode("UL") }
func Ol() *VNode { return newVNode("OL") }
func Li() *VNode { return newVNode("LI") }

func Table() *VNode { return newVNode("TABLE") }
func THead() *VNode { return newVNode("THEAD") }
func TBody() *VNode { return newVNode("TBODY") }
func TFoot() *VNode { return newVNode("TFOOT") }
func Tr() *VNode    { return newVNode("TR") }
func Th() *VNode    { return newVNode("TH") }
func Td() *VNode    { return newVNode("TD") }

func Nav() *VNode     { return newVNode("NAV") }
func Header() *VNode  { return newVNode("HEADER") }
func Footer() *VNode  { return newVNode("FOOTER") }
func Main() *VNode    { return newVNode("MAIN") }
func Section() *VNode { return newVNode("SECTION") }
func Article() *VNode { return newVNode("ARTICLE") }
func Aside() *VNode   { return newVNode("ASIDE") }

func Video() *VNode  { return newVNode("VIDEO") }
func Audio() *VNode  { return newVNode("AUDIO") }
func Source() *VNode { return newVNode("SOURCE") }

func Canvas() *VNode { return newVNode("CANVAS") }
func Svg() *VNode    { return newVNodeNS("svg", svgNamespace) }
func Path() *VNode   { return newVNodeNS("path", svgNamespace) }

func Br() *VNode { return newVNode("BR") }
func Hr() *VNode { return newVNode("HR") }

func Code() *VNode { return newVNode("CODE") }
func Pre() *VNode  { return newVNode("PRE") }

func Noop() *NoopNode { return asNoop(newVNode(noopIdNode)) }
