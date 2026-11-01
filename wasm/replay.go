//go:build js && wasm

package wasm

import (
	"github.com/deltegui/hx"
	"syscall/js"
)

// replayMaxAge is how old, in milliseconds, a click or a key can be and still
// be replayed. A click from half a minute ago that suddenly does something is
// worse than a lost one. Input and change are state, not actions: they are
// always replayed, and read the value the element has now.
const replayMaxAge = 10000.0

// StopReplay stops recording the events that ReplayScript records. Hydrate
// stops nothing by itself, as a page can have several islands that hydrate at
// different times: call it once the last one has been hydrated. Calling it
// when the page has no ReplayScript does nothing.
func StopReplay() {
	if stop := js.Global().Get("__hxqStop"); stop.Type() == js.TypeFunction {
		stop.Invoke()
	}
}

// takeReplay marks mount as hydrated, so the page stops recording what happens
// inside it, and returns, in order, the recorded events whose target is inside
// it. The events of other parts of the page stay in the queue.
func takeReplay(mount js.Value) []js.Value {
	mount.Set("__hxLive", true)
	queue := js.Global().Get("__hxq")
	if queue.IsUndefined() || queue.IsNull() || queue.Length() == 0 {
		return nil
	}
	all := queue.Call("splice", 0, queue.Length())
	var mine []js.Value
	for i := 0; i < all.Length(); i++ {
		event := all.Index(i)
		if mount.Call("contains", event.Get("target")).Bool() {
			mine = append(mine, event)
		} else {
			queue.Call("push", event)
		}
	}
	return mine
}

// listen remembers that the element of node has to be found again by replay
// (it has listeners). The element carries the position as a property: it is
// only set when there is something to replay, so a page that nobody touched
// pays nothing.
func (h *hydrator) listen(node *VNode, raw js.Value) {
	if h.replaying && len(node.eventListeners) > 0 {
		raw.Set("__hxn", len(h.listening))
		h.listening = append(h.listening, node)
	}
}

// replay fires the handlers for what the user did before the page was alive. Each
// event goes up from its target through the elements that hydration adopted,
// as it would have done, until a handler stops it. An event whose element was
// rendered again (it is not in the page now) or that nobody listens to is dropped.
func (h *hydrator) replay(events []js.Value) {
	now := js.Global().Get("performance").Call("now").Float()
	for _, event := range events {
		kind := hx.Event(event.Get("type").String())
		target := event.Get("target")
		if !target.Get("isConnected").Bool() {
			continue
		}
		if (kind == hx.EventClick || kind == hx.EventKeyDown) && now-event.Get("timeStamp").Float() > replayMaxAge {
			continue
		}
		for el := target; el.Truthy(); el = el.Get("parentNode") {
			index := el.Get("__hxn")
			if index.IsUndefined() {
				continue
			}
			node := h.listening[index.Int()]
			if listener := node.eventListeners[kind]; listener != nil {
				fireReplayed(node, listener, event)
				if event.Get("cancelBubble").Truthy() {
					break
				}
			}
		}
	}
}

// fireReplayed runs a listener the way the DOM listener of a node does.
func fireReplayed(node *VNode, listener *eventListener, event js.Value) {
	hx.RunCallback(func() {
		hx.Untrack(func() {
			listener.fire(hx.EventContext{Target: node, Event: event})
		})
	})
	node.scheludeRender()
}
