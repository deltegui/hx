//go:build js && wasm

package wasm

import (
	"fmt"
	"github.com/deltegui/hx"
	"log"
	"strings"
	"syscall/js"
)

// Hydrate brings to life a page whose HTML was written by RenderToString.
//
// It calls build to create the UI, as it would for a normal render, but instead
// of creating DOM elements it adopts the ones already in the element with that
// id: each node takes the element that corresponds to it, the text, attributes
// and classes that already match are not written again, and the event
// listeners are attached. From then on the UI works as if the browser had
// rendered it.
//
// build must create the same UI, with the same initial state, that the server
// rendered. If a part of the page does not match, a message is logged and that
// part is rendered again in the browser. Functions registered with OnMount run
// when the page has been hydrated.
//
// If the page has hx.ReplayScript, the clicks, keys, text typed and changes
// that the user made inside the element before this point are replayed once it
// is hydrated: each handler runs with the event, in order, as if the page had
// been alive.
//
// Hydrate creates a root for the UI. The returned function stops everything
// that the UI started.
func Hydrate(id string, build func() hx.INode) (dispose func()) {
	mount := elementByID(id)
	if mount.IsUndefined() {
		log.Printf("hx: Hydrate: there is no element with id %q", id)
		return func() {}
	}
	return HydrateElement(mount, build)
}

// HydrateElement is Hydrate for an element you already have.
func HydrateElement(mount js.Value, build func() hx.INode) (dispose func()) {
	previousMode := currentMountMode
	currentMountMode = mountCollect
	mountQueue = nil
	defer func() { currentMountMode = previousMode }()

	// What the user did while the WASM was loading (see hx.ReplayScript).
	events := takeReplay(mount)

	hx.CreateRoot(func(rootDispose func()) struct{} {
		dispose = rootDispose

		root := New(mount)
		root.Body(build())

		h := &hydrator{renderer: root.renderer.(*DiffRenderer), replaying: len(events) > 0}
		rebuilt := h.adoptChildren(root)
		h.finish(root, rebuilt)
		h.renderer.flushRendered()

		js.Global().Set("__hxHydrationMismatches", h.mismatches)

		queue := mountQueue
		mountQueue = nil
		currentMountMode = mountNow
		for _, fn := range queue {
			fn()
		}
		h.replay(events)
		return struct{}{}
	})
	return dispose
}

// ReadState returns the text of the element with that id, or an empty string.
// It reads what the server wrote with StateScript.
func ReadState(id string) string {
	element := elementByID(id)
	if element.IsUndefined() {
		return ""
	}
	return element.Get("textContent").String()
}

type hydrator struct {
	renderer   *DiffRenderer
	mismatches int

	replaying bool     // there are events to replay
	listening []*VNode // the adopted nodes with listeners, found again by replay
}

func (h *hydrator) warn(node *VNode, reason string) {
	h.mismatches++
	log.Printf("hx: hydration mismatch in <%s>: %s. That part is rendered again in the browser.", strings.ToLower(node.tag), reason)
}

// fixed reports a difference that the browser repairs without rendering the node again.
func (h *hydrator) fixed(node *VNode, reasons []string) {
	h.mismatches++
	log.Printf("hx: hydration mismatch in <%s>: %s. The browser sets it.", strings.ToLower(node.tag), strings.Join(reasons, "; "))
}

// topNodes returns the nodes that put an element directly inside the DOM
// element of owner. Noop children are transparent.
func topNodes(owner *VNode, tops []*VNode) []*VNode {
	for _, child := range owner.children {
		if child == nil {
			continue
		}
		if child.tag == noopIdNode {
			tops = topNodes(child, tops)
		} else {
			tops = append(tops, child)
		}
	}
	return tops
}

// adoptChildren pairs the children of owner with the child elements of its DOM
// element, one by one. It returns true if some part had to be rendered again.
func (h *hydrator) adoptChildren(owner *VNode) (rebuilt bool) {
	raw := owner.domElement
	h.adoptText(owner, raw)

	tops := topNodes(owner, nil)
	kids, closed := slots(raw)
	count := len(kids)
	if !closed || len(tops) != count {
		if !closed {
			h.warn(owner, "the raw HTML in the page is not closed")
		} else {
			h.warn(owner, fmt.Sprintf("the page has %d child elements and the UI has %d", count, len(tops)))
		}
		clearChildren(raw, owner.textNode)
		for _, top := range tops {
			h.resetFresh(top)
		}
		if len(tops) > 0 {
			owner.setDirty(flagChildren)
		}
		return true
	}
	for i, top := range tops {
		if !h.adoptNode(top, kids[i]) {
			rebuilt = true
		}
	}
	return rebuilt
}

