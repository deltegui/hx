//go:build js && wasm

package hx

import (
	"strings"
	"syscall/js"

	"honnef.co/go/js/dom/v2"
)

// honnef.co/go/js/dom wraps every DOM value it returns by comparing its
// constructor with dozens of globals, which costs many Go <-> JS calls per
// element. The functions in this file talk to the DOM directly on the hot
// paths (creating and removing nodes), which is more than 10x faster.

// typedTags are created through honnef so Underlying() returns the specific
// type (for example *dom.HTMLInputElement) that callers may want to assert.
var typedTags = map[string]bool{
	"A": true, "AUDIO": true, "BUTTON": true, "CANVAS": true, "FORM": true,
	"IMG": true, "INPUT": true, "LABEL": true, "OPTION": true, "SELECT": true,
	"TEXTAREA": true, "VIDEO": true,
}

var documentWrapper dom.Document

func domDocument() dom.Document {
	if documentWrapper == nil {
		documentWrapper = dom.GetWindow().Document()
	}
	return documentWrapper
}

func rawDocument() js.Value { return js.Global().Get("document") }

func newDomElement(tag, namespace string) dom.Element {
	if namespace != "" {
		raw := rawDocument().Call("createElementNS", namespace, tag)
		return &dom.BasicElement{BasicNode: &dom.BasicNode{Value: raw}}
	}
	if typedTags[tag] {
		return domDocument().CreateElement(tag)
	}
	raw := rawDocument().Call("createElement", tag)
	return &dom.BasicHTMLElement{BasicElement: &dom.BasicElement{BasicNode: &dom.BasicNode{Value: raw}}}
}

// wrapExisting wraps a DOM element that already exists (the page was rendered by a server).
// It gives the same type newDomElement would have given.
func wrapExisting(raw js.Value, tag, namespace string) dom.Element {
	if namespace != "" {
		return &dom.BasicElement{BasicNode: &dom.BasicNode{Value: raw}}
	}
	if typedTags[strings.ToUpper(tag)] {
		return dom.WrapElement(raw)
	}
	return &dom.BasicHTMLElement{BasicElement: &dom.BasicElement{BasicNode: &dom.BasicNode{Value: raw}}}
}

// removeFromParent removes the element from its parent, if it has one.
func removeFromParent(element dom.Element) {
	raw := element.Underlying()
	if parent := raw.Get("parentNode"); !parent.IsNull() {
		parent.Call("removeChild", raw)
	}
}

// prependText inserts a new text node as the first child of element and returns it.
func prependText(element dom.Element, text string) js.Value {
	parent := element.Underlying()
	node := rawDocument().Call("createTextNode", text)
	parent.Call("insertBefore", node, parent.Get("firstChild"))
	return node
}
