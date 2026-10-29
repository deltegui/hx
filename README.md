![logo](https://raw.githubusercontent.com/deltegui/hix/refs/heads/main/logo.png)

# Hix

Hix is a reactive frontend library for Go (WASM).
It aims to be **small, simple, and predictable**.

In about 1.9K lines of code, it can:

* Write HTML in plain old Go. No code generation needed. No strange mixed files without editor support.

* Handle JavaScript events directly.

* Provide reactivity using **Signals**, **Computed**, and **Effects** (the *Solid.js* model), with owners (`CreateRoot`, node effects) so nothing leaks.

* Update the DOM efficiently — only when changes occur, once per animation frame, minimizing Go ↔ JS calls.

* Work seamlessly with **TinyGo**, reducing bundle sizes dramatically.

Hix only depends on *honnef.co/go/js/dom/v2*.

Hix doesn’t try to be a full-fledged framework — it’s just a library for writing reactive web interfaces in Go.

If you need a full framework to build PWAs or complex applications, try go-app.

## A taste

```go
package main

import (
	"fmt"

	"github.com/deltegui/hx"
)

func main() {
	count := hx.Signal(0)
	label := hx.Computed(func() string { return fmt.Sprintf("Clicked %d times", count.Get()) })

	hx.NewFromId("app").Body(
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

## Documentation

| | |
|---|---|
| [Getting started](docs/getting-started.md) | From an empty folder to a running page. Go and TinyGo builds. |
| [Reactivity](docs/reactivity.md) | Signals, computed values, effects, roots, cleanup and who stops what. |
| [Building the UI](docs/building-the-ui.md) | Elements, content, attributes, classes, styles, events, forms, SVG. |
| [Dynamic UI](docs/dynamic-ui.md) | Bindings, conditionals, lists (`Each`, `EachKeyed`), reusing nodes. |
| [Recipes](docs/recipes.md) | Counter, todo list, search, tabs, timer, validation, components. |
| [How it works](docs/how-it-works.md) | Dependency tracking, the renderer, the string renderer. |
| [API reference](docs/api.md) | Every exported function and method. |
| [Troubleshooting and limits](docs/troubleshooting.md) | Common mistakes and what Hix does not do. |

Recipes in the documentation are complete and runnable. [`examples/demo`](examples/demo) is a larger example with a tab per feature and a page of checks that run against the real DOM.

## Sizes

Example with code using TinyGo (just 631K of WASM, 191K gzip): [Hix example](https://deltegui.github.io/hx/).

Thanks to TinyGo and Hix simplicity, the example shown above can achieve these sizes:

| Compiled                  | Command                                                                   | Size |
|---------------------------|---------------------------------------------------------------------------|------|
| Go full                   | `GOOS=js GOARCH=wasm go build -o main.wasm main.go`                       | 5.1M |
| TinyGo                    | `GOOS=js GOARCH=wasm tinygo build -o main.wasm main.go `                  | 1.5M |
| Optimized TinyGo          | `GOOS=js GOARCH=wasm tinygo build --no-debug -opt=z -o main.wasm main.go` | 631K |
| Optimized TinyGo + GZip   | `gzip -k -9 main.wasm`                                                    | 191K |
| Optimized TinyGo + brotli | `brotli -f -q 11 main.wasm`                                               | 138K |

These numbers were measured on an earlier version of the library.

## Status

Hix is young and used in a side project. The API is small on purpose. The main limits are: it only runs as WASM, and signal writes are not batched. See [Troubleshooting and limits](docs/troubleshooting.md).

Run the tests with:

```sh
GOOS=js GOARCH=wasm go test -exec "$(go env GOROOT)/lib/wasm/go_js_wasm_exec" ./...
```

(they need Node.js and cover signals and the string renderer; the DOM code is checked in a real browser by [`examples/demo`](examples/demo)).

## Why Hix?

The Go WASM ecosystem feels somewhat abandoned. Vecty hasn’t received updates in four years. Vugu also hasn’t had any major updates for quite some time (and .vugu files have little to no support in editors). go-app seems to be the way to go, but it’s mainly oriented towards building PWAs.

I needed interactivity for a project that already uses backend rendering, but I didn’t want to rely on the Node.js ecosystem — they don’t seem to care much about having a ton of dependencies, dependency hell or breaking changes, which compromises long-term maintainability. I started Hix as a small demo, but I liked it so much that I began using it in my side project.
