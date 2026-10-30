//go:build js && wasm

package app

import (
	"github.com/deltegui/hx"
	"github.com/deltegui/hx/wasm"
)

// What follows needs the browser, so on the server it is an empty function.

// preventDefault stops the browser from sending the form by itself.
func preventDefault(ctx hx.EventContext) { wasm.PreventDefault(ctx) }

// resetForm puts the form back as it was when the page loaded, and empties the file inputs.
func resetForm(form hx.INode) {
	if element := wasm.Underlying(form); element.Truthy() {
		element.Call("reset")
	}
}
