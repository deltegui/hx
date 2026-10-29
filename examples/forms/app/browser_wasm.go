//go:build js && wasm

package app

import "github.com/deltegui/hx"

// What follows needs the browser, so on the server it is an empty function.

// preventDefault stops the browser from sending the form by itself.
func preventDefault(ctx hx.EventContext) { ctx.Event.PreventDefault() }

// resetForm puts the form back as it was when the page loaded, and empties the file inputs.
func resetForm(form *hx.VNode) {
	if element := form.Underlying(); element != nil {
		element.Underlying().Call("reset")
	}
}
