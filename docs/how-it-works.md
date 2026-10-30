# How it works

You do not need this page to use Hix. It explains what happens under the hood so the behaviour is predictable.

## One API, two engines

1. **`hx`, the core.** Signals, computed values and effects (`signal.go`), the `INode` interface and the constructors (`Div`, `Span`...), the operators (`If`, `Each`, `EachKeyed`) and the state of requests (`Action`, `Resource`). Making the requests is `hx/hxhttp`. It knows nothing about the DOM or about HTML. Code that builds a UI imports only this, so a library of components, like a Bootstrap wrapper, works on the server and in the browser.
2. **An engine**, which makes the nodes and decides what a render is. A program imports the one it needs, and the engine registers itself (`hx.UseEngine`):
   - **`hx/wasm`** (browser). A `VNode` tree and the code that applies it to the DOM (`element.go`, `diffrenderer.go`), and `Hydrate`.
   - **`hx/ssr`** (server). A plain tree that is written as HTML. No DOM code is linked.

The constructors ask the engine for the node: `hx.Div()` is `engine.Element("DIV", "")`. The two meet in bindings and `node.Effect`: in the browser an effect that calls node methods like `Text` or `Class` and runs again when its signals change; on a server, a function that runs once.

A program has one engine. Mixing nodes of two engines panics, with a message.

## Tracking is the engine's choice

The browser needs signals to follow who reads them. A server does not, since it builds the UI once. The ssr engine turns tracking off: a `Signal` is a plain value, a `Computed` is calculated on every read, an effect runs once, `OnMount` is ignored and `CreateRoot`/`OnCleanup` do nothing. There is no global state left, so many UIs are rendered at the same time with no lock.

## Dependency tracking

There is one global "current effect". While an effect runs, every `Signal.Get` registers that effect as a subscriber of the signal. `Set` runs the subscribers synchronously. Before an effect re-runs, it unsubscribes from everything and its child effects are disposed, so dependencies are always the ones from the last run.

Effects belong to an owner (a parent effect, a root or a node) so they can be stopped. See [Reactivity](reactivity.md#who-stops-an-effect-ownership).

## A node is a description of what you want

A `VNode` stores the **desired state**: text, attributes, classes, styles, listeners, children. Each property remembers two values, the one already in the DOM and the pending one, plus a flag saying what changed. Calling a method only edits that description and marks the node "dirty". (This is the node of the browser engine. The ssr node only keeps the values, since nothing is drawn twice.)

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

- The Go DOM library `honnef.co/go/js/dom` identifies the type of every value it returns with dozens of JavaScript calls. Hix avoids it when creating and removing nodes and the text node (`wasm/domraw.go`), which made rendering 5,000 rows about 3x faster and removing them about 20x faster. Elements you are likely to need typed (inputs, textareas, selects, links, buttons, forms, images...) still come back with their specific type.
- Every property change is a single DOM call. Styles are written together in the `style` attribute.
- `Each` rebuilds its rows when the list signal changes. Use `EachKeyed` to keep them (see [Dynamic UI](dynamic-ui.md#keeping-the-rows-eachkeyed)), or bind each row to its own signal.

## StringRenderer (tests)

`wasm.StringRenderer` implements the same `Renderer` interface as the DOM renderer but writes HTML text. The browser engine keeps it so its nodes can be tested on any platform, without a browser. It is not what a server uses: `hx/ssr` has its own nodes and its own writer. The two write the same HTML (tags in lower case, `Noop` nodes transparent, attributes, classes and styles sorted), because hydration needs the tree the server wrote and the tree the browser builds to be the same. See [Server rendering and hydration](server-rendering.md).

Custom renderers are possible: implement `Renderer` (`ScheduleRender` and `Mark`) and mount with `wasm.NewWithRenderer`.

## Requests

`hxhttp.Do` uses `XMLHttpRequest`, not `fetch`, because it is the only one that tells you how much of a body has been uploaded. It blocks the goroutine that calls it, and the answer is delivered from the browser through a channel, which is why it cannot be waited for inside an event handler. `Action` wraps it: it runs the function on a goroutine, keeps the state in signals, and cancels the previous run when you start a new one.

`hx` and `hxhttp` do not import `encoding/json` or `net/url`: Go keeps every package a program imports even if it is never used, and `encoding/json` alone adds more than 800 KB to a TinyGo WASM file (measured). JSON lives in `hxjson`, on top of `hxhttp`.

## Hydration

`Hydrate` builds the node tree exactly as a normal render does, but instead of creating DOM elements it takes the ones the server wrote. The tree already says how many elements each node produces (a normal node one, a `Noop` the sum of its children), so the HTML needs no markers.

For each node, in order:

1. It checks that the element has the same tag. If not, the element is removed and the node is rendered by the browser as new.
2. It records that the text, attributes, classes and styles that match are already applied, using the same "committed value" that the renderer keeps, so the first render writes nothing. What does not match is left pending, and the renderer sets it.
3. It does the same for the children, matching them with the child elements one by one. If the count differs, the children are rendered by the browser.
4. It attaches the event listeners.

After that the nodes are in the same state as if the browser had created them, and everything above (signals, bindings, `EachKeyed`...) works unchanged.

The browser engine is split by build tags: `wasm/dom_wasm.go` has the part that talks to the DOM, and `wasm/dom_native.go` gives the rest of the package empty versions of it, so the nodes can be tested without a browser. The core (`hx`) has no build tags at all. A test renders the same UI with both engines and requires the same HTML, since the hydration depends on it.
