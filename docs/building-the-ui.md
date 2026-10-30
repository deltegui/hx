# Building the UI

You write HTML as Go code. Every element is a `hx.INode` made by a constructor, and every method returns an `hx.INode`, so you chain calls. `INode` is an interface: the engine in use (`hx/wasm` in the browser, `hx/ssr` on a server) makes the nodes, and code that only uses `hx` works with both.

```go
hx.Div().Class("card").Body(
	hx.H2().Text("Title"),
	hx.P().Text("Some text"),
	hx.A().Href("/docs").Text("Read more"),
)
```

Methods can be chained **in any order** and on any element: `hx.Input().Placeholder("name").Class("big").Type("text")` is fine. Every method returns `hx.INode`, so a function returning one can be chained too:

```go
func card() hx.INode { return hx.Div().Class("card") }

card().Class("wide").Text("hi")
```

## Mounting

The tree needs a place in the page. Mount a node on an existing element:

```go
root := wasm.NewFromId("app")          // <div id="app">
root.Body(hx.H1().Text("Hi"))

// or with a DOM element you already have
root = wasm.New(someElement) // a js.Value
```

You can mount several roots. They are independent.

Mounting belongs to the browser engine (`hx/wasm`). On a server there is nothing to mount: the UI is written with `ssr.Render` (see [Server rendering](server-rendering.md)).

## Elements

There are constructors for the common ones:

`Div Span P A Button I Strong Em Small Code Pre Br Hr Img Label Form Input TextArea Select Option H1..H6 Ul Ol Li Table THead TBody TFoot Tr Th Td Nav Header Footer Main Section Article Aside Video Audio Source Canvas Svg Path`

For any other tag use `El`:

```go
hx.El("details").Body(
	hx.El("summary").Text("More"),
	hx.P().Text("Hidden text"),
)
```

## Content

```go
hx.P().Text("plain text")                 // set text, it is escaped by the browser
hx.P().Textf("%d items left", n)          // with a format
hx.Div().Body(child1, child2)             // set children
hx.Ul().BodyList(itemsAsSlice)            // set children from a slice
```

