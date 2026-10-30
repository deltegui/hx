# Forms that work with and without WASM

A contact form with text fields, a select, a checkbox, an avatar and several attachments. A Go server renders the page and handles the form. It works in three situations:

| | What sends the form | The answer |
|---|---|---|
| **No JavaScript** (or before the WASM has loaded) | the browser, as any HTML form | HTML: a redirect after a good send, or the same page with `422`, the errors and what was typed |
| **WASM running** | `hxhttp.Do` with `hxhttp.FormOf(form)` | JSON, and the page does not reload |
| **Buttons, lists, requests** | `hx.Action`, `hx.Resource` | JSON |

It is the same endpoint (`POST /contacts`): it looks at the `Accept` header. And it is the same UI code: `app/app.go` is shared by the server and the browser.

```sh
make serve      # builds the WASM and starts the server on http://localhost:8095
make check      # 47 checks in headless Chromium, with JavaScript off and on (Node 22+ and chromium)
```

Try it with JavaScript turned off in the browser: the form still works, files included.

## Files

| | |
|---|---|
| `app/app.go` | The UI: the form, the list of contacts, and a "Requests" section. Shared. |
| `app/model.go` | The types and `Validate`, used by the server and by the browser. |
| `app/browser_wasm.go`, `app/browser_native.go` | The two things that need the browser (`PreventDefault`, `form.reset()`), one file for each side. |
| `server/main.go` | `ssr.Render`, the form handler (HTML or JSON, CSRF, uploads), and the endpoints of the lab. |
| `client/main.go` | `hx.Hydrate`. |
| `check.mjs` | The checks. |

## What it shows

- **Progressive enhancement.** `form.OnSubmit` calls `PreventDefault` and runs an `Action`; without WASM that handler never runs and the browser sends the form.
- **Files.** `hxhttp.FormOf(form)` sends them; the server reports each one's size and hash, and the checks compare them with the real ones. A progress bar follows the upload (`OnProgress`).
- **Validation in one place.** `Validate` runs in the browser (no request) and on the server (it cannot trust the browser). The server also has a rule only it knows (a `.invalid` email domain), and its `422` JSON is shown next to the field.
- **CSRF.** A token in a cookie and in the page: a hidden field for forms, a header for JSON.
- **Keyed list.** Saving adds a contact at the top without recreating the others; deleting removes one.
- **`Resource`.** The list starts with what the server rendered (`ResourceWith`) and "Refresh" loads it again.
- **Requests.** The "Requests" section starts a slow request and cancels it (the server sees the connection close), a timeout, a 404, a network error, and a 3 MB upload with its progress.
- **Hydration of a page that came back from a failed send**, with the typed values, the selected country and the checkbox.

## What `make check` verifies

Without JavaScript (scripts are turned off in the browser): the form has what a browser needs, an invalid send answers `422` with the page and the values, a valid one redirects, files arrive with the right size and hash, and the delete button (a form too) works.

With JavaScript: hydrating changes nothing in the DOM but one text, the form is validated in the browser without a request, the server's `422` shows its errors, a send does not reload the page and uploads the files, the form is emptied (files included), the keyed list keeps the first element, delete and refresh work, the server saw JSON requests only, and the requests of the lab end as they should.

And the server refuses a form without the right CSRF token.
