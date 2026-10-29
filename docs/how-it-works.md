# How it works

You do not need this page to use Hix. It explains what happens under the hood so the behaviour is predictable.

## Two halves

1. **Reactivity** (`signal.go`). Signals, computed values and effects. It knows nothing about the DOM.
2. **Nodes and renderer** (`element.go`, `diffrenderer.go`). A tree of `VNode` and the code that applies it to the DOM.

They meet in bindings and `node.Effect`: an effect that calls node methods like `Text` or `Class`.

## Dependency tracking

There is one global "current effect". While an effect runs, every `Signal.Get` registers that effect as a subscriber of the signal. `Set` runs the subscribers synchronously. Before an effect re-runs, it unsubscribes from everything and its child effects are disposed, so dependencies are always the ones from the last run.

Effects belong to an owner (a parent effect, a root or a node) so they can be stopped. See [Reactivity](reactivity.md#who-stops-an-effect-ownership).

## A node is a description of what you want

A `VNode` stores the **desired state**: text, attributes, classes, styles, listeners, children. Each property remembers two values, the one already in the DOM and the pending one, plus a flag saying what changed. Calling a method only edits that description and marks the node "dirty".

This is why `GetAttribute`, `HaveClass`... answer about the state *after the next render*, and why setting a value and then setting it back before the render changes nothing in the DOM.

## Rendering

1. A change marks the node with the renderer and asks for a render.
2. The `DiffRenderer` schedules **one** `requestAnimationFrame` per frame, however many changes happen.
3. In the frame it finds the lowest node that contains every marked node, and walks that part of the tree:
   - nodes that are new get an element created and inserted,
   - nodes that were removed have their element removed and are deactivated,
   - then each marked node applies its pending text, value, attributes, classes, styles and listeners.

So ten `Set` calls in a row can run your effects ten times, but they cost one DOM update.

## Children

`Body` marks the old children as removed and adds the new ones. Nodes that appear in both are kept. The children you give are the children you get: anything that is not the same node as before is created new. `EachKeyed` is what gives rows an identity, by passing the same nodes again for the same key.

After the nodes are created or removed, each node that had its children changed puts their DOM elements in order (`placeChildren`):

1. It reads the order the elements currently have in the DOM.
2. It finds the longest run of elements that are already in the right relative order. Those are not touched.
3. The rest are inserted before the next element, walking from the end.

So adding a row inserts one element, removing one removes one, and moving the last row to the top moves one element. Elements that stay keep their focus, scroll and selection. A `Noop` is transparent here: its children are placed inside the DOM element that holds the `Noop`, between the siblings around it.

## Text

Text is stored in its own text node, placed before the children. Changing it only changes that text node, so children are untouched.

## Events

Each node has at most one DOM listener per event type. `On` stores the handler and the listener is registered the first time the node renders. Calling `On` again swaps the handler. When a node leaves the tree its listeners are removed (and the Go function that backs them is released). If it comes back, they are registered again.

## Speed notes

- The Go DOM library `honnef.co/go/js/dom` identifies the type of every value it returns with dozens of JavaScript calls. Hix avoids it when creating and removing nodes and the text node (`domraw.go`), which made rendering 5,000 rows about 3x faster and removing them about 20x faster. Elements you are likely to need typed (inputs, textareas, selects, links, buttons, forms, images...) still come back with their specific type.
- Every property change is a single DOM call. Styles are written together in the `style` attribute.
- `Each` rebuilds its rows when the list signal changes. Use `EachKeyed` to keep them (see [Dynamic UI](dynamic-ui.md#keeping-the-rows-eachkeyed)), or bind each row to its own signal.

## StringRenderer

`StringRenderer` implements the same `Renderer` interface but writes HTML text instead of touching the DOM:

```go
var r hx.StringRenderer
r.Render(hx.Div().Class("card").Body(hx.P().Text("hi")))
r.String() // <div class="card"><p>hi</p></div>
```

Tags are lowercase, `Noop` nodes are transparent, removed entries are skipped, and attributes, classes and styles come out sorted so the output is stable. It builds on every platform, which is what makes server rendering possible: `RenderToString` runs your UI in a root and returns what a `StringRenderer` wrote. See [Server rendering and hydration](server-rendering.md).

Custom renderers are possible: implement `Renderer` (`ScheduleRender` and `Mark`) and mount with `hx.NewWithRenderer`.

## Requests

`hx.Do` uses `XMLHttpRequest`, not `fetch`, because it is the only one that tells you how much of a body has been uploaded. It blocks the goroutine that calls it, and the answer is delivered from the browser through a channel, which is why it cannot be waited for inside an event handler. `Action` wraps it: it runs the function on a goroutine, keeps the state in signals, and cancels the previous run when you start a new one.

The core does not import `encoding/json` or `net/url`: Go keeps every package a program imports even if it is never used, and they would add about 350 KB to every WASM file. JSON lives in `hxjson`.

## Hydration

`Hydrate` builds the node tree exactly as a normal render does, but instead of creating DOM elements it takes the ones the server wrote. The tree already says how many elements each node produces (a normal node one, a `Noop` the sum of its children), so the HTML needs no markers.

For each node, in order:

1. It checks that the element has the same tag. If not, the element is removed and the node is rendered by the browser as new.
2. It records that the text, attributes, classes and styles that match are already applied, using the same "committed value" that the renderer keeps, so the first render writes nothing. What does not match is left pending, and the renderer sets it.
3. It does the same for the children, matching them with the child elements one by one. If the count differs, the children are rendered by the browser.
4. It attaches the event listeners.

After that the nodes are in the same state as if the browser had created them, and everything above (signals, bindings, `EachKeyed`...) works unchanged.

The core is split by build tags: `dom_wasm.go` and `dom_native.go` give the shared code the DOM-side fields and functions (or empty versions of them on a server).
