![logo](https://raw.githubusercontent.com/deltegui/hix/refs/heads/main/logo.png)

# Hix

Hix is a reactive frontend library for Go (WASM).
It aims to be **small, simple, and predictable**.

The core is about 1.2K lines (comments included). The browser engine is about 1.9K and the server engine about 300. It can:

* Write HTML in plain old Go. No code generation needed. No strange mixed files without editor support.

* Handle JavaScript events directly.

* Provide reactivity using **Signals**, **Computed**, and **Effects** (the *Solid.js* model), with owners (`CreateRoot`, node effects) so nothing leaks.

* Update the DOM efficiently — only when changes occur, once per animation frame, minimizing Go ↔ JS calls.

* Work seamlessly with **TinyGo**, reducing bundle sizes dramatically.

* **Send forms, files and JSON** with `hxhttp.Do`, `hx.Action` and `hx.Resource`: progress, cancel, timeout, state in signals. The same form also works without WASM.

* **Render on the server and hydrate in the browser.** A Go backend writes plain HTML from the same UI code, and the WASM adopts it without touching the DOM.

Hix has no dependencies: only the standard library (`syscall/js` in the browser).

Hix doesn’t try to be a full-fledged framework — it’s just a library for writing reactive web interfaces in Go.

If you need a full framework to build PWAs or complex applications, try go-app.

**[Live demo](https://deltegui.github.io/hx/)** — the code is in [`examples/demo`](examples/demo): a tab per feature and 39 checks that run against the real DOM, in your browser (about 319K of WASM over the network, built with TinyGo).

## A taste

```go
package main

import (
	"fmt"

	"github.com/deltegui/hx"
	"github.com/deltegui/hx/wasm"
)

func main() {
	count := hx.Signal(0)
	label := hx.Computed(func() string { return fmt.Sprintf("Clicked %d times", count.Get()) })

	wasm.NewFromId("app").Body(
		hx.H1().Text("Hello, Hix"),
		hx.P().BindText(label),
		hx.Button().Text("Click me").OnClick(func(hx.EventContext) {
			count.Update(func(n int) int { return n + 1 })
		}),
	)

	select {}
}
```

There is no "update the DOM" code. `count` is state, `label` is derived from it, and the paragraph follows `label`.

```sh
go get github.com/deltegui/hx
GOOS=js GOARCH=wasm go build -o main.wasm .
```

[Getting started](docs/getting-started.md) has the HTML page and every step.

## What it looks like

```go
// everything chains, in any order
hx.Input().Placeholder("Your name").Class("big").BindValue(name).BindOnInput(name)

// a list that keeps its rows (and their inputs, focus, scroll) per key
hx.Ul().Body(hx.EachKeyed(todos,
	func(t Todo) int { return t.ID },
	func(id int, todo hx.Gettable[Todo], index hx.Gettable[int]) hx.INode {
		return hx.Li().BindText(hx.Computed(func() string { return todo.Get().Title }))
	}))

// lists and conditionals inside a node effect; the effect belongs to the node
list := hx.Ul()
list.Effect(func() {
	list.Body(hx.Each(todos, func(i int, todo string) hx.INode {
		return hx.Li().Text(todo)
	}))
})

box.Effect(func() {
	box.Body(hx.IfFn(open, func() hx.INode { return details() }))
})

// effects that clean up after themselves
hx.EffectFunc(func() {
	id := startTimer()
	hx.OnCleanup(func() { stopTimer(id) })
})
```

## Packages

One API, two engines. You write the UI against `hx`, and each program imports the engine it needs:

| Package | For | What it is |
|---|---|---|
| `hx` | everywhere | Signals, effects, the `INode` interface and the constructors (`Div`, `Span`...), `If`/`Each`/`EachKeyed`, `Action`, `Resource`. |
| `hx/wasm` | the browser | The engine that keeps the tree and updates the DOM (`wasm.NewFromId`, `wasm.Hydrate`). |
| `hx/ssr` | a server | The engine that writes HTML (`ssr.Render`). No DOM code, no tracking of signals, so renders run in parallel. |
| `hx/hxhttp` | the browser | Requests: `hxhttp.Do` (XMLHttpRequest) with bodies for forms and files, progress, cancel and timeout. The types build everywhere. |
| `hx/hxjson` | the browser | JSON requests on top of `hxhttp`, apart so `encoding/json` is only linked if you use it. |

A library of components (a Bootstrap wrapper, say) imports only `hx`, so the same code builds on the server and in the browser, and the browser can hydrate what the server wrote.

```go
// server                                      // browser
html := ssr.Render(func() hx.INode {          wasm.Hydrate("app", func() hx.INode {
	return app.App(state)                          return app.App(state)
})                                            })
```

## Documentation

| | |
|---|---|
| [Getting started](docs/getting-started.md) | From an empty folder to a running page. Go and TinyGo builds. |
| [Reactivity](docs/reactivity.md) | Signals, computed values, effects, roots, cleanup and who stops what. |
| [Building the UI](docs/building-the-ui.md) | Elements, content, attributes, classes, styles, events, forms, SVG. |
| [Forms and requests](docs/forms-and-requests.md) | Forms with or without WASM, files, JSON, `hxhttp.Do`, `Action`, `Resource`, CSRF. |
| [Server rendering](docs/server-rendering.md) | `ssr.Render`, `wasm.Hydrate`, shared state, `OnMount`, what to watch for. |
| [Dynamic UI](docs/dynamic-ui.md) | Bindings, conditionals, lists (`Each`, `EachKeyed`), reusing nodes. |
| [Recipes](docs/recipes.md) | Counter, todo list, search, tabs, timer, validation, components. |
| [How it works](docs/how-it-works.md) | The two engines, dependency tracking, the DOM renderer, hydration. |
| [API reference](docs/api.md) | Every exported function and method. |
| [Troubleshooting and limits](docs/troubleshooting.md) | Common mistakes and what Hix does not do. |

Recipes in the documentation are complete and runnable. [`examples/demo`](examples/demo) is the [live demo](https://deltegui.github.io/hx/): a larger example with a tab per feature and a page of checks that run against the real DOM. [`examples/ssr`](examples/ssr) renders a page on a Go server and hydrates it in the browser. [`examples/forms`](examples/forms) is a form with file uploads that works with and without JavaScript, and [`examples/form-minimal`](examples/form-minimal) is the smallest version of it, to see what the mechanism needs.

## Sizes

Thanks to TinyGo and Hix simplicity, the example shown above ("A taste") can achieve these sizes:

| Compiled                  | Command                                                                   | Size |
|---------------------------|---------------------------------------------------------------------------|------|
| Go full                   | `GOOS=js GOARCH=wasm go build -o main.wasm main.go`                       | 2.9M |
| TinyGo                    | `GOOS=js GOARCH=wasm tinygo build -o main.wasm main.go `                  | 958K |
| Optimized TinyGo          | `GOOS=js GOARCH=wasm tinygo build --no-debug -opt=z -o main.wasm main.go` | 292K |
| Optimized TinyGo + GZip   | `gzip -k -9 main.wasm`                                                    | 110K |
| Optimized TinyGo + brotli | `brotli -f -q 11 main.wasm`                                               | 90K  |

Measured with Go 1.27.3 and TinyGo 0.43.0. For a bigger program, the [live demo](https://deltegui.github.io/hx/) ([`examples/demo`](examples/demo), every feature, 39 checks) is 4.2M with Go and 757K with optimized TinyGo (233K gzip, 179K brotli). Both builds pass the 39 checks in Chromium.

## Status

Hix is young and used in a side project. The API is small on purpose. The main limit is that the browser engine only runs as WASM (the core and the server engine build anywhere). See [Troubleshooting and limits](docs/troubleshooting.md).

Run the tests with:

```sh
go test -race ./...                                                         # the core and both engines, natively
GOOS=js GOARCH=wasm go test -exec "$(go env GOROOT)/lib/wasm/go_js_wasm_exec" ./...   # also in WASM (needs Node.js)
```

The DOM code is checked in a real browser: [`examples/demo`](examples/demo) has 39 checks, [`examples/ssr`](examples/ssr) (`make check`) checks the hydration, and [`examples/forms`](examples/forms) (`make check`) checks the forms with JavaScript off and on. A test also checks that the two engines write the same HTML for the same UI, which is what hydration needs.

## Why Hix?

The Go WASM ecosystem feels somewhat abandoned. Vecty hasn’t received updates in four years. Vugu also hasn’t had any major updates for quite some time (and .vugu files have little to no support in editors). go-app seems to be the way to go, but it’s mainly oriented towards building PWAs.

I needed interactivity for a project that already uses backend rendering, but I didn’t want to rely on the Node.js ecosystem — they don’t seem to care much about having a ton of dependencies, dependency hell or breaking changes, which compromises long-term maintainability. I started Hix as a small demo, but I liked it so much that I began using it in my side project.