- `Body` **replaces** all the children. Passing `nil` children is fine, they are skipped (this is what `hx.If` relies on).
- Text and children can live together. The text goes first.
- If a node you pass to `Body` was already a child, it is moved, not recreated. See [Dynamic UI](dynamic-ui.md#reusing-nodes).

## Attributes, classes, styles

```go
el.Attribute("title", "Hello")     // any attribute
el.Attribute("disabled", "")       // an empty value is valid
el.RemoveAttribute("title")

el.Class("a", "b")                 // add classes
el.RemoveClass("a")
el.ClassIf("active", isActive)     // add or remove depending on a bool

el.Style("color", "red")           // CSS properties, one by one
el.RemoveStyle("color")

el.Id("main")
```

Shortcuts for common attributes: `Href`, `Src`, `Placeholder`, `Type`, `Selected`.

You can ask what a node will have after the next render:

```go
el.HaveClass("a")                  // bool
el.GetAttribute("title")           // string, "" if none
el.HaveAttribute("title", "Hello") // bool
el.GetStyle("color")               // string
el.HaveStyle("color", "red")       // bool
```

Setting the same value again, or going back to the value that is already rendered, does not touch the DOM.

## Events

```go
hx.Button().Text("Save").OnClick(func(ctx hx.EventContext) {
	save()
})
```

The handler receives an `EventContext`:

- `ctx.Target` is the node (`hx.INode`) that has the listener.
- `ctx.Event` is the browser event, and `nil` on a server. `wasm.EventOf(ctx)` gives it as a `js.Value`, so `wasm.EventOf(ctx).Get("target")` is the JavaScript `event.target`. `wasm.Key(ctx)` is the key of a keyboard event.

Available helpers: `OnClick`, `OnInput`, `OnChange`, `OnKeyUp`, `OnKeyDown`, `OnSubmit`, `OnFocus`, `OnBlur`. For any other event use `On` with one of the `hx.Event*` constants (`EventDblClick`, `EventMouseEnter`, `EventMouseLeave`...) or your own `hx.Event("scroll")`:

```go
el.On(hx.EventDblClick, func(ctx hx.EventContext) { ... })
```

Things to know:

- Handlers are ignored on a server: there is nobody to click.
- Calling `On` again for the same event **replaces** the handler. A binding of the same event (`BindOnInput`, `BindOnChange`, `BindOnChecked`) is not replaced: it has its own place, and it runs first, so `Input().BindOnInput(name).OnInput(f)` writes the signal and then calls `f`, which already finds the new value in it.
- Handlers run untracked: reading signals inside them never creates a dependency.
- A handler runs in a [batch](reactivity.md#batching-several-changes-at-once): however many signals it sets, each effect that depends on them runs once, when it ends. A render is scheduled after every handler.
- Read the key or the value of the event:

```go
hx.Input().OnKeyUp(func(ctx hx.EventContext) {
	if wasm.Key(ctx) == "Enter" {
		submit()
	}
})
```

- `wasm.PreventDefault(ctx)` and `wasm.StopPropagation(ctx)` are available. Use `OnSubmit` plus `PreventDefault` on forms so the page does not reload.

## Forms

For inputs, `BindValue` shows a signal and `BindOnInput` writes to it, so one signal drives the input in both directions:

```go
name := hx.Signal("")

hx.Input().Placeholder("Your name").BindValue(name).BindOnInput(name)
```

- `BindOnInput` updates on every keystroke. `BindOnChange` updates when the control loses focus or the value is committed (use it for `select`).
- `BindValue` on a `TextArea` sets its text as well.
- `name.Set("")` clears the input.

```go
// textarea
hx.TextArea().BindValue(notes).BindOnInput(notes)

// select
color := hx.Signal("green")
hx.Select().BindOnChange(color).Body(
	hx.Option().Value("red").Text("Red"),
	hx.Option().Value("green").Selected().Text("Green"),
)

// checkbox and radio: the checked state
agree := hx.Signal(false)
hx.Input().Type("checkbox").BindChecked(agree).BindOnChecked(agree)
```

To send a form to a server, with or without files, see [Forms and requests](forms-and-requests.md).

## SVG

`Svg()` and `Path()` are created in the SVG namespace, so they render correctly:

```go
hx.Svg().Attribute("viewBox", "0 0 100 100").Body(
	hx.Path().Attribute("d", "M10 10 L90 10 L50 90 Z").Attribute("fill", "tomato"),
)
```

For other SVG shapes there is no constructor yet, and `El` creates HTML-namespace elements.

## Fragments with `Noop`

`hx.Noop()` is a container that renders **only its children**, with no element of its own. `hx.Each` returns one. Use it when you need to return several siblings as one node:

```go
hx.Ul().Body(
	hx.Li().Text("first"),
	hx.Noop().Body(hx.Li().Text("a"), hx.Li().Text("b")),
	hx.Li().Text("last"),
)
// <ul><li>first</li><li>a</li><li>b</li><li>last</li></ul>
```

A `Noop` ignores things that need an element (`Class`, `Style`, `Attribute`, `On`, `Text`).

## Raw HTML

`hx.Raw(html)` is HTML that is written **as it is**, with no escaping. Like `Noop` it has no element of its own: what it holds becomes siblings of the nodes around it.

```go
hx.Div().Class("prose").Body(
	hx.H2().Text("Release notes"),
	hx.Raw(notesAsHTML), // markdown that you rendered and sanitized
)

hx.Button().Body(hx.Raw(checkIconSVG), hx.Span().Text("Done"))
hx.TBody().Body(hx.Raw("<tr><td>1</td></tr>")) // anything, even a row, goes where you put it
```

### URLs in attributes

In `href`, `src`, `action`, `formaction`, `poster`, `cite`, `data`, `background`, `manifest` and `ping` (set with `Href`, `Src` or `Attribute`), a URL whose scheme is not `http`, `https`, `mailto` or `tel` (`javascript:`, `data:`, `vbscript:`...) is replaced by `hx.UnsafeURL` (`about:invalid#hx-unsafe-url`), in the server and in the browser. Relative URLs (`/a`, `?q=1`, `#x`) always pass, and so do tabs, line breaks and capitals that browsers ignore in a scheme (`" JavaScript:"` is caught). To allow another scheme call `hx.AllowURLScheme("whatsapp")` at startup. Not covered: `srcset`, `style` and `on*` attributes; do not fill those from user input.

> **Raw is for HTML you trust.** Markup you wrote, or text that went through a sanitizer. **Never give it something a user typed**, or the page is open to script injection. `Text` escapes; `Raw` does not. `Href`, `Src` and the other attributes are not Raw, and they are checked (next section).

- On a server the HTML goes between two comments, `<!--hx-raw-->` and `<!--/hx-raw-->`. They are valid anywhere (also inside a table) and they are how the browser finds where it ends.
- When the browser hydrates the page it **adopts the HTML as it is and never touches it**. It does not compare it with the string, because the browser writes the same markup in its own way, so a difference inside raw HTML is not noticed. If the markers are missing or not closed, the parent is rebuilt by the browser and logs a mismatch, like any other difference.
- A browser that creates a `Raw` itself (not hydrating) parses the HTML the way a `<template>` does, so a `<tr>` or a `<td>` is fine wherever you put it. A `<script>` inside it runs when the server's page loads, and does not when the browser creates it.
- A `Raw` that moves (a list that is reordered) moves all its nodes as one. To change what it shows, build a new one, for example inside `hx.Dynamic`.
- Attributes, classes, styles, events, text and children set on a `Raw` do nothing.

Use it for what is not Hix: an icon, text that came out of a markdown renderer, the markup of another library. For anything that is yours, write a component.

## Reaching the DOM element

After a node has been rendered, `wasm.Underlying(node)` returns the DOM element as a `js.Value`: `wasm.Underlying(input).Get("value")`, `wasm.Underlying(form).Call("reset")`. Before it is rendered, and for a node of another engine, it is undefined (`IsUndefined()` is true).

The node is not rendered when you create it: the renderer runs on the next animation frame. To use the element as soon as it exists, for example to hand a `<canvas>` to a chart library, ask for it with `OnRendered`:

```go
canvas := hx.Canvas()
var chart js.Value
canvas.OnRendered(func() {
	// the canvas is in the document here
	chart = js.Global().Get("Chart").New(wasm.Underlying(canvas), config)
}).OnRemoved(func() {
	chart.Call("destroy") // the node left the tree
})
```

`OnRendered` runs after the renderer has finished the whole frame, so the rest of the page is there too. It runs again if the node is removed and later put back, and `OnRemoved` is its pair. Neither runs while rendering on a server. If the UI is only disposed with its root (the nodes are not removed), use `OnCleanup`. With `wasm.Hydrate`, `OnRendered` runs once the page has been adopted.