// isRawMarker tells if a DOM node is the comment that opens (or closes) raw HTML.
func isRawMarker(node js.Value, text string) bool {
	return node.Get("nodeType").Int() == 8 && node.Get("data").String() == text
}

// clearChildren removes everything inside an element except keep, which is its
// own text node if it has taken one. It leaves the element ready to be filled
// again, with no stray text from raw HTML whose markers are not trusted.
func clearChildren(raw, keep js.Value) {
	for node := raw.Get("firstChild"); !node.IsNull(); {
		next := node.Get("nextSibling")
		if keep.IsUndefined() || keep.IsNull() || !node.Equal(keep) {
			raw.Call("removeChild", node)
		}
		node = next
	}
}

// slots lists what the server put inside an element, one per node of the tree
// that has something in the DOM: each child element, and each run of raw HTML,
// which the comment that opens it stands for. Text and other comments are not
// counted, as before. closed is false if a run of raw HTML is never closed.
func slots(raw js.Value) (found []js.Value, closed bool) {
	found = []js.Value{}
	for node := raw.Get("firstChild"); !node.IsNull(); node = node.Get("nextSibling") {
		switch node.Get("nodeType").Int() {
		case 1:
			found = append(found, node)
		case 8:
			if isRawMarker(node, rawStartMarker) {
				found = append(found, node)
				for !node.IsNull() && !isRawMarker(node, rawEndMarker) {
					node = node.Get("nextSibling")
				}
				if node.IsNull() {
					return found, false
				}
			}
		}
	}
	return found, true
}

// rawEndOf finds the comment that closes the raw HTML that start opens, or
// null if the page does not have it.
func rawEndOf(start js.Value) js.Value {
	node := start
	for !node.IsNull() && !isRawMarker(node, rawEndMarker) {
		node = node.Get("nextSibling")
	}
	return node
}

// removeSlot removes a slot: an element, or a whole run of raw HTML.
func removeSlot(slot js.Value) {
	if slot.Get("nodeType").Int() == 8 {
		if end := rawEndOf(slot); !end.IsNull() {
			removeRawRange(slot, end)
			return
		}
	}
	removeRaw(slot)
}

func removeRaw(raw js.Value) {
	if parent := raw.Get("parentNode"); !parent.IsNull() {
		parent.Call("removeChild", raw)
	}
}

// adoptText takes the text node of owner, or leaves the text pending so the renderer writes it.
func (h *hydrator) adoptText(owner *VNode, raw js.Value) {
	desired := owner.text.nextValue
	first := raw.Get("firstChild")
	hasText := !first.IsNull() && first.Get("nodeType").Int() == 3

	if desired == "" {
		if hasText && strings.TrimSpace(first.Get("nodeValue").String()) != "" {
			h.warn(owner, "the page has text that the UI does not have")
			removeRaw(first)
		}
		return
	}
	if !hasText {
		h.fixed(owner, []string{"the page has no text where the UI has some"})
		return // the text stays pending, the renderer creates the text node
	}
	owner.textNode = first
	if first.Get("nodeValue").String() != desired {
		h.fixed(owner, []string{"the text is different"})
		return // pending, the renderer writes it
	}
	owner.text.tick()
}

// adoptNode makes node take the element raw. If they do not match, the element
// is removed and node is left to be rendered by the browser, and it returns false.
func (h *hydrator) adoptNode(node *VNode, raw js.Value) bool {
	if node.tag == rawIdNode {
		return h.adoptRaw(node, raw)
	}
	if name := raw.Get("nodeName").String(); !strings.EqualFold(name, node.tag) {
		h.warn(node, fmt.Sprintf("the page has <%s>", strings.ToLower(name)))
		removeSlot(raw)
		h.resetFresh(node)
		if owner := domParentOf(node); owner != nil {
			owner.setDirty(flagChildren)
		}
		return false
	}

	node.domElement = raw
	node.haveDomElement = true
	node.status = unchanged
	h.listen(node, raw)
	if differences := node.commit(raw); len(differences) > 0 {
		h.fixed(node, differences)
	}

	rebuilt := h.adoptChildren(node)
	h.finish(node, rebuilt)
	return true
}

