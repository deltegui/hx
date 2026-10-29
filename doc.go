// Package hx is a small reactive frontend library for Go (WASM).
//
// You describe the UI with Go functions, keep the state in signals, and hx
// updates the DOM when the signals change.
//
//	count := hx.Signal(0)
//	label := hx.Computed(func() string { return fmt.Sprintf("Clicked %d times", count.Get()) })
//
//	hx.NewFromId("app").Body(
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
// other tag. All the methods of VNode return *VNode, so calls are chained.
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
// # Rendering
//
// Changes are collected and applied to the DOM once per animation frame by a
// DiffRenderer. A StringRenderer writes HTML instead.
//
// # Server rendering
//
// RenderToString writes the HTML of a UI on a server, and Hydrate brings that
// HTML to life in the browser. The core of the package (signals, nodes,
// StringRenderer, RenderToString) builds for any platform. The part that talks
// to the DOM (New, NewFromId, Hydrate...) only builds for GOOS=js GOARCH=wasm.
package hx
