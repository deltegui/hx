# Building the UI

You write HTML as Go code. Every element is a `*hx.VNode` made by a constructor, and every method returns the same `*hx.VNode`, so you chain calls.

```go
hx.Div().Class("card").Body(
	hx.H2().Text("Title"),
	hx.P().Text("Some text"),
	hx.A().Href("/docs").Text("Read more"),
)
```

Methods can be chained **in any order** and on any element: `hx.Input().Placeholder("name").Class("big").Type("text")` is fine. `hx.INode` is just another name for `*hx.VNode`, so a function returning `hx.INode` can be chained too:

```go
func card() hx.INode { return hx.Div().Class("card") }

card().Class("wide").Text("hi")
```

## Mounting

The tree needs a place in the page. Mount a node on an existing element:

```go
root := hx.NewFromId("app")          // <div id="app">
root.Body(hx.H1().Text("Hi"))

// or with a DOM element you already have
root = hx.New(someElement)
```

You can mount several roots. They are independent.

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

- `ctx.Target` is the node (`*hx.VNode`) that has the listener.
- `ctx.Event` is the browser event (`dom.Event` from `honnef.co/go/js/dom/v2`).

Available helpers: `OnClick`, `OnInput`, `OnChange`, `OnKeyUp`, `OnKeyDown`, `OnSubmit`, `OnFocus`, `OnBlur`. For any other event use `On` with one of the `hx.Event*` constants (`EventDblClick`, `EventMouseEnter`, `EventMouseLeave`...) or your own `hx.Event("scroll")`:

```go
el.On(hx.EventDblClick, func(ctx hx.EventContext) { ... })
```

Things to know:

- Calling `On` again for the same event **replaces** the handler. There is only one listener per event.
- Handlers run untracked: reading signals inside them never creates a dependency.
- A render is scheduled after every handler.
- Read the event's value or key with the underlying event:

```go
hx.Input().OnKeyUp(func(ctx hx.EventContext) {
	if key, ok := ctx.Event.(*dom.KeyboardEvent); ok && key.Key() == "Enter" {
		submit()
	}
})
```

- `ctx.Event.PreventDefault()` and `StopPropagation()` are available. Use `OnSubmit` plus `PreventDefault` on forms so the page does not reload.

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

## Reaching the DOM element

After a node has been rendered, `node.Underlying()` returns its `dom.Element`. Inputs, textareas, selects, options, links, buttons, forms, images, labels, canvases, videos and audios give you the specific `honnef` type (for example `*dom.HTMLInputElement`). Other elements give a `*dom.BasicHTMLElement`.

The node is not rendered when you create it: the renderer runs on the next animation frame. To use the element as soon as it exists, for example to hand a `<canvas>` to a chart library, ask for it with `OnRendered`:

```go
canvas := hx.Canvas()
var chart js.Value
canvas.OnRendered(func() {
	// the canvas is in the document here
	chart = js.Global().Get("Chart").New(canvas.Underlying().Underlying(), config)
}).OnRemoved(func() {
	chart.Call("destroy") // the node left the tree
})
```

`OnRendered` runs after the renderer has finished the whole frame, so the rest of the page is there too. It runs again if the node is removed and later put back, and `OnRemoved` is its pair. Neither runs while rendering on a server. If the UI is only disposed with its root (the nodes are not removed), use `OnCleanup`. With `Hydrate`, `OnRendered` runs once the page has been adopted.
