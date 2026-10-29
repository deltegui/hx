# Server rendering and hydration

Hix can write the HTML of your UI on a server, and then bring that HTML to life in the browser. You write the UI once, in Go.

1. **The server** runs your UI and sends plain HTML. The page shows its content right away, search engines and `curl` see it, and it works before any WASM is loaded.
2. **The browser** loads the WASM, runs the same UI code, and *adopts* the HTML that is already there: it attaches the event listeners and starts the signals. This is called hydration. From then on it is a normal Hix page.

```go
// shared by the server and the browser
func App(state State) hx.INode { ... }

// server
html := hx.RenderToString(func() hx.INode { return App(state) })

// browser (WASM)
hx.Hydrate("app", func() hx.INode { return App(state) })
```

A complete, runnable example is in [`examples/ssr`](../examples/ssr).

## What builds where

The library has two halves:

| | Builds for | Contains |
|---|---|---|
| Core | every platform | signals, effects, roots, nodes, `Each`/`If`/`EachKeyed`, `StringRenderer`, `RenderToString`, `OnMount`, `IsServer`, `StateScript` |
| Browser | `GOOS=js GOARCH=wasm` only | `New`, `NewFromId`, `Hydrate`, `ReadState`, `Underlying()`, the DOM renderer |

So a Go server can import `github.com/deltegui/hx` and build UI, and the same package compiled to WASM runs it in the browser.

On the server, `EventContext.Event` is `any` (there is no browser event), and `Underlying()` does not exist.

## Project layout

```
myapp/
  app/          the UI. Imports hx. Must NOT import syscall/js or honnef.
    app.go
  server/       main package for the server (builds natively)
    main.go
  client/       main package for the browser (builds with GOOS=js GOARCH=wasm)
    main.go
```

The rule for the shared package is: **it has to build for both**. Anything that needs the browser goes in a file with a build tag:

```go
//go:build js && wasm            // enter_wasm.go: uses the browser event
func isEnter(ctx hx.EventContext) bool { ... }

//go:build !(js && wasm)         // enter_native.go: on the server it never happens
func isEnter(hx.EventContext) bool { return false }
```

## The server

```go
func handler(w http.ResponseWriter, r *http.Request) {
	state := loadState(r) // from your database, for example

	body := hx.RenderToString(func() hx.INode { return app.App(state) })
	data, _ := json.Marshal(state)

	fmt.Fprintf(w, `<!doctype html>
<html><head><meta charset="utf-8"><title>My app</title></head>
<body>
<div id="app">%s</div>
%s
<script src="/static/wasm_exec.js"></script>
<script>
  const go = new Go();
  WebAssembly.instantiateStreaming(fetch("/static/main.wasm"), go.importObject)
    .then((r) => go.run(r.instance));
</script>
</body></html>`, body, hx.StateScript("hx-state", string(data)))
}
```

`hx.RenderToString(build)`:

- calls `build` inside a root and writes the result as HTML,
- runs every effect and binding **once**, to compute the initial text and attributes,
- stops all of them before it returns, so nothing leaks from one request to the next,
- ignores `OnMount`.

It is safe to call from many goroutines. Calls are **serialized**, because the tracking of effects is global state: two requests are rendered one after the other.

The HTML is written without extra whitespace, tags in lowercase, attributes sorted, text and attribute values escaped.

## The browser

```go
func main() {
	var state app.State
	json.Unmarshal([]byte(hx.ReadState("hx-state")), &state)

	hx.Hydrate("app", func() hx.INode { return app.App(state) })

	select {}
}
```

`hx.Hydrate(id, build)` calls `build` as a normal render would, but does not create elements. It walks the nodes and the HTML at the same time, pairing each node with its element, and:

- records that the text, attributes and classes that already match are in the DOM (nothing is written again),
- attaches the event listeners,
- runs the functions given to `OnMount`.

It returns a function that stops everything the UI started.

Hydration **does not change the DOM** when the server and the browser agree. In the example, hydrating creates no nodes and sets no attributes; the only change is what `OnMount` does.

### Sharing the state

