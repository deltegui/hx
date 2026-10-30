package hx

// Engine builds the nodes and decides what a render is. Hix has two: hx/wasm
// draws in the browser and hx/ssr writes HTML on a server. The code that builds
// a UI only uses this package, so it works with either. A program imports the
// engine it needs, and the engine registers itself when it is imported.
type Engine interface {
	// Element creates a node. The tag of an HTML element is in upper case, the
	// namespace is empty for HTML.
	Element(tag, namespace string) INode
	// Fragment creates a node that has no element of its own: its children go
	// to its parent.
	Fragment() INode
	// Raw creates a node that has no element of its own and holds HTML that is
	// written as it is. See hx.Raw.
	Raw(html string) INode
	// OnMount is what the function of the same name does in this engine.
	OnMount(fn func())
	// IsServer tells if the code runs on a server.
	IsServer() bool
	// Tracks tells if the engine needs signals to follow who reads them. A
	// server does not: it builds the UI once, so a signal is a plain value there
	// and many UIs can be built at the same time.
	Tracks() bool
}

var engine Engine

// UseEngine sets the engine. The engine packages call it when they are
// imported, so only an engine or a test needs to. Call it before building any
// signal or node.
func UseEngine(e Engine) {
	engine = e
	tracking = e.Tracks()
}

func current() Engine {
	if engine == nil {
		panic("hx: there is no engine, import hx/wasm (browser) or hx/ssr (server)")
	}
	return engine
}

// OnMount runs fn only in the browser, once the UI is on the page.
//
// While the UI is being rendered to a string on the server the function is
// ignored. While Hydrate is building the UI it waits until the page has been
// hydrated. In any other case it runs right away. Use it for what needs the
// browser: timers, local storage, focusing an input...
func OnMount(fn func()) { current().OnMount(fn) }

// IsServer tells if the code runs on a server (true) or in the browser (false).
// Use it in code shared by both to skip what only makes sense in the browser.
func IsServer() bool { return current().IsServer() }

// binder is what a node has when it can own an effect and tell when it leaves the tree.
type binder interface {
	BindWith(run, stop func())
}

// discarder is what a node has when it can free what it holds before it is used.
type discarder interface {
	Discard()
}
