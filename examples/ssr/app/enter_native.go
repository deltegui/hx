//go:build !(js && wasm)

package app

import "github.com/deltegui/hx"

func isEnter(hx.EventContext) bool { return false }
