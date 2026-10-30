//go:build js && wasm

package app

import (
	"github.com/deltegui/hx"
	"github.com/deltegui/hx/wasm"
)

// preventDefault stops the browser from sending the form by itself.
func preventDefault(ctx hx.EventContext) { wasm.EventOf(ctx).PreventDefault() }
