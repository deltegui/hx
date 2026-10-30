//go:build !(js && wasm)

package wasm

import "github.com/deltegui/hx"

// The browser parts of the library do not exist outside WASM. This file gives
// the shared code empty versions of them, so the rest builds on a server.

type domState struct{}

type domListener struct{}

func isServer() bool { return true }

func (element *VNode) releaseListeners() {}

func (element *VNode) reviveListeners() {}

func targetValue(ctx hx.EventContext) string { return "" }

func targetChecked(ctx hx.EventContext) bool { return false }

func (element *VNode) queueRendered() {}
