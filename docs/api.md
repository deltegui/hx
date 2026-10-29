# API reference

Everything is in the package `github.com/deltegui/hx`. The parts that need the browser are marked "browser only" and only exist for `GOOS=js GOARCH=wasm`; the rest builds on every platform. The same text is available with `GOOS=js GOARCH=wasm go doc -all github.com/deltegui/hx`.

## Signals

| | |
|---|---|
| `Signal[T](initial T) *SignalT[T]` | Create a reactive value. |
| `(*SignalT[T]) Get() T` | Read it. Inside an effect it subscribes the effect. |
| `(*SignalT[T]) Set(v T)` | Write it and run the effects that read it, now. Always notifies. |
| `(*SignalT[T]) Update(fn func(T) T)` | `Set(fn(current))`. Does not subscribe. |
| `Computed[T comparable](fn func() T) *ComputedT[T]` | Value derived from other signals. Notifies only when the result changes. |
| `(*ComputedT[T]) Get() T` | Read it. Subscribes like a signal. |
| `Value[T](v T) ValueWrapper[T]` | A constant that can be used as a `Gettable`. `Get`/`Set` are plain, not reactive. |
| `Gettable[T]` | `interface{ Get() T }`. Signals, computed values and `Value` satisfy it. |
| `Settable[T]` | `interface{ Set(T) }`. |

## Effects and ownership

| | |
|---|---|
| `EffectFunc(fn func()) *Effect` | Run `fn` now and again when a signal it read changes. |
| `(*Effect) Dispose()` | Stop it for good, with its child effects. |
| `CreateRoot[T](fn func(dispose func()) T) T` | Run `fn` in a new root. Effects created in it, outside other effects, belong to the root. |
| `(*Root) Dispose()` | Stop the root's effects and run its cleanups. Safe to call twice. (The `dispose` argument of `CreateRoot` is this method.) |
| `OnCleanup(fn func())` | Run `fn` before the current effect re-runs or when it is disposed (or when the current root is). |
| `Untrack(fn func())` | Run `fn` without subscribing the current effect. |
| `UntrackGet[T](g Gettable[T]) T` | Read without subscribing. |

## Mounting

| | |
|---|---|
| `New(element dom.Element) *VNode` | Mount on a DOM element. |
| `NewFromId(id string) *VNode` | Mount on the element with that id. |
| `NewWithRenderer(element, renderer)` / `NewFromIdWithRenderer(id, renderer)` | Same with a custom `Renderer`. |
| `NewWithoutMount(tag string, renderer Renderer) *VNode` | A node that is not in the DOM (for `StringRenderer`). |

## Elements

All return `*VNode` (also available as `INode`).

`A Article Aside Audio Br Button Canvas Code Div Em Footer Form H1 H2 H3 H4 H5 H6 Header Hr I Img Input Label Li Main Nav Noop Ol Option P Path Pre Section Select Small Source Span Strong Svg Table TBody Td TextArea TFoot Th THead Tr Ul Video`

| | |
|---|---|
| `El(tag string) *VNode` | Any tag. |
| `Noop() *VNode` | A fragment: renders its children, no element. |
| `Svg()`, `Path()` | Created in the SVG namespace. |

## VNode methods

All of them return `*VNode`, so they can be chained in any order, except the ones marked "returns a value".

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
| `Class(names...)` / `RemoveClass(names...)` | |
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
| `On(event Event, handler func(EventContext))` | One handler per event; calling it again replaces it. |
| `OnClick`, `OnInput`, `OnChange`, `OnKeyUp`, `OnKeyDown`, `OnSubmit`, `OnFocus`, `OnBlur` | Shortcuts for `On`. |

Event names: `EventClick EventInput EventChange EventKeyUp EventKeyDown EventSubmit EventFocus EventBlur EventDblClick EventMouseEnter EventMouseLeave`. `Event` is a string, so `hx.Event("scroll")` works.

`EventContext` has two fields: `Target *VNode` and `Event dom.Event` (`any` on a server).

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
| `Underlying() dom.Element` | The DOM element, after the first render. |
| `AsVNode() *VNode` | The node itself. |

## Conditionals and lists

| | |
|---|---|
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
| `RenderToString(build func() INode) string` | The HTML of the UI. Runs effects once and releases them. Serialized. Builds everywhere. |
| `StateScript(id, json string) string` | A `<script type="application/json">` carrying `json`, escaped. Builds everywhere. |
| `OnMount(fn func())` | Runs `fn` only in the browser, after hydration. Ignored on the server. |
| `IsServer() bool` | `true` on the server, `false` in the browser. |
| `Hydrate(id string, build func() INode) (dispose func())` | Adopts the server's HTML in the element with that id. Browser only. |
| `HydrateElement(mount dom.Element, build) (dispose func())` | Same for an element you have. Browser only. |
| `ReadState(id string) string` | The text of the element with that id (what `StateScript` wrote). Browser only. |

## Requests

Browser only for the request itself: on a server `Do` returns `ErrNotInBrowser`, but the types build everywhere. See [Forms and requests](forms-and-requests.md).

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
| `FormOf(form *VNode)` | the fields and files of a `<form>` in the page (`multipart/form-data`) |
| `NewMultipart()` | `.Field(name, value)`, `.File(name, file)`, `.Bytes(name, filename, type, data)`, `.Body()` |
| `FilesOf(input *VNode) []File` | the files chosen in a file input |
| `File` | `Name`, `Size`, `Type`, `Bytes(ctx) ([]byte, error)` |
| `IsCanceled(err)`, `IsTimeout(err)` | what ended a request |
| `ErrNotInBrowser` | `Do` was called on a server |

### `hxjson`: package `github.com/deltegui/hx/hxjson`

JSON, in its own package so programs that do not use it do not carry `encoding/json` (about 250 KB).

| | |
|---|---|
| `hxjson.Body(v) (*hx.Body, error)` | a body that sends `v` as JSON |
| `hxjson.Decode(response, &v) error` | read the JSON body of an answer |
| `hxjson.Send[T](ctx, method, url, body) (T, error)` | send JSON, decode the answer. A status that is not 2xx is an `*hx.HTTPError` |
| `hxjson.Get[T](ctx, url)`, `hxjson.Post[T](ctx, url, body)` | shortcuts |

### Actions and resources

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
| `(*ResourceT) Value()`, `Loading()`, `Error()` | Signals |
| `Refetch()`, `Cancel()`, `Set(v)` | Load again / stop / replace the value |

## Renderers

| | |
|---|---|
| `Renderer` | `interface{ ScheduleRender(); Mark(*VNode) }`. |
| `DiffRenderer` | The DOM renderer used by `New` and `NewFromId`. Batches changes per animation frame. |
| `StringRenderer` | Writes HTML. `Render(*VNode)`, `String()`, `Reset()`. |

## Compatibility aliases

`INode = *VNode`, and `InputVNode`, `TextAreaNode`, `OptionNode`, `AVNode`, `NoopNode` are all `VNode`. They exist so older code keeps compiling.
