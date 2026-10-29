# API reference

Everything is in the package `github.com/deltegui/hx`. The same text is available with `GOOS=js GOARCH=wasm go doc -all github.com/deltegui/hx`.

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

### Events

| | |
|---|---|
| `On(event Event, handler func(EventContext))` | One handler per event; calling it again replaces it. |
| `OnClick`, `OnInput`, `OnChange`, `OnKeyUp`, `OnKeyDown`, `OnSubmit`, `OnFocus`, `OnBlur` | Shortcuts for `On`. |

Event names: `EventClick EventInput EventChange EventKeyUp EventKeyDown EventSubmit EventFocus EventBlur EventDblClick EventMouseEnter EventMouseLeave`. `Event` is a string, so `hx.Event("scroll")` works.

`EventContext` has two fields: `Target *VNode` and `Event dom.Event`.

### Bindings and effects

| | |
|---|---|
| `BindText(g Gettable[string])` | Text follows the signal. |
| `BindValue(g Gettable[string])` | Value follows the signal (a textarea also gets it as text). |
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

## Renderers

| | |
|---|---|
| `Renderer` | `interface{ ScheduleRender(); Mark(*VNode) }`. |
| `DiffRenderer` | The DOM renderer used by `New` and `NewFromId`. Batches changes per animation frame. |
| `StringRenderer` | Writes HTML. `Render(*VNode)`, `String()`, `Reset()`. |

## Compatibility aliases

`INode = *VNode`, and `InputVNode`, `TextAreaNode`, `OptionNode`, `AVNode`, `NoopNode` are all `VNode`. They exist so older code keeps compiling.