Both sides have to start from the same state, or the browser would build a UI that differs from the HTML. The easy way is the one in the example: the server writes the state as JSON in the page with `hx.StateScript(id, json)`, and the browser reads it with `hx.ReadState(id)`.

`StateScript` escapes the JSON, so it can never close its own `<script>` element.

You can use any other way to get the same data to both sides (a fetch, query parameters...). What matters is that `build` is called with equal data.

## `OnMount` and `IsServer`

Some code only makes sense in the browser: timers, local storage, focusing an input, measuring an element.

```go
hx.OnMount(func() {
	input.Underlying().Focus()
})
```

`OnMount(fn)` runs `fn`:

- **never** while rendering on the server,
- **after hydration** when called from the UI that `Hydrate` is building,
- right away in any other case (a normal browser render).

For code outside `OnMount`, `hx.IsServer()` is `true` on the server and `false` in the browser.

Remember that `node.Effect` and the bindings also run on the server, once. Do not put browser-only calls inside them without a guard.

## When the server and the browser disagree

If the page does not match what `build` produces, Hix logs a message in the console and repairs it:

| The page has... | Hix does... |
|---|---|
| a different text | writes the text (one text change) |
| a different or missing attribute, class or id | sets it (one attribute change) |
| a different tag, or a different number of child elements | removes that part and renders it again in the browser |
| no HTML at all (an empty `#app`) | renders everything in the browser |

The messages say where:

```
hx: hydration mismatch in <ul>: the page has 3 child elements and the UI has 2. That part is rendered again in the browser.
hx: hydration mismatch in <div>: the class "page" is missing. The browser sets it.
```

The page always ends up correct. A mismatch costs time, and it is a bug to fix: it usually means the two sides had different data. After hydration, `window.__hxHydrationMismatches` has the number of problems found.

## Things that change the HTML on the way

The browser parses HTML with its own rules, and it can differ from what Hix wrote:

- **Tables.** The browser adds a `<tbody>` to `<table><tr>`. Write `Table().Body(TBody().Body(Tr()...))` yourself.
- **Invalid nesting.** `<p>` cannot contain `<div>` or another `<p>`, `<a>` cannot contain `<a>`, a `<button>` cannot contain a `<button>`. The browser closes them early and the structure no longer matches.
- **A newline at the start of `<textarea>` or `<pre>`** is dropped by the browser. Hix writes one extra newline in those cases, so the text matches.
- **`<script>` and `<style>` content.** Text is escaped, which is wrong for code. Do not put code in those with `Text`.
- **Whitespace.** Hix writes none between elements. If you process or pretty-print the HTML, stray text nodes can appear.

## What to know

- **Clicks on buttons are lost before hydration.** Between the first paint and the end of the WASM load (it can be hundreds of KB) the page shows its content but a button with an `OnClick` does nothing. Hix does not record those events. **Forms do work**: they are HTML, so the browser sends them by itself (see [Forms and requests](forms-and-requests.md)). A link is a link. Design the important actions as forms and links.
- **Typing before hydration.** Text typed in an input before it is hydrated stays in the input, but the signal does not know about it until the next `input` event. If you need it, read the value in `OnMount`.
- **Streaming** is not supported: `RenderToString` returns the whole page.
- **Signals created outside `build`** (global ones) are shared by every render. Their subscribers are released when the render ends, but if two requests need different values, create the signals inside `build`.
- **Concurrency.** Because renders are serialized, one slow `build` delays the next. Do the slow work (database, network) before you call `RenderToString` and pass the data in.

## Testing it

On the server, the HTML is a string, so you can test it with plain `go test`:

```go
func TestPage(t *testing.T) {
	got := hx.RenderToString(func() hx.INode { return app.App(app.State{Count: 3}) })
	if !strings.Contains(got, `<p id="count">count: 3</p>`) {
		t.Fatal(got)
	}
}
```

To test the hydration in a browser, [`examples/ssr/check.mjs`](../examples/ssr/check.mjs) starts the server and a headless Chromium, loads the page, and checks that hydrating changes nothing, that the page works, and that pages with broken HTML recover. Run it with `make check` in `examples/ssr` (needs Node 22+ and Chromium).
