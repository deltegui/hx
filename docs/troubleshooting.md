# Troubleshooting and limits

## Nothing happens in the browser

- **The page is blank and the console says `Go is not defined`.** Load `wasm_exec.js` before your script, and use the one that matches your compiler (Go and TinyGo have different files).
- **The WASM does not load from `file://`.** Serve the folder over HTTP.
- **It rendered once and then nothing reacts.** `main` returned. End it with `select {}`.
- **`getElementById` fails or the node does not appear.** The mount `id` has to exist when `main` runs. Put the script after the element, or at the end of `<body>`.

## "Cannot call Get if there is no effect"

Hix logs this when you call `signal.Get()` while building the tree, outside any effect. The value is read once and will never update:

```go
hx.P().Text(fmt.Sprint(count.Get()))   // read once: it will not change
hx.P().BindText(label)                  // follows the signal
```

Inside an event handler it is fine and does not warn.

## The UI does not update after I change a slice or a map

Hix does not look inside values. Always `Set` a **new** slice or map:

```go
items.Update(func(l []string) []string { return append(slices.Clone(l), "x") })
```

## My list rebuilds on every change

That is how `Each` works: when the signal changes, the effect that contains it runs again and sets a new body. Use `EachKeyed` so rows are built once per key and reused. See [Dynamic UI](dynamic-ui.md#keeping-the-rows-eachkeyed).

## A row loses focus, typed text or scroll when the list changes

With `Each` the old row is removed and a new one is created. Use `EachKeyed`.

## `EachKeyed` rows show old values

The row is built once, so anything you read with `item.Get()` while building is a snapshot. Read it inside a binding (`BindText(hx.Computed(...))`) or an effect, which run again when the item changes.

## "EachKeyed: duplicated key" in the console

Two items returned the same key. Keys have to be unique (an ID, not an index or a name that can repeat). The repeated item is still rendered, but as a row that is rebuilt every time.

## An effect keeps running after its node is gone

You probably created it with `hx.EffectFunc` outside of a node, an effect or a root. Use `node.Effect(fn)`, or wrap the code in `hx.CreateRoot`. See [Reactivity: ownership](reactivity.md#who-stops-an-effect-ownership).

## `EachMap` shows the rows in a different order each time

Go maps have no order. Use `EachMapSorted`, or put the keys in a sorted slice yourself.

## `Class`, `Style` or `Text` on a `Noop` does nothing

A `Noop` has no element of its own, so there is nothing to style. Put a real element around it.

## A class with a space is ignored

`Class("a b")` logs a warning and ignores it. Pass them separately: `Class("a", "b")`.

## I need the DOM element

Use `node.Underlying()` after the node has been rendered. Before that it is `nil`.

## Limits worth knowing

- **WASM only.** The package imports `syscall/js`, so it only builds with `GOOS=js GOARCH=wasm`. There is no server-side rendering outside the browser.
- **`Each` rebuilds its rows.** Use `EachKeyed` when rows have state or the list is long.
- **No batching of signal writes.** Each `Set` runs its effects right away. DOM updates are still batched once per frame.
- **`Computed` needs a comparable type.** For slices, maps or structs holding them, use a signal and update it from an effect.
- **No router, no global store, no component lifecycle hooks.** Components are plain functions, and `node.Effect` + `OnCleanup` cover setup and teardown.
- **Only a few element constructors.** Use `El("tag")` for the rest. SVG shapes other than `Svg` and `Path` need their own namespace and do not have constructors yet.
- **Events.** Helpers exist for the common ones; use `On` with `hx.Event("name")` for any other.
- **`Underlying()` type.** Inputs, textareas, selects, options, links, buttons, forms, images, labels, canvases, videos and audios return their specific `honnef` type. Other elements return `*dom.BasicHTMLElement`.
