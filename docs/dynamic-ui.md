# Dynamic UI

This page is about UI that changes: text that follows a signal, parts that appear and disappear, and lists.

## The two ways to react

1. **Bindings** change one property of one node: `BindText`, `BindValue`, and your own `node.Effect`.
2. **Rebuilding the children** of a node inside an effect, for structure that changes (conditionals, lists).

### Bindings

```go
name := hx.Signal("world")

hx.H1().BindText(hx.Computed(func() string { return "Hello, " + name.Get() }))
```

`BindText` takes anything with a `Get() string`: a `Signal[string]` directly, or a `Computed` when you need to format.

For anything else, write the effect yourself on the node:

```go
box := hx.Div().Text("box")

box.Effect(func() {
	box.ClassIf("dark", dark.Get())
	box.Style("font-size", size.Get()+"px")
})
```

`node.Effect` is the effect version of a binding. It runs now and again when a signal it read changes, and **it belongs to the node**: when the node leaves the tree the effect stops, and if the node comes back it runs again. You do not need to dispose it.

### Rebuilding children

When the shape of the UI depends on a signal, give the node an effect that sets its body:

```go
todos := hx.Signal([]string{"write docs"})

list := hx.Ul()
list.Effect(func() {
	list.Body(hx.Each(todos, func(i int, todo string) hx.INode {
		return hx.Li().Text(todo)
	}))
})
```

Each time `todos` changes the effect runs and sets a new body. `hx.Each` calls `todos.Get()`, so the effect depends on it.

The old children are removed and their effects and listeners are released.

## Conditionals

```go
visible := hx.Signal(true)

box := hx.Div()
box.Effect(func() {
	box.Body(hx.If(visible, hx.P().Text("Now you see me")))
})
```

| Function | Result |
|---|---|
| `If(cond, node)` | `node` if `cond` is true, otherwise `nil` (nothing) |
| `Show(cond, a, b)` | `a` if true, otherwise `b` |
| `IfFn(cond, build)` | like `If`, but `build()` is only called when true |
| `ShowFn(cond, buildA, buildB)` | like `Show`, only the chosen function is called. `buildB` can be `nil` |

`cond` is anything with `Get() bool`: a signal, a `Computed`, or `hx.Value(true)` for a constant.

**`If`/`Show` versus `IfFn`/`ShowFn`.** With `If` and `Show` the nodes are already built when the function is called. The one that is not used is *discarded*: its bindings and effects are released at once, so nothing leaks, but you still paid for building it. Use `IfFn`/`ShowFn` when building is expensive or has side effects, or when the unused branch should not exist at all:

```go
box.Effect(func() {
	box.Body(hx.ShowFn(loggedIn,
		func() hx.INode { return dashboard() },
		func() hx.INode { return loginForm() },
	))
})
```

## Lists

```go
hx.Each(items, func(index int, item T) hx.INode)
hx.EachMap(m, func(key K, value V) hx.INode)         // order is NOT defined
hx.EachMapSorted(m, func(key K, value V) hx.INode)   // ordered by key (K must be ordered)
hx.EachKeyed(items, key, render)                     // rows are kept per key, see below
```

They take a signal (or anything with `Get()`) holding a slice or map and return a fragment (`Noop`) with one node per item. Call them inside an effect, as shown above. For rows that keep their state see `EachKeyed` below.

Be careful with `EachMap`: Go maps have no order, so the rows can change position between renders. Use `EachMapSorted` if you show the data to people.

### Keeping the rows: `EachKeyed`

`Each` rebuilds every row when the list changes. That is simple and fast enough for hundreds of items, but a row loses its state: text typed in an input, focus, scroll position, a running animation.

`EachKeyed` builds a row **once per key** and reuses it:

