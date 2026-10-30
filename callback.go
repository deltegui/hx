package hx

import "sync"

var (
	callbackMu    sync.Mutex
	callbackDepth int
)

// RunCallback runs fn marked as running inside a callback that JavaScript made:
// an event handler, an animation frame... The engine of the browser does it for
// every handler and render it runs, and you can do it for a callback of your own
// (the body of a js.FuncOf).
//
// Code in a callback must not wait for something that only JavaScript can
// deliver, such as the answer of a request: the page cannot deliver it until the
// callback returns, and Go stops with a deadlock. hxhttp.Do checks InCallback
// and fails with an error instead.
func RunCallback(fn func()) {
	callbackMu.Lock()
	callbackDepth++
	callbackMu.Unlock()
	defer func() {
		callbackMu.Lock()
		callbackDepth--
		callbackMu.Unlock()
	}()
	fn()
}

// InCallback tells if the code is running inside RunCallback.
func InCallback() bool {
	callbackMu.Lock()
	defer callbackMu.Unlock()
	return callbackDepth > 0
}
