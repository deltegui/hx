//go:build js && wasm

package hx

import (
	"syscall/js"

	"honnef.co/go/js/dom/v2"
)

// What follows is the part of the library that exists only in the browser.
// The rest (signals, nodes, StringRenderer, RenderToString) also builds for a
// server.

type domEvent = dom.Event

// domState is the DOM side of a node.
type domState struct {
	domElement     dom.Element
	haveDomElement bool
	textNode       js.Value
}

// domListener is the DOM side of an event listener.
type domListener struct {
	wrapper js.Func
}

// IsServer tells if the code runs on a server (true) or in the browser (false).
// Use it in code shared by both to skip what only makes sense in the browser.
func IsServer() bool { return false }

// NewWithRenderer mounts a node on an existing DOM element with a custom Renderer.
func NewWithRenderer(element dom.Element, renderer Renderer) *VNode {
	VNode := newVNode(element.NodeName())
	VNode.status = unchanged
	VNode.domElement = element
	VNode.haveDomElement = true
	VNode.renderer = renderer
	VNode.haveRenderer = true
	return VNode
}

// New mounts a node on an existing DOM element. Its children are rendered inside the element.
func New(element dom.Element) *VNode {
	return NewWithRenderer(element, newRenderer(element))
}

// NewFromId mounts a node on the DOM element with that id.
func NewFromId(id string) *VNode {
	mountPoint := dom.GetWindow().Document().GetElementByID(id)
	return New(mountPoint)
}

// NewFromIdWithRenderer is NewFromId with a custom Renderer.
func NewFromIdWithRenderer(id string, renderer Renderer) *VNode {
	mountPoint := dom.GetWindow().Document().GetElementByID(id)
	return NewWithRenderer(mountPoint, renderer)
}

// Underlying returns the DOM element, once it has been rendered.
// Input, textarea, select, option, a, button, form, img, label, canvas, video and audio nodes keep honnef's specific type (for example *dom.HTMLInputElement). Other nodes are a *dom.BasicHTMLElement.
func (element *VNode) Underlying() dom.Element {
	return element.domElement
}

func targetValue(ctx EventContext) string {
	return ctx.Event.Target().Underlying().Get("value").String()
}

func targetChecked(ctx EventContext) bool {
	return ctx.Event.Target().Underlying().Get("checked").Truthy()
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
