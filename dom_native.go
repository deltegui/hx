//go:build !(js && wasm)

package hx

// The browser parts of the library do not exist outside WASM. This file gives
// the shared code empty versions of them, so the rest builds on a server.

type domEvent = any

type domState struct{}

type domListener struct{}

// IsServer tells if the code runs on a server (true) or in the browser (false).
// Use it in code shared by both to skip what only makes sense in the browser.
func IsServer() bool { return true }

func (element *VNode) releaseListeners() {}

func (element *VNode) reviveListeners() {}

func targetValue(ctx EventContext) string { return "" }

func targetChecked(ctx EventContext) bool { return false }