// adoptRaw makes a raw node take the run of HTML that starts at the comment
// start. What is inside is not looked at: it is trusted, and the browser cannot
// compare it with the string, since it writes the same markup in its own way.
func (h *hydrator) adoptRaw(node *VNode, start js.Value) bool {
	end := js.Null()
	if isRawMarker(start, rawStartMarker) {
		end = rawEndOf(start)
	}
	if end.IsNull() {
		h.warn(node, "the page has no raw HTML where the UI has some")
		removeSlot(start)
		h.resetFresh(node)
		if owner := domParentOf(node); owner != nil {
			owner.setDirty(flagChildren)
		}
		return false
	}
	node.rawStart, node.rawEnd = start, end
	node.status = unchanged
	return true
}

// commit records that what is pending in the node is already in the DOM element.
// What the element does not have is left pending, so the renderer sets it, and
// it is returned.
func (node *VNode) commit(raw js.Value) (differences []string) {
	if id := node.id.nextValue; id != "" && raw.Get("id").String() != id {
		differences = append(differences, fmt.Sprintf("the id should be %q", id))
	} else {
		node.id.tick()
	}
	node.value.tick()
	if raw.Get("checked").Truthy() != node.checked.nextValue {
		differences = append(differences, fmt.Sprintf("it should be checked=%v", node.checked.nextValue))
	} else {
		node.checked.tick()
	}

	for _, key := range sortedKeys(node.attributes) {
		attr := node.attributes[key]
		if attr.status == changeDeleted {
			delete(node.attributes, key)
			continue
		}
		if got := raw.Call("getAttribute", key); got.IsNull() || got.String() != attr.nextValue {
			differences = append(differences, fmt.Sprintf("the attribute %q should be %q", key, attr.nextValue))
			continue
		}
		attr.tick()
		node.attributes[key] = attr
	}
	for key, style := range node.styles {
		if style.status == changeDeleted {
			delete(node.styles, key)
			continue
		}
		style.tick()
		node.styles[key] = style
	}
	classList := raw.Get("classList")
	for _, class := range sortedKeys(node.classes) {
		if node.classes[class] == changeDeleted {
			delete(node.classes, class)
			continue
		}
		if !classList.Call("contains", class).Bool() {
			differences = append(differences, fmt.Sprintf("the class %q is missing", class))
			continue
		}
		node.classes[class] = unchanged
	}
	return differences
}

// finish attaches the listeners of an adopted node and drops its pending work.
func (h *hydrator) finish(node *VNode, rebuilt bool) {
	node.dirtyFlags[flagChildren] = false
	node.render() // attaches the event listeners
	if rebuilt {
		// something below has to be created, so the children must be put in order
		node.setDirty(flagChildren)
		return
	}
	delete(h.renderer.markNodes, node)
}

// resetFresh makes node, and everything below, look like a node that has just
// been created, so the renderer creates its DOM elements.
func (h *hydrator) resetFresh(node *VNode) {
	node.status = changeNew
	node.domElement = js.Undefined()
	node.haveDomElement = false
	node.textNode = js.Undefined()
	node.rawStart, node.rawEnd = js.Undefined(), js.Undefined()

	pending := func(v diffValue[string]) diffValue[string] {
		if v.nextValue == "" {
			return diffValue[string]{}
		}
		return diffValue[string]{nextValue: v.nextValue, status: changeModified}
	}
	node.id = pending(node.id)
	node.text = pending(node.text)
	node.value = pending(node.value)
	if node.checked.nextValue {
		node.checked = diffValue[bool]{nextValue: true, status: changeModified}
	} else {
		node.checked = diffValue[bool]{}
	}

	for key, attr := range node.attributes {
		if attr.status == changeDeleted {
			delete(node.attributes, key)
			continue
		}
		node.attributes[key] = diffValue[string]{nextValue: attr.nextValue, status: changeNew}
	}
	for key, style := range node.styles {
		if style.status == changeDeleted {
			delete(node.styles, key)
			continue
		}
		node.styles[key] = diffValue[string]{nextValue: style.nextValue, status: changeNew}
	}
	for class, status := range node.classes {
		if status == changeDeleted {
			delete(node.classes, class)
		} else {
			node.classes[class] = changeNew
		}
	}
	for _, listener := range node.eventListeners {
		listener.attached = false
		listener.wrapper = js.Func{}
	}

	for i := range node.dirtyFlags {
		node.dirtyFlags[i] = true
	}
	h.renderer.Mark(node)
	h.renderer.ScheduleRender()

	for _, child := range node.children {
		if child != nil {
			h.resetFresh(child)
		}
	}
}