```go
type Todo struct {
	ID    int
	Title string
}

todos := hx.Signal([]Todo{{1, "write docs"}, {2, "ship it"}})

list := hx.Ul().Body(hx.EachKeyed(todos,
	func(t Todo) int { return t.ID }, // the identity of an item
	func(id int, todo hx.Gettable[Todo], index hx.Gettable[int]) hx.INode {
		return hx.Li().Body(
			hx.Span().BindText(hx.Computed(func() string {
				return fmt.Sprintf("%d. %s ", index.Get()+1, todo.Get().Title)
			})),
			hx.Input().Placeholder("notes"),
		)
	}))
```

What happens when `todos` changes:

- A key that is new gets a new row (`render` runs).
- A key that was already there keeps its row and its DOM element. Its `todo` and `index` change, so whatever is bound to them updates.
- A key that is gone loses its row. Everything created inside `render` (bindings, effects, computed values) is stopped.
- Rows that changed position are moved with the fewest DOM moves. The ones that did not move are not touched, so they keep focus.

Rules:

- **Use it directly as a child**, as in the example. It follows the signal by itself, so it does **not** go inside an effect like `Each` does.
- **`todo` and `index` are values that change.** Read them with `Get()` inside a binding or an effect. If you read `todo.Get()` once while building the row, you get the value it had at that moment.
- **Keys must be unique.** With a repeated key, Hix logs a warning and renders the later item as a row that is rebuilt every time.
- `render` may return `nil` to skip a row.
- Effects and computed values created inside `render` are owned by the row. You do not have to dispose them.

Use `Each` when the rows are cheap and stateless. Use `EachKeyed` when rows have state, when the list is long and changes a little, or when rows are reordered.

### Updating one item without rebuilding the list (with `Each`)

If you prefer `Each`, you can still avoid rebuilding on every change: give each row its own signal and bind to it. Only the row's text updates:

```go
type Row struct{ Label string; Count *hx.SignalT[int] }

rows := hx.Signal([]Row{ ... })   // changes only when rows are added or removed

list.Effect(func() {
	list.Body(hx.Each(rows, func(i int, row Row) hx.INode {
		return hx.Li().BindText(hx.Computed(func() string {
			return fmt.Sprintf("%s: %d", row.Label, row.Count.Get())
		}))
	}))
})

row.Count.Update(func(n int) int { return n + 1 }) // the list is not rebuilt, only that <li>'s text changes
```

## Reusing nodes

If you pass to `Body` a node that is already a child, it is **moved**, not recreated. That keeps its DOM element, its state and its listeners:

```go
chips := []hx.INode{a, b, c}      // built once
holder.BodyList(chips)

rand.Shuffle(len(chips), func(i, j int) { chips[i], chips[j] = chips[j], chips[i] })
holder.BodyList(chips)            // same nodes, new order
```

A node that is removed and later added again also works: its bindings and node effects stop while it is away and start again when it returns, with the current values of the signals.

## What happens when a node is removed

When `Body` replaces children, or an `If` turns false, the removed nodes and all their descendants are *deactivated*:

- their DOM listeners are removed,
- their bindings and `node.Effect`s stop and unsubscribe from their signals.

You do not need to clean anything. If you create effects with `hx.EffectFunc` instead, see [Reactivity: ownership](reactivity.md#who-stops-an-effect-ownership).

## Patterns

**Derived UI state.** Put the logic in a `Computed`, bind the result:

```go
remaining := hx.Computed(func() string { return fmt.Sprintf("%d left", countOpen(todos.Get())) })
hx.Span().BindText(remaining)
```

**Component = a function that returns a node.** State lives in signals created inside the function (private) or passed in (shared):

```go
func counter(label string) hx.INode {
	n := hx.Signal(0)
	return hx.Div().Body(
		hx.Span().BindText(hx.Computed(func() string { return fmt.Sprintf("%s: %d", label, n.Get()) })),
		hx.Button().Text("+").OnClick(func(hx.EventContext) { n.Update(func(v int) int { return v + 1 }) }),
	)
}
```

**Do not read signals while building, unless it is inside an effect.** `hx.Div().Text(count.Get())` reads once and never updates. Use `BindText` or an effect.
