# API reference

Hix is five packages. Unless a section says otherwise, a name is in `github.com/deltegui/hx`.

| Package | What is in it |
|---|---|
| `hx` | The API a UI is written against: signals, effects, `INode` and the constructors, operators, `Action`, `Resource`, `OnMount`. Builds on every platform. |
| `hx/wasm` | The engine of the browser: mounting, hydration, the DOM renderer. The part that touches the DOM only exists for `GOOS=js GOARCH=wasm`. |
| `hx/ssr` | The engine of the server: `ssr.Render`. Builds on every platform. |
| `hx/hxhttp` | Requests. `Do` only works in the browser; the types build everywhere. |
| `hx/hxjson` | JSON on top of `hxhttp`. |

The same text is available with `go doc -all` on each package (use `GOOS=js GOARCH=wasm` for `hx/wasm`).

## Signals

| | |
|---|---|
| `Signal[T](initial T) *SignalT[T]` | Create a reactive value. |
| `(*SignalT[T]) Get() T` | Read it. Inside an effect it subscribes the effect. |
| `(*SignalT[T]) Set(v T)` | Write it and run the effects that read it: when `Set` returns, or when the open handler, effect or `Batch` ends. Always notifies. |
| `(*SignalT[T]) Update(fn func(T) T)` | `Set(fn(current))`. Does not subscribe. |
| `(*SignalT[T]) Subscribers() int` | How many effects follow it. For tests. |
| `Computed[T comparable](fn func() T) *ComputedT[T]` | Value derived from other signals. Calculated again when read after a change, so it is never out of date. Wakes effects only when the result changes. |
| `(*ComputedT[T]) Get() T` | Read it. Subscribes like a signal. |
| `Value[T](v T) ValueWrapper[T]` | A constant that can be used as a `Gettable`. `Get`/`Set` are plain, not reactive. |
| `Gettable[T]` | `interface{ Get() T }`. Signals, computed values and `Value` satisfy it. |
| `Settable[T]` | `interface{ Set(T) }`. |

## Effects and ownership

| | |
|---|---|
| `EffectFunc(fn func()) *Effect` | Run `fn` now and again when a signal it read changes. |
| `(*Effect) Dispose()` | Stop it for good, with its child effects. |
| `(*Effect) Disposed() bool` | Whether it has been stopped. |
| `CreateRoot[T](fn func(dispose func()) T) T` | Run `fn` in a new root. Effects created in it, outside other effects, belong to the root. |
| `(*Root) Dispose()` | Stop the root's effects and run its cleanups. Safe to call twice. (The `dispose` argument of `CreateRoot` is this method.) |
| `OnCleanup(fn func())` | Run `fn` before the current effect re-runs or when it is disposed (or when the current root is). |
| `Untrack(fn func())` | Run `fn` without subscribing the current effect. |
| `RunCallback(fn func())` / `InCallback() bool` | Mark `fn` as running inside a callback of JavaScript (an event handler, a frame). Hix does it for its own; do it for the body of a `js.FuncOf` of yours. `hxhttp.Do` fails with `ErrInCallback` there instead of deadlocking. |
| `Batch(fn func())` | Run `fn` and hold the effects back until it ends: each affected effect runs once, with the final values. Nests. Event handlers and the end of an `Action` are already in one. |
| `UntrackGet[T](g Gettable[T]) T` | Read without subscribing. |

## Engines

The package `hx` has the API a UI is written against. A program also imports one engine, which makes the nodes:

| | |
|---|---|
| `hx/wasm` | The browser. Mounting and hydration (below). |
| `hx/ssr` | A server. `ssr.Render(build)` and `ssr.Node(root)` write HTML. |
| `Engine` | `interface{ Element(tag, namespace) INode; Fragment() INode; Raw(html) INode; OnMount(fn); IsServer() bool; Tracks() bool }`. What an engine implements. |
| `UseEngine(Engine)` | What an engine calls when it is imported. You only need it in a test; `wasm.Use()` and `ssr.Use()` do it for their engine. |

