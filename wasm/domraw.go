//go:build js && wasm

package wasm

import "syscall/js"

func rawDocument() js.Value { return js.Global().Get("document") }

func newDomElement(tag, namespace string) js.Value {
	if namespace != "" {
		return rawDocument().Call("createElementNS", namespace, tag)
	}
	return rawDocument().Call("createElement", tag)
}

// removeFromParent removes the element from its parent, if it has one.
func removeFromParent(element js.Value) {
	if parent := element.Get("parentNode"); !parent.IsNull() {
		parent.Call("removeChild", element)
	}
}

// prependText inserts a new text node as the first child of element and returns it.
func prependText(element js.Value, text string) js.Value {
	node := rawDocument().Call("createTextNode", text)
	element.Call("insertBefore", node, element.Get("firstChild"))
	return node
}

// newRawRange parses html the way a <template> does, which accepts any markup
// wherever it goes (a <tr> is fine), and returns the two comment markers around
// the nodes. The markers and the nodes are in a fragment until they are placed.
func newRawRange(html string) (start, end js.Value) {
	doc := rawDocument()
	template := doc.Call("createElement", "template")
	template.Set("innerHTML", html)
	fragment := doc.Call("createDocumentFragment")
	start = doc.Call("createComment", rawStartMarker)
	end = doc.Call("createComment", rawEndMarker)
	fragment.Call("appendChild", start)
	fragment.Call("appendChild", template.Get("content"))
	fragment.Call("appendChild", end)
	return start, end
}

// rawRange lists the nodes from start to end, both included, as they are now.
func rawRange(start, end js.Value) []js.Value {
	nodes := []js.Value{}
	for node := start; !node.IsNull() && !node.IsUndefined(); node = node.Get("nextSibling") {
		nodes = append(nodes, node)
		if node.Equal(end) {
			break
		}
	}
	return nodes
}

// moveRawRange puts the nodes of a raw range inside parent, before next.
func moveRawRange(start, end, parent, next js.Value) {
	for _, node := range rawRange(start, end) {
		parent.Call("insertBefore", node, next)
	}
}

// removeRawRange removes the nodes of a raw range from where they are.
func removeRawRange(start, end js.Value) {
	for _, node := range rawRange(start, end) {
		if parent := node.Get("parentNode"); !parent.IsNull() {
			parent.Call("removeChild", node)
		}
	}
}
