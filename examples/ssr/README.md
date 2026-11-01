# Server rendering and hydration

A Go server renders a page with Hix and sends plain HTML. The browser loads the WASM, which hydrates that HTML: it adopts the elements the server wrote instead of creating them, attaches the event listeners, and from then on the page works as a normal Hix page.

```sh
make serve      # builds the WASM and starts the server on http://localhost:8090
make check      # 57 checks in headless Chromium (Node 22+ and chromium)
```

Look at the page with `curl localhost:8090` to see the HTML the server writes.

## Files

| | |
|---|---|
| `app/app.go` | The UI, shared by the server and the browser. It imports only `hx`, and must not use `syscall/js`. |
| `app/enter_wasm.go`, `app/enter_native.go` | The one piece that needs the browser event, one file for each side. |
| `server/main.go` | `ssr.Render` the UI (engine `hx/ssr`), `hx.StateScript` the state, serve the page and the WASM. |
| `client/main.go` | `wasm.ReadState` the state and `wasm.Hydrate` the UI (engine `hx/wasm`). |
| `check.mjs` | The checks. |

## What `make check` verifies

- Hydrating the page the server rendered makes **no mismatches**, creates and removes **no nodes**, and sets **no attributes**. The only change to the DOM is the text that `OnMount` writes.
- The elements the server wrote are the ones in use afterwards.
- The page works: counter, add and remove with a keyed list, an input bound to a signal, a select, and a conditional that was empty on the server.
- **Nothing is lost while the WASM loads** (`/?slow=1` holds the WASM for 1.5 s, and `hx.ReplayScript()` is in the head): clicks, text typed, a select and a removal made before hydration are replayed, the signals get the text, and a click afterwards counts once.
- The page recovers when its HTML is broken on purpose (`/?break=class`, `attr`, `id`, `text`, `tag`, `count`, `empty`): the mismatch is reported in the console, the DOM ends up identical to the normal one, and the page still works.

Try a broken page by hand: <http://localhost:8090/?break=count> and look at the browser console.
