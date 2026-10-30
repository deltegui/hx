//go:build js && wasm

package wasm

import (
	"fmt"
	"github.com/deltegui/hx"
	"log"
	"strings"
	"syscall/js"

	"honnef.co/go/js/dom/v2"
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
// Hydrate creates a root for the UI. The returned function stops everything
// that the UI started.
func Hydrate(id string, build func() hx.INode) (dispose func()) {
	mount := dom.GetWindow().Document().GetElementByID(id)
	if mount == nil {
		log.Printf("hx: Hydrate: there is no element with id %q", id)
		return func() {}
	}
	return HydrateElement(mount, build)
}

// HydrateElement is Hydrate for an element you already have.
func HydrateElement(mount dom.Element, build func() hx.INode) (dispose func()) {
	previousMode := currentMountMode
	currentMountMode = mountCollect
	mountQueue = nil
	defer func() { currentMountMode = previousMode }()

	hx.CreateRoot(func(rootDispose func()) struct{} {
		dispose = rootDispose

		root := New(mount)
		root.Body(build())

		h := &hydrator{renderer: root.renderer.(*DiffRenderer)}
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
		return struct{}{}
	})
	return dispose
}

// ReadState returns the text of the element with that id, or an empty string.
// It reads what the server wrote with StateScript.
func ReadState(id string) string {
	element := dom.GetWindow().Document().GetElementByID(id)
	if element == nil {
		return ""
	}
	return element.TextContent()
}

type hydrator struct {
	renderer   *DiffRenderer
	mismatches int
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
	raw := owner.domElement.Underlying()
	h.adoptText(owner, raw)

	tops := topNodes(owner, nil)
	kids := raw.Get("children")
	count := kids.Length()
	if len(tops) != count {
		h.warn(owner, fmt.Sprintf("the page has %d child elements and the UI has %d", count, len(tops)))
		for i := count - 1; i >= 0; i-- {
			removeRaw(kids.Index(i))
		}
		for _, top := range tops {
			h.resetFresh(top)
		}
		if len(tops) > 0 {
			owner.setDirty(flagChildren)
		}
		return true
	}
	for i, top := range tops {
		if !h.adoptNode(top, kids.Index(i)) {
			rebuilt = true
		}
	}
	return rebuilt
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
	if name := raw.Get("nodeName").String(); !strings.EqualFold(name, node.tag) {
		h.warn(node, fmt.Sprintf("the page has <%s>", strings.ToLower(name)))
		removeRaw(raw)
		h.resetFresh(node)
		if owner := domParentOf(node); owner != nil {
			owner.setDirty(flagChildren)
		}
		return false
	}

	node.domElement = wrapExisting(raw, node.tag, node.namespace)
	node.haveDomElement = true
	node.status = unchanged
	if differences := node.commit(raw); len(differences) > 0 {
		h.fixed(node, differences)
	}

	rebuilt := h.adoptChildren(node)
	h.finish(node, rebuilt)
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
	node.domElement = nil
	node.haveDomElement = false
	node.textNode = js.Undefined()

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
