# Reactivity

Hix uses the same model as Solid.js: **signals** hold state, **computed values** derive state, and **effects** react to changes. Dependencies are found automatically, you never list them.

## Signals

```go
count := hx.Signal(0)

count.Get()                                  // read
count.Set(5)                                 // write
count.Update(func(n int) int { return n+1 }) // write using the current value
```

A signal can hold any type: numbers, strings, structs, slices, maps. To change a slice or a map, **create a new one** instead of changing it in place. Hix does not look inside the value:

```go
items := hx.Signal([]string{"a"})

// wrong: the slice is the same, nothing reacts if you only mutate it
// right: Set with a new slice
items.Update(func(l []string) []string {
	return append(slices.Clone(l), "b")
})
```

`Set` always notifies. It does not compare the new value with the old one.

## Effects

An effect is a function that runs once, remembers every signal it read, and runs again when one of them changes.

```go
name := hx.Signal("world")

hx.EffectFunc(func() {
	fmt.Println("hello", name.Get()) // prints now, and after every name.Set
})

name.Set("Hix") // prints "hello Hix"
```

Effects run **synchronously**: when `Set` returns, every effect that depended on the signal has already run. The exception is inside an event handler, another effect or a [`Batch`](#batching-several-changes-at-once), where they run when that one ends, all together.

Dependencies are tracked again on each run. If an effect reads `a` only when `flag` is true, it only depends on `a` while that is true.

## Computed values

`Computed` is a read-only signal calculated from others:

```go
price := hx.Signal(10)
qty := hx.Signal(3)
total := hx.Computed(func() int { return price.Get() * qty.Get() })

total.Get() // 30
qty.Set(4)
total.Get() // 40
```

A computed value only notifies its own subscribers when the result is **different** from the previous one. Because of that, its type must be `comparable` (no slices or maps). For those, use a plain signal and update it yourself.

Computed values can depend on other computed values.

A computed value is calculated again **when it is read**, after something it depends on changed, so what you read is never out of date, not even in the middle of a [`Batch`](#batching-several-changes-at-once). If the new result is the same, the effects that read it do not run.

### Never half a change

When one change reaches an effect through several paths, the effect runs **once** and sees all of it:

```go
a := hx.Signal(1)
b := hx.Computed(func() int { return a.Get() + 1 })
c := hx.Computed(func() int { return a.Get() * 10 })

hx.EffectFunc(func() { fmt.Println(b.Get(), c.Get()) })

a.Set(2) // prints "3 20". Never "3 10": b and c are both up to date
```

How: a `Set` first marks everything that depends on it, and only then runs the effects, each at most once. An effect runs only if something it read really has a new value.

## Batching several changes at once

`hx.Batch` holds the effects back until it ends. Any number of `Set` calls inside it run each affected effect **once**, with the final values:

```go
hx.Batch(func() {
	first.Set("Ada")
	last.Set("Lovelace")
	age.Set(36)
}) // the effects that read them run here, once
```

- Batches nest: the effects run when the outermost one ends, even if the function panics.
- A `Computed` read inside the batch is current.
- **Event handlers already run in a batch**, and so does the end of an `Action` (its result and `Pending` change together). You need `Batch` for changes that start somewhere else: a timer, a callback from JavaScript, a goroutine of yours.
- A `Set` inside an effect also waits: the effects that depend on it run when the one that is running ends. An effect does not run itself again for a signal it wrote.

## Reading without subscribing

```go
hx.EffectFunc(func() {
	a.Get()                   // the effect depends on a
	bValue := hx.UntrackGet[int](b) // reads b without depending on it
	fmt.Println(bValue)
})

hx.Untrack(func() {
	// nothing read in here is tracked
})
```

Event handlers already run inside `Untrack`, so you can call `Get` freely in them.

Reading a signal with `Get` **outside** any effect, and outside `Untrack` or a handler, works but logs a warning with the caller's file and line. It usually means you forgot to wrap something in an effect: the value is read once and the UI never follows it.

While you develop, build with the tag `hxstrict` (`go build -tags hxstrict`, `go test -tags hxstrict ./...`) and the warning becomes a `panic` with the same file and line, so a mistake like that cannot go unnoticed. What is fine stays fine: reading inside an effect, a binding, `Untrack`, `UntrackGet` or a handler. The demo and the examples run clean in strict mode.

## Who stops an effect? Ownership

An effect that is never stopped leaks: it keeps running and keeps its signals and nodes alive. Hix gives every effect an owner that stops it:

| You create it with... | It is stopped when... |
|---|---|
| `node.Effect(fn)`, `BindText`, `BindValue` | the node leaves the tree (it starts again if the node comes back) |
| `hx.EffectFunc` **inside another effect** | the parent effect runs again or is disposed |
| `hx.EffectFunc` **inside `hx.CreateRoot`** | you call the root's `dispose` function |
| `hx.EffectFunc` anywhere else | never, unless you call `effect.Dispose()` yourself |

### Roots

`CreateRoot` groups effects so you can stop them together:

```go
dispose := hx.CreateRoot(func(dispose func()) func() {
	hx.EffectFunc(func() { fmt.Println(count.Get()) })
	hx.EffectFunc(func() { fmt.Println(name.Get()) })
	return dispose
})

// later
dispose() // both effects stop; calling it again does nothing
```

Wrapping the whole application in `CreateRoot` is a good habit. See [`examples/demo/main.go`](../examples/demo/main.go).

### Cleaning up

`OnCleanup` registers a function that runs before the effect runs again and when it is disposed. Use it to undo what the effect did:

```go
hx.EffectFunc(func() {
	if !running.Get() {
		return
	}
	cb := js.FuncOf(func(js.Value, []js.Value) any { ticks.Update(inc); return nil })
	id := js.Global().Call("setInterval", cb, 1000)

	hx.OnCleanup(func() {
		js.Global().Call("clearInterval", id)
		cb.Release()
	})
})
```

Inside `CreateRoot` (and outside any effect), `OnCleanup` runs when the root is disposed.

### Disposing by hand

```go
e := hx.EffectFunc(func() { ... })
e.Dispose() // never runs again; also disposes its child effects
```

## On a server

All of the above is what the browser engine (`hx/wasm`) does. The server engine (`hx/ssr`) builds the UI once and writes it, so it does not follow who reads what. The same code gives the same first render, but the reactivity is switched off:

| | Browser | Server |
|---|---|---|
| `Signal` | a value that notifies | a plain value: `Get` and `Set` just read and write |
| `Computed` | calculated when its signals change | calculated again on every `Get` |
| `EffectFunc`, `node.Effect`, `Bind*` | run again when a signal changes | run once |
| `CreateRoot`, `OnCleanup`, `Untrack`, `Batch` | own and stop effects, hold them back | do nothing (there is nothing to stop or to hold back) |

Because nothing is global, many UIs can be rendered at the same time. See [Server rendering](server-rendering.md).

## Rules of thumb

- Keep state in signals, derive with `Computed`, react with effects.
- Avoid writing to a signal from an effect that reads it. It is a loop. Hix does not let an effect re-enter itself while it is already re-running, but it can still run more times than you expect.
- Two `Set` calls in a row, outside a handler, run the dependent effects twice. Use `hx.Batch` to run them once. DOM work is done once per animation frame either way (see [How it works](how-it-works.md)).
- Prefer `node.Effect` over `hx.EffectFunc` for anything that touches a node.
