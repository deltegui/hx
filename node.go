package hx

// INode is a node of the tree. Every method returns INode, so calls can be
// chained in any order. The node is made by the engine in use (see Engine): the
// browser engine keeps what it needs to update the DOM and the server engine
// keeps what it needs to write HTML. Code that builds a UI only sees this
// interface, so a library of components works with both.
type INode interface {
	// Content
	Body(children ...INode) INode
	BodyList(children []INode) INode
	Text(text string) INode
	Textf(format string, args ...any) INode
	BindText(signal Gettable[string]) INode

	// Attributes
	Id(id string) INode
	Attribute(key, value string) INode
	RemoveAttribute(key string) INode
	GetAttribute(key string) string
	HaveAttribute(key, value string) bool
	Src(value string) INode
	Href(value string) INode
	Placeholder(value string) INode
	Type(value string) INode
	Selected() INode

	// Classes and styles
	Class(classes ...string) INode
	RemoveClass(classes ...string) INode
	ClassIf(class string, on bool) INode
	HaveClass(class string) bool
	Style(key, value string) INode
	RemoveStyle(key string) INode
	GetStyle(key string) string
	HaveStyle(key, value string) bool

	// Form controls
	Value(value string) INode
	Checked(on bool) INode
	BindValue(signal Gettable[string]) INode
	BindChecked(signal Gettable[bool]) INode
	BindOnInput(signal Settable[string]) INode
	BindOnChange(signal Settable[string]) INode
	BindOnChecked(signal Settable[bool]) INode

	// Events. On the server they do nothing.
	On(event Event, handler func(ctx EventContext)) INode
	OnClick(handler func(ctx EventContext)) INode
	OnInput(handler func(ctx EventContext)) INode
	OnChange(handler func(ctx EventContext)) INode
	OnKeyUp(handler func(ctx EventContext)) INode
	OnKeyDown(handler func(ctx EventContext)) INode
	OnSubmit(handler func(ctx EventContext)) INode
	OnFocus(handler func(ctx EventContext)) INode
	OnBlur(handler func(ctx EventContext)) INode

	// Life of the node
	Effect(fn func()) INode
	OnRendered(fn func()) INode
	OnRemoved(fn func()) INode
}

// EventContext is what an event handler receives: the node that has the listener and the browser event.
type EventContext struct {
	Target INode
	// Event is the event of the browser, and nil on a server. In browser code use
	// wasm.EventOf(ctx) to get it as a js.Value (see also wasm.PreventDefault).
	Event any
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
