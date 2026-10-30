# Getting started

This page takes you from an empty folder to a working page. It needs Go 1.25 or newer. TinyGo is optional and makes the result much smaller.

## 1. Create the project

```sh
mkdir hello && cd hello
go mod init hello
go get github.com/deltegui/hx
```

## 2. Write the program

`main.go`:

```go
//go:build js && wasm

package main

import (
	"fmt"

	"github.com/deltegui/hx"
	"github.com/deltegui/hx/wasm"
)

func main() {
	count := hx.Signal(0)
	label := hx.Computed(func() string {
		return fmt.Sprintf("Clicked %d times", count.Get())
	})

	wasm.NewFromId("app").Body(
		hx.H1().Text("Hello, Hix"),
		hx.P().BindText(label),
		hx.Button().Text("Click me").OnClick(func(hx.EventContext) {
			count.Update(func(n int) int { return n + 1 })
		}),
	)

	select {} // keep the program alive, the page needs it to react
}
```

`hx` is what you build the UI with. `hx/wasm` is the engine that draws it in the browser: it brings the nodes and the DOM code, and `wasm.NewFromId` mounts them. A server uses `hx/ssr` instead (see [Server rendering](server-rendering.md)); a library of components imports only `hx` and works with either.

Three things to notice:

- `hx.Signal(0)` is the state. `hx.Computed` derives a value from it.
- `hx.P().BindText(label)` keeps the text of the paragraph in sync with `label`. You never write "update the DOM" code.
- `select {}` is required. When `main` returns, the WASM program stops and nothing reacts anymore.

## 3. Write the page

`index.html`:

```html
<!doctype html>
<html>
<head><meta charset="utf-8"><title>Hello, Hix</title></head>
<body>
  <div id="app"></div>
  <script src="wasm_exec.js"></script>
  <script>
    const go = new Go();
    WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject)
      .then((result) => go.run(result.instance));
  </script>
</body>
</html>
```

The `div` with `id="app"` is the mount point passed to `wasm.NewFromId`.

## 4. Build and serve

```sh
GOOS=js GOARCH=wasm go build -o main.wasm .
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" .
python3 -m http.server 8080      # any static server works
```

Open <http://localhost:8080>. The browser must load the page through HTTP, not from `file://`.

## Smaller output with TinyGo

```sh
GOOS=js GOARCH=wasm tinygo build --no-debug -opt=z -o main.wasm main.go
cp "$(tinygo env TINYGOROOT)/targets/wasm_exec.js" .
```

TinyGo has its own `wasm_exec.js`. Do not mix it with the one from Go.

| Compiler | Size |
|---|---|
| Go | ~4.5 MB |
| TinyGo | ~1.7 MB |
| TinyGo, optimized | ~560 KB (about 156 KB gzip) |

## Where to go next

- [Reactivity](reactivity.md): signals, computed values and effects.
- [Building the UI](building-the-ui.md): elements, text, attributes, events, forms.
- [Dynamic UI](dynamic-ui.md): bindings, conditionals and lists.
- [Forms and requests](forms-and-requests.md): forms that work with and without WASM, files, JSON.
- [Server rendering](server-rendering.md): render on a Go server and hydrate in the browser.
- [Recipes](recipes.md): complete examples (todo list, tabs, timer...).
- [`examples/demo`](../examples/demo): a larger runnable example with a tab for each feature and a page of checks against the real DOM.
