//go:build js && wasm

package app

import (
	"github.com/deltegui/hx"
	"honnef.co/go/js/dom/v2"
)

// isEnter tells if the event is the Enter key. It needs the browser event, so it
// only exists in the browser; on the server no event ever happens.
func isEnter(ctx hx.EventContext) bool {
	key, ok := ctx.Event.(*dom.KeyboardEvent)
	return ok && key.Key() == "Enter"
}
