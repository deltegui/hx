// Package hx is a small reactive frontend library for Go (WASM).
//
// You describe the UI with Go functions, keep the state in signals, and hx
// updates the DOM when the signals change. This package is the API a UI is
// written against; what makes the nodes is an engine, and a program imports the
// one it needs: hx/wasm draws in the browser and hx/ssr writes HTML on a
// server. A library of components that imports only hx works with both.
//
//	count := hx.Signal(0)
//	label := hx.Computed(func() string { return fmt.Sprintf("Clicked %d times", count.Get()) })
//
//	wasm.NewFromId("app").Body(
//		hx.H1().Text("Hello, Hix"),
//		hx.P().BindText(label),
//		hx.Button().Text("Click me").OnClick(func(hx.EventContext) {
//			count.Update(func(n int) int { return n + 1 })
//		}),
//	)
//
// # Reactivity
//
// Signal holds a value, Computed derives a value from other signals and
// EffectFunc runs a function again when the signals it read change.
// Effects have an owner so they can be stopped: the node they belong to
// (VNode.Effect, BindText, BindValue), another effect, or a root (CreateRoot).
//
// # Building the UI
//
// Every element has a constructor (Div, Span, Input...) and El creates any
// other tag. All the methods of INode return INode, so calls are chained.
// Body sets the children. If, Show, IfFn, ShowFn and Each build parts of the
// UI that depend on signals.
//
// # Forms and requests
//
// Do makes a request in the browser, with bodies for forms and files, progress,
// timeout and cancel. Action and Resource keep the state of a request in
// signals. The package hxjson sends and reads JSON. A form is plain HTML, so it
// also works without WASM when the page was rendered by a server.
//
// # Engines
//
// An Engine makes the nodes and decides what a render is. hx/wasm keeps a tree
// of nodes, applies the changes to the DOM once per animation frame and brings
// the HTML of a server to life (wasm.Hydrate). hx/ssr keeps a plain tree and
// writes it as HTML (ssr.Render); it does not track signals, so a signal is a
// plain value there and many UIs are rendered at the same time. The engine
// registers itself when it is imported (UseEngine).
//
// # Server rendering
//
// ssr.Render writes the HTML of a UI on a server, and wasm.Hydrate brings that
// HTML to life in the browser. This package builds for any platform. The part
// that talks to the DOM (wasm.New, wasm.NewFromId, wasm.Hydrate...) only builds
// for GOOS=js GOARCH=wasm.
package hx
