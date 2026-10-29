# The smallest form that works with and without WASM

One field: an email, to subscribe to a newsletter. About 200 lines in all. It is the same mechanism as [`../forms`](../forms), without anything else, so you can see what it takes.

```sh
make serve      # builds the WASM and starts the server on http://localhost:8097
make check      # 8 checks in headless Chromium, with JavaScript off and on (Node 22+ and chromium)
```

Try it with JavaScript turned off: it still works. Turn it on: the page does not reload.

## What the mechanism needs

Six pieces. The first three are the mechanism; the rest is plumbing.

| # | Piece | File | Lines | What it does |
|---|---|---|---|---|
| 1 | **A form that is plain HTML** | `app/app.go` | 1 | `hx.Form().Attribute("method", "post").Attribute("action", "/subscribe")` and every field with a `name`. With only this, the browser sends the form by itself. |
| 2 | **A handler that answers both** | `server/main.go` | ~25 | One endpoint. It reads the fields (`r.FormValue`), validates, and looks at the `Accept` header: JSON for the WASM, HTML for the browser (redirect on success, the same page with `422` and the error otherwise). |
| 3 | **The enhancement** | `app/app.go` | ~25 | `form.OnSubmit` calls `PreventDefault` and runs an `hx.Action` that sends `hx.FormOf(form)` with `hx.Do`. Without WASM it never runs, and piece 1 is what works. |
| 4 | **UI code both sides can build** | `app/` | | The package with the UI imports only `hx` (no `syscall/js`), so the server renders it and the browser runs it. The two things that need the browser (`PreventDefault`) go in two tiny files with build tags, `browser_wasm.go` and `browser_native.go`. |
| 5 | **The page** | `server/main.go` | ~12 | `hx.RenderToString(...)` inside `<div id="app">`, a `hx.StateScript(...)` with the state, and the loader of the WASM. |
| 6 | **The hydration** | `client/main.go` | 4 | `hx.ReadState` the state and `hx.Hydrate("app", ...)`. |

And a build step: `GOOS=js GOARCH=wasm go build -o static/main.wasm ./client` and copy `wasm_exec.js` (the `Makefile` does both).

## Reading order

1. `app/app.go`: the whole UI and the form. The comments mark the three ideas.
2. `server/main.go`: `subscribe` is the handler of piece 2; `page` is piece 5.
3. `client/main.go`: piece 6.

## What it does not have (and `../forms` does)

No CSRF token, no files, no progress, no list. A real application adds the token (a cookie and a hidden field), and then it is exactly what [`../forms`](../forms) does. The guide is [Forms and requests](../../docs/forms-and-requests.md).

## Why each piece is there

- **Why the state goes in the page (`StateScript`)?** The browser builds the UI again and must start from the same place as the server, or the page would not match. After a failed send without JavaScript the state holds what was typed and the error.
- **Why `Accept`?** It lets one URL serve both. A plain form sends `text/html`; `hx.Do` was told to send `application/json`.
- **Why validate twice?** The browser does it to answer at once, and the server does it because it cannot trust the browser. Both call the same `Validate`, so the rules cannot differ.
- **Why post, redirect, get?** After a good send the server redirects (`303`), so reloading the page does not subscribe again.
