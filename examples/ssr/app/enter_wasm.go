//go:build js && wasm

package app

import (
	"github.com/deltegui/hx"
	"github.com/deltegui/hx/wasm"
)

// isEnter tells if the event is the Enter key. It needs the browser event, so it
// only exists in the browser; on the server no event ever happens.
func isEnter(ctx hx.EventContext) bool {
	event := wasm.EventOf(ctx)
	return !event.IsUndefined() && event.Get("key").String() == "Enter"
}
