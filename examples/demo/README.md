# Hix demo

A big example that uses almost everything in Hix. It is a separate Go module that depends on the library in the parent folder (`replace github.com/deltegui/hx => ../..`). It runs in the browser only, so it imports the engine `hx/wasm`.

```sh
make serve      # builds main.wasm and serves http://localhost:8080
```

Tabs:

| Tab | What it shows |
|---|---|
| Reactivity | Signals, computed values chained, `Untrack`, two-way binding, `OnCleanup` with an interval |
| Lifecycle | `CreateRoot`, node-owned effects with `IfFn`, reusing a node with `Show` |
| Lists | `Each` vs `EachKeyed` side by side, a `Noop` between siblings, reusing nodes in `Body`, `EachMap` vs `EachMapSorted`, text with children |
| DOM details | Classes, attributes, styles, replacing handlers, SVG |
| Forms | Inputs, textarea, select, checkbox, keyboard events |
| Stress | 100 to 5000 rows with a signal and a binding each, and a keyed list with reverse and rotate, with timings |
| Checks | 39 checks that run against the real DOM |

## Checks

Open the **Checks** tab and press *Run checks*. Opening `http://localhost:8080/#checks-auto` runs them when the page loads and writes the result in `document.body.dataset.checks` (for example `30/39 checks passed`), which makes it possible to run them from a headless browser with the Chrome DevTools Protocol.

## Files

- `main.go` – the page layout and the tabs (an effect rebuilds the content when the tab changes).
- `reactivity.go`, `lifecycle.go`, `lists.go`, `dom.go`, `forms.go`, `stress.go` – one file per tab.
- `checks.go` – the checks.
- `helpers.go` – small helpers (cards, logger, `nextFrame`).
