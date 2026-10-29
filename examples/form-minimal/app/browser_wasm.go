//go:build js && wasm

package app

import "github.com/deltegui/hx"

// preventDefault stops the browser from sending the form by itself.
func preventDefault(ctx hx.EventContext) { ctx.Event.PreventDefault() }
