//go:build js && wasm

package wasm

import (
	"syscall/js"

	"github.com/deltegui/hx"
)

// What follows is the part of the library that exists only in the browser.
// The rest (signals, nodes, StringRenderer, RenderToString) also builds for a
// server.

// domState is the DOM side of a node.
type domState struct {
	domElement     js.Value // the DOM element, once it exists (see haveDomElement)
	haveDomElement bool
	textNode       js.Value
	// The markers of a raw node, once its HTML is in the DOM (or in a fragment,
	// before it is placed). Undefined until then.
	rawStart, rawEnd js.Value
}

// hasDom tells if the node has something in the DOM that its owner has to place:
// an element, or the nodes of a raw node.
func (element *VNode) hasDom() bool {
	return element.haveDomElement || (element.tag == rawIdNode && !element.rawStart.IsUndefined())
}

// anchor is the DOM node that stands for the node when its owner puts its
// children in order: its element, or the first marker of a raw node.
func (element *VNode) anchor() js.Value {
	if element.tag == rawIdNode {
		return element.rawStart
	}
	return element.domElement
}

// moveBefore puts everything the node has in the DOM inside parent, before next.
func (element *VNode) moveBefore(parent, next js.Value) {
	if element.tag == rawIdNode {
		moveRawRange(element.rawStart, element.rawEnd, parent, next)
		return
	}
	parent.Call("insertBefore", element.domElement, next)
}

// domListener is the DOM side of an event listener.
type domListener struct {
	wrapper js.Func
}

func isServer() bool { return false }

// NewWithRenderer mounts a node on an existing DOM element with a custom Renderer.
func NewWithRenderer(element js.Value, renderer Renderer) *VNode {
	VNode := newVNode(element.Get("nodeName").String())
	VNode.status = unchanged
	VNode.domElement = element
	VNode.haveDomElement = true
	VNode.renderer = renderer
	VNode.haveRenderer = true
	return VNode
}

// New mounts a node on an existing DOM element. Its children are rendered inside the element.
func New(element js.Value) *VNode {
	return NewWithRenderer(element, newRenderer(element))
}

// elementByID is the element with that id, or an undefined value if the page has none.
func elementByID(id string) js.Value {
	element := rawDocument().Call("getElementById", id)
	if element.IsNull() {
		return js.Undefined()
	}
	return element
}

// NewFromId mounts a node on the DOM element with that id. It panics if the page has none.
func NewFromId(id string) *VNode {
	return New(mustElementByID(id))
}

// NewFromIdWithRenderer is NewFromId with a custom Renderer.
func NewFromIdWithRenderer(id string, renderer Renderer) *VNode {
	return NewWithRenderer(mustElementByID(id), renderer)
}

func mustElementByID(id string) js.Value {
	element := elementByID(id)
	if element.IsUndefined() {
		panic("hx: there is no element with id " + id)
	}
	return element
}

// Underlying returns the DOM element, once it has been rendered, as a js.Value.
// Before that it is undefined (IsUndefined is true).
func (element *VNode) Underlying() js.Value {
	return element.domElement
}

func targetValue(ctx hx.EventContext) string {
	return EventOf(ctx).Get("target").Get("value").String()
}

func targetChecked(ctx hx.EventContext) bool {
	return EventOf(ctx).Get("target").Get("checked").Truthy()
}

// queueRendered asks the renderer to run the OnRendered functions of the node
// once the frame is done. Asking twice does no harm: it runs once per time the
// node is in the page.
func (element *VNode) queueRendered() {
	if len(element.onRendered) == 0 || element.attached || !element.haveDomElement {
		return
	}
	if renderer, ok := element.renderer.(*DiffRenderer); ok {
		renderer.rendered = append(renderer.rendered, element)
		renderer.ScheduleRender()
	}
}

// Underlying returns the DOM element of a node as a js.Value, once it has been
// rendered. Before that, and for a node of another engine, it is undefined
// (IsUndefined is true).
func Underlying(node hx.INode) js.Value {
	if v, ok := node.(*VNode); ok {
		return v.domElement
	}
	return js.Undefined()
}

// EventOf returns the event of the browser that a handler received, as a js.Value.
// It is undefined if the handler did not come from the browser.
func EventOf(ctx hx.EventContext) js.Value {
	if event, ok := ctx.Event.(js.Value); ok {
		return event
	}
	return js.Undefined()
}

// Key is the key of a keyboard event ("Enter", "a", "ArrowDown"...), or an empty
// string if the event has none.
func Key(ctx hx.EventContext) string {
	if event := EventOf(ctx); !event.IsUndefined() {
		if key := event.Get("key"); key.Type() == js.TypeString {
			return key.String()
		}
	}
	return ""
}

// PreventDefault stops the browser from doing what it does by itself with the
// event, for example sending a form.
func PreventDefault(ctx hx.EventContext) {
	if event := EventOf(ctx); !event.IsUndefined() {
		event.Call("preventDefault")
	}
}

// StopPropagation stops the event from reaching the elements around the one that has the listener.
func StopPropagation(ctx hx.EventContext) {
	if event := EventOf(ctx); !event.IsUndefined() {
		event.Call("stopPropagation")
	}
}
