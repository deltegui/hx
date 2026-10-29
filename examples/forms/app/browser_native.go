//go:build !(js && wasm)

package app

import "github.com/deltegui/hx"

func preventDefault(hx.EventContext) {}

func resetForm(*hx.VNode) {}
