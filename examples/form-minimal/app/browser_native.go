//go:build !(js && wasm)

package app

import "github.com/deltegui/hx"

// On the server no event ever happens.
func preventDefault(hx.EventContext) {}
