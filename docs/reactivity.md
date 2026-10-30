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

Effects run **synchronously** inside `Set`. When `Set` returns, every effect that depended on the signal has already run.

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

Reading a signal with `Get` **outside** any effect, and outside `Untrack` or a handler, works but logs a warning with the caller's file and line. It usually means you forgot to wrap something in an effect.

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
| `CreateRoot`, `OnCleanup`, `Untrack` | own and stop effects | do nothing (there is nothing to stop) |

Because nothing is global, many UIs can be rendered at the same time. See [Server rendering](server-rendering.md).

## Rules of thumb

- Keep state in signals, derive with `Computed`, react with effects.
- Avoid writing to a signal from an effect that reads it. It is a loop. Hix does not let an effect re-enter itself while it is already re-running, but it can still run more times than you expect.
- There is no batching. Two `Set` calls in a row run the dependent effects twice. DOM work is still done once, because rendering is batched per animation frame (see [How it works](how-it-works.md)).
- Prefer `node.Effect` over `hx.EffectFunc` for anything that touches a node.