On a server (`Tracks()` is false) a signal is a plain value, a `Computed` is calculated on every read, an effect runs once, and `CreateRoot`, `OnCleanup` and `Untrack` do nothing. See [Reactivity](reactivity.md#on-a-server).

## Mounting (`hx/wasm`)

| | |
|---|---|
| `wasm.New(element js.Value) *wasm.VNode` | Mount on a DOM element. |
| `wasm.NewFromId(id string) *wasm.VNode` | Mount on the element with that id. |
| `wasm.NewWithRenderer(element, renderer)` / `wasm.NewFromIdWithRenderer(id, renderer)` | Same with a custom `wasm.Renderer`. |
| `wasm.NewWithoutMount(tag string, renderer Renderer) *wasm.VNode` | A node that is not in the DOM (for `wasm.StringRenderer`, in tests). |
| `wasm.Underlying(node hx.INode) js.Value` | The DOM element of a node, once rendered (undefined before). |
| `wasm.EventOf(ctx hx.EventContext) js.Value` | The event of the browser that a handler received. |
| `wasm.Key(ctx hx.EventContext) string` | The key of a keyboard event. |
| `wasm.PreventDefault(ctx)`, `wasm.StopPropagation(ctx)` | What the names say, on the event of a handler. |

## Elements

All return `INode`.

`A Article Aside Audio Br Button Canvas Code Div Em Footer Form H1 H2 H3 H4 H5 H6 Header Hr I Img Input Label Li Main Nav Noop Ol Option P Path Pre Section Select Small Source Span Strong Svg Table TBody Td TextArea TFoot Th THead Tr Ul Video`

| | |
|---|---|
| `El(tag string) INode` | Any tag. |
| `Noop() INode` | A fragment: renders its children, no element. |
| `Raw(html string) INode` | HTML written as it is, with no escaping, and no element of its own. **For trusted HTML only.** See [Raw HTML](building-the-ui.md#raw-html). |
| `Svg()`, `Path()` | Created in the SVG namespace. |

## INode methods

All of them return `INode`, so they can be chained in any order, except the ones marked "returns a value".

### Content and children

| | |
|---|---|
| `Text(string)` | Text of the node. |
| `Textf(format, args...)` | Text with a format. |
| `Body(children ...INode)` | Replace the children. `nil`s are skipped. Nodes that were already children are moved. |
| `BodyList([]INode)` | Same with a slice. |

### Attributes, classes, styles

| | |
|---|---|
| `Attribute(key, value)` / `RemoveAttribute(key)` | Empty values are valid. |
| `GetAttribute(key) string` / `HaveAttribute(key, value) bool` | Returns a value. State after the next render. |
| `Class(names...)` / `RemoveClass(names...)` | Empty names are skipped. A name with a space is ignored with a warning: pass them separately. |
| `ClassIf(name, on bool)` | Add or remove. |
| `HaveClass(name) bool` | Returns a value. |
| `Style(key, value)` / `RemoveStyle(key)` | |
| `GetStyle(key) string` / `HaveStyle(key, value) bool` | Returns a value. |
| `Id(string)` | |
| `Href(v)`, `Src(v)`, `Placeholder(v)`, `Type(v)`, `Selected()` | Shortcuts for `Attribute`. |
| `Value(string)` | The `value` property of inputs, textareas, selects and options. |
| `Checked(bool)` | Whether a checkbox or radio button is checked. |

### Events

| | |
|---|---|
| `On(event Event, handler func(EventContext))` | One handler per event; calling it again replaces it. It does not replace the binding of the event (`BindOnInput`...), which runs first. |
| `OnClick`, `OnInput`, `OnChange`, `OnKeyUp`, `OnKeyDown`, `OnSubmit`, `OnFocus`, `OnBlur` | Shortcuts for `On`. |

Event names: `EventClick EventInput EventChange EventKeyUp EventKeyDown EventSubmit EventFocus EventBlur EventDblClick EventMouseEnter EventMouseLeave`. `Event` is a string, so `hx.Event("scroll")` works.

`EventContext` has two fields: `Target INode` and `Event any`, the event of the browser (`nil` on a server). In browser code `wasm.EventOf(ctx)` returns it as a `js.Value`.

### Bindings and effects

| | |
|---|---|
| `BindText(g Gettable[string])` | Text follows the signal. |
| `BindValue(g Gettable[string])` | Value follows the signal (a textarea also gets it as text). |
| `BindChecked(g Gettable[bool])` | The checked state of a checkbox or radio follows the signal. |
| `BindOnChecked(s Settable[bool])` | Write the checked state to the signal on `change`. |
| `BindOnInput(s Settable[string])` | Write the element's value to the signal on `input`. |
| `BindOnChange(s Settable[string])` | Same on `change`. |
| `Effect(fn func())` | An effect owned by the node. |

### Other

| | |
|---|---|
| `OnRendered(fn func())` | Runs `fn` when the node is in the page and rendered, so `wasm.Underlying(node)` works. Again each time the node is put back. Never on a server. |
| `OnRemoved(fn func())` | Runs `fn` when a node that `OnRendered` ran for leaves the tree. |

## Conditionals and lists

| | |
|---|---|
| `Dynamic(build func() INode) INode` | A part of the UI built again when the signals `build` reads change. `build` can return `nil`. |
| `If(cond Gettable[bool], child INode) INode` | `child` or `nil`. The child is already built, the unused one is discarded. |
| `Show(cond, a, b INode) INode` | `a` or `b`. |
| `IfFn(cond, build func() INode) INode` | Builds only when true. |
| `ShowFn(cond, a, b func() INode) INode` | Builds only the used path. `b` may be `nil`. |
| `Each[T](src Gettable[[]T], render func(i int, v T) INode) INode` | One node per item. |
| `EachMap[K comparable, T](src Gettable[map[K]T], render func(k K, v T) INode) INode` | One node per entry, **unspecified order**. |
| `EachMapSorted[K cmp.Ordered, T](src, render) INode` | Ordered by key. |
| `EachKeyed[T any, K comparable](src Gettable[[]T], key func(T) K, render func(key K, item Gettable[T], index Gettable[int]) INode) INode` | Rows are built once per key and reused: they keep their DOM element and state. Use it directly as a child, not inside an effect. |

## Server rendering and hydration

| | |
|---|---|
| `ssr.Render(build func() INode) string` | The HTML of the UI. Runs effects once. Safe to call from many goroutines at the same time. Builds everywhere. |
| `StateScript(id, json string) string` | A `<script type="application/json">` carrying `json`, escaped. Builds everywhere. |
| `OnMount(fn func())` | Runs `fn` only in the browser, after hydration. Ignored on the server. |
| `IsServer() bool` | `true` on the server, `false` in the browser. |
| `wasm.Hydrate(id string, build func() INode) (dispose func())` | Adopts the server's HTML in the element with that id. Browser only. |
| `wasm.HydrateElement(mount js.Value, build) (dispose func())` | Same for an element you have. Browser only. |
| `wasm.ReadState(id string) string` | The text of the element with that id (what `StateScript` wrote). Browser only. |

## Requests

Package `hxhttp`. Browser only for the request itself: on a server `hxhttp.Do` returns `hxhttp.ErrNotInBrowser`, but the types build everywhere. See [Forms and requests](forms-and-requests.md).

| | |
|---|---|
| `Do(ctx, Request) (Response, error)` | Make a request and wait for the answer. Blocks: use a goroutine or an `Action`. |
| `Request` | `Method`, `URL`, `Header`, `Body`, `Timeout`, `Credentials`, `OnProgress` |
| `Response` | `Status`, `StatusText`, `Header`, `OK()`, `Text()`, `Bytes()`, `Err()` |
| `HTTPError` | `Status`, `StatusText`, `Body`. What `Response.Err()` returns for a status that is not 2xx |
| `Progress` | `Upload`, `Loaded`, `Total`, `Fraction()` |
| `Body` | what a request sends |
| `TextBody(contentType, text)` | text |
| `FormValues(map[string][]string)` | `application/x-www-form-urlencoded` (a `url.Values` works) |
| `FormOf(form hx.INode)` | the fields and files of a `<form>` in the page (`multipart/form-data`) |
| `NewMultipart()` | `.Field(name, value)`, `.File(name, file)`, `.Bytes(name, filename, type, data)`, `.Body()` |
| `FilesOf(input hx.INode) []File` | the files chosen in a file input |
| `File` | `Name`, `Size`, `Type`, `Bytes(ctx) ([]byte, error)` |
| `hx.IsCanceled(err)`, `hx.IsTimeout(err)` | what ended a request (these two are in `hx`) |
| `ErrNotInBrowser` | `Do` was called on a server |
| `ErrInCallback` | `Do` was called inside an event handler, where it cannot wait for the answer |

### `hxjson`: package `github.com/deltegui/hx/hxjson`

JSON, in its own package so programs that do not use it do not carry `encoding/json` (more than 800 KB with TinyGo).

| | |
|---|---|
| `hxjson.Body(v) (*hxhttp.Body, error)` | a body that sends `v` as JSON |
| `hxjson.Decode(response, &v) error` | read the JSON body of an answer |
| `hxjson.Send[T](ctx, method, url, body) (T, error)` | send JSON, decode the answer. A status that is not 2xx is an `*hxhttp.HTTPError` |
| `hxjson.Get[T](ctx, url)`, `hxjson.Post[T](ctx, url, body)` | shortcuts |

### Actions and resources (package `hx`)

| | |
|---|---|
| `Action[A, R](fn func(ctx, A) (R, error)) *ActionT[A, R]` | The state of something the user triggers |
| `(*ActionT) Run(arg A)` | Start it. If it is running, that run is canceled: the last call wins |
| `Cancel()`, `Reset()` | Stop the run / stop it and forget the error and the result |
| `Pending()`, `Error()`, `Value()` | Signals (`Gettable`) |
| `OnSuccess(fn)`, `OnError(fn)` | Called when a run ends, after the state is updated. Return the action, to chain |
| `Resource[T](fetch) *ResourceT[T]` | Data from a server, loaded when the UI is in the browser |
| `ResourceWith[T](initial, fetch)` | Same, starting from `initial` and not loading by itself |
| `ResourceFor[S, T](source, fetch)` | Loads again each time the `source` signal changes |
| `ResourceForWith[S, T](initial, source, fetch)` | Same, starting from `initial` (what the server rendered): the first load is skipped |
| `(*ResourceT) Value()`, `Loading()`, `Error()` | Signals |
| `Refetch()`, `Cancel()`, `Set(v)` | Load again / stop / replace the value |

## Renderers (`hx/wasm`)

| | |
|---|---|
| `wasm.Renderer` | `interface{ ScheduleRender(); Mark(*VNode) }`. |
| `wasm.DiffRenderer` | The DOM renderer used by `wasm.New` and `wasm.NewFromId`. Batches changes per animation frame. |
| `wasm.StringRenderer` | Writes the HTML of a `wasm.VNode`. For tests. A server uses `ssr.Render`. |

- `hx.SafeAttribute(key, value)`, `hx.UnsafeURL`, `hx.AllowURLScheme(scheme)`: URL attributes (`href`, `src`, `action`...) only keep `http`, `https`, `mailto`, `tel` and relative URLs; anything else becomes `UnsafeURL`. Both engines apply it.
