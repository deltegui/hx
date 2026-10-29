# Forms and requests

How to send something to a server: a form, a file, some JSON. Hix gives you three levels, and you can mix them:

| Level | What | Needs WASM |
|---|---|---|
| **A plain HTML form** | `<form method="post">`, the server handles it | no |
| **`hx.Do`** | One request: method, headers, body, files, progress, timeout, cancel | yes |
| **`hx.Action` / `hx.Resource`** | The state of a request as signals: pending, error, result | yes |

The best pages use the first and the other two together: the form works without JavaScript, and when the WASM is running it sends the same fields without reloading the page. [`examples/forms`](../examples/forms) does exactly that.

## Plain forms: they work without WASM

A form in Hix is HTML. Give it `method`, `action`, and a `name` to every field, and the browser sends it by itself:

```go
form := hx.Form().
	Attribute("method", "post").
	Attribute("action", "/contacts").
	Attribute("enctype", "multipart/form-data"). // needed for files
	Body(
		hx.Input().Attribute("name", "email").BindValue(email).BindOnInput(email),
		hx.Input().Type("file").Attribute("name", "avatar"),
		hx.Button().Type("submit").Text("Save"),
	)
```

With [server rendering](server-rendering.md) this form is in the HTML the server sends, so it works before any WASM has loaded, and it keeps working if the WASM never loads.

On the server, handle it like any form. Three things make it pleasant:

1. **Post, redirect, get.** After a valid send, answer `303 See Other` to a page. Reloading does not send the form again.
2. **On an invalid send, answer `422` with the same page**, rendered with what was typed and what is wrong. Put both in the `State` you give to `RenderToString`; the UI shows them. The user does not lose anything.
3. **Check a CSRF token.** Put it in a cookie and in a hidden field. Cookies travel with every request to the same site.

The example server does all three, in one handler, for both kinds of requests (see below).

### Checkboxes and the rest of the controls

| Control | In Hix |
|---|---|
| text, textarea | `BindValue(sig)` + `BindOnInput(sig)` |
| select | `BindValue(sig)` + `BindOnChange(sig)`, `Option().Value("x").Text("X")`, `Selected()` for the first |
| checkbox, radio | `BindChecked(sig)` + `BindOnChecked(sig)`, or `Checked(bool)` |
| file | `Type("file")`, `Attribute("multiple", "")`, `Attribute("accept", ".png")`. Read it with `FilesOf` |
| hidden | `Type("hidden")` with `Attribute("value", ...)` |

On the server `BindChecked(true)` writes `checked=""`, and the checkbox is hydrated like any other field.

## Sending it with WASM: `hx.Do`

```go
response, err := hx.Do(ctx, hx.Request{
	Method: "POST",
	URL:    "/contacts",
	Header: map[string]string{"Accept": "application/json"},
	Body:   hx.FormOf(form),
})
```

`Do` makes the request and **waits** for the answer. That is why this matters:

> **Never call `Do` straight from an event handler.** The answer can only be handled after the handler returns, so waiting for it there freezes the page. Run it in a goroutine (`go func() { ... }()`), or, better, inside an `Action`, which does it for you.

`Request`:

| Field | |
|---|---|
| `Method` | `GET` if empty |
| `URL` | absolute, or relative to the page |
| `Header` | `Content-Type` is set from the body unless you give it |
| `Body` | `*hx.Body`, or `nil` |
| `Timeout` | aborts the request after this long (0 means never) |
| `Credentials` | send cookies on requests to *other* origins. Same-origin requests always send them |
| `OnProgress` | called while uploading and downloading. Runs on its own goroutine |

`Response`:

| | |
|---|---|
| `Status`, `StatusText` | |
| `Header` | names in lowercase. The browser only shows the headers that CORS allows |
| `OK()` | `true` for 2xx |
| `Text()`, `Bytes()` | the body |
| `Err()` | `nil` for 2xx, an `*hx.HTTPError` (with the status and the body) for any other status |

A status that is not 2xx is **not** an error of `Do`: you got an answer. `Do` fails (returns `err`) when there is no answer:

| Error | |
|---|---|
| `hx.ErrNotInBrowser` | `Do` was called on a server |
| `context.Canceled` (`hx.IsCanceled(err)`) | the context was canceled |
| `context.DeadlineExceeded` (`hx.IsTimeout(err)`) | the timeout passed |
| another | the network failed: `hx: network error` |

Cancel a request by canceling its context:

```go
ctx, cancel := context.WithCancel(context.Background())
go func() { _, err := hx.Do(ctx, request); ... }()
cancel() // the browser aborts it, and the server sees the connection close
```

## Bodies

| | Content-Type | |
|---|---|---|
| `hx.FormOf(form)` | `multipart/form-data` | what the `<form>` would send: every field with a `name`, **files included**. The form must already be in the page |
| `hx.NewMultipart()...Body()` | `multipart/form-data` | build it by hand (below) |
| `hx.FormValues(values)` | `application/x-www-form-urlencoded` | like a form without files. A `url.Values` works |
| `hx.TextBody(type, text)` | what you say | any text |
| `hxjson.Body(v)` | `application/json` | JSON (see [JSON](#json)) |

### Files

`FormOf` sends the files the user chose in the file inputs of the form. The browser reads them while it sends them, so a big file is not copied into memory.

To work with the files in Go:

```go
files := hx.FilesOf(fileInput) // []hx.File{Name, Size, Type}
for _, f := range files {
	fmt.Println(f.Name, f.Size, f.Type)
}
data, err := files[0].Bytes(ctx) // reads the content. Blocks: use a goroutine
```

A multipart body by hand, for when there is no form:

```go
body := hx.NewMultipart().
	Field("title", "Holiday").
	File("photo", files[0]).                                // a file the user chose
	Bytes("report", "report.csv", "text/csv", csvBytes).     // a file made in Go
	Body()
```

### Progress

```go
hx.Request{
	Body: hx.FormOf(form),
	OnProgress: func(p hx.Progress) {
		if p.Upload {
			progress.Set(p.Fraction()) // 0 to 1, or 0 if the size is unknown
		}
	},
}
```

`Progress` has `Upload`, `Loaded`, `Total` and `Fraction()`. This is why Hix uses `XMLHttpRequest` and not `fetch`: it is the only one that reports how much of the body has been uploaded.

## JSON

JSON is in its own package, `github.com/deltegui/hx/hxjson`:

```go
import "github.com/deltegui/hx/hxjson"

saved, err := hxjson.Post[Saved](ctx, "/api/contacts", contact) // sends and decodes
items, err := hxjson.Get[[]Item](ctx, "/api/items")
_, err = hxjson.Send[Saved](ctx, "PUT", "/api/contacts/3", contact)
```

`Send` returns an `*hx.HTTPError` for a status that is not 2xx. When the error answer has JSON you want to read (a 422 with the errors of each field), use `Do` and `Decode`:

```go
response, err := hx.Do(ctx, hx.Request{Method: "POST", URL: "/contacts", Body: hx.FormOf(form),
	Header: map[string]string{"Accept": "application/json"}})
if err != nil {
	return Result{}, err
}
var result Result
if err := hxjson.Decode(response, &result); err != nil { // works for a 422 too
	return Result{}, err
}
```

**Why a separate package?** `encoding/json` adds about 250 KB to a WASM file (and `net/url` another 95 KB), and Go keeps a package as soon as something imports it, even if the program never calls it. The core of Hix does not import them, so a program that does not use requests pays about 4 KB for all of server rendering, hydration and requests together. A program that sends JSON imports `hxjson` and pays for JSON, which it would anyway.

For other headers on a JSON request, build the `Request` yourself with `hxjson.Body`:

```go
body, err := hxjson.Body(payload)
response, err := hx.Do(ctx, hx.Request{Method: "DELETE", URL: url, Body: body,
	Header: map[string]string{"X-CSRF-Token": token, "Accept": "application/json"}})
```

## `Action`: the state of a request

Doing the request is half of it. The UI also wants to show "saving...", disable the button, show the error. An `Action` keeps all of that in signals:

```go
save := hx.Action(func(ctx context.Context, c Contact) (Saved, error) {
	return hxjson.Post[Saved](ctx, "/api/contacts", c)
})

save.Run(contact)

save.Pending() // Gettable[bool]
save.Error()   // Gettable[error], nil if the last run worked
save.Value()   // Gettable[Saved]: the result of the last run that worked
```

The function runs on its own goroutine, so it can block on the network. Use the three signals like any other:

```go
button := hx.Button().Type("submit")
button.Effect(func() {
	if save.Pending().Get() {
		button.Text("Saving...")
		button.Attribute("disabled", "")
	} else {
		button.Text("Save")
		button.RemoveAttribute("disabled")
	}
})
```

What an `Action` does for you:

- **Last call wins.** If you `Run` again while it is running, the previous run is canceled (its context is canceled, so `Do` aborts the request) and its answer is ignored. Nothing you started earlier can overwrite a newer result.
- **`Cancel()`** stops the run in progress. It does not count as an error.
- **`Reset()`** cancels, and forgets the error and the result.
- The error is cleared when a run starts. The last result is kept until there is a new one, so the UI can show it while it reloads.
- The result is set **before** `Pending` becomes false, so the UI never shows "finished" without the answer.

React to the end of a run, after the signals are up to date:

```go
save.OnSuccess(func(saved Saved, arg Contact) {
	name.Set("")
	banner.Set("Saved")
})
save.OnError(func(err error, arg Contact) {
	message.Set("Could not save: " + err.Error())
})
```

They run on the goroutine of the action. A canceled run calls neither.

### A form, end to end

```go
form := hx.Form().Attribute("method", "post").Attribute("action", "/contacts").
	Attribute("enctype", "multipart/form-data")

send := hx.Action(func(ctx context.Context, _ struct{}) (Result, error) {
	response, err := hx.Do(ctx, hx.Request{
		Method: "POST", URL: "/contacts",
		Header: map[string]string{"Accept": "application/json"},
		Body:   hx.FormOf(form),
	})
	if err != nil {
		return Result{}, err
	}
	var result Result
	return result, hxjson.Decode(response, &result)
})

form.OnSubmit(func(ctx hx.EventContext) {
	ctx.Event.PreventDefault() // the browser does not send it, Do does
	send.Run(struct{}{})
})
```

`PreventDefault` only exists in the browser (on a server there is no event). Put it behind a build tag, as in `examples/forms/app/browser_wasm.go`: one file with the real function, one with an empty one.

Without the WASM, the `OnSubmit` never runs and the browser sends the form by itself. The same code serves both.

**Validate in one place.** Put the validation in a function the server and the browser both call, as the example does with `Validate(values)`. The browser tells the user without a request; the server checks anyway, because it cannot trust the browser.

## `Resource`: data that comes from a server

```go
items := hx.Resource(func(ctx context.Context) ([]Item, error) {
	return hxjson.Get[[]Item](ctx, "/api/items")
})

items.Value()    // Gettable[[]Item]
items.Loading()  // Gettable[bool]
items.Error()    // Gettable[error]
items.Refetch()  // load again; the current value stays until the new one arrives
items.Set(v)     // replace the value without loading (after an action changed the data)
```

| | |
|---|---|
| `hx.Resource(fetch)` | loads when the UI is in the browser |
| `hx.ResourceWith(initial, fetch)` | starts from `initial` (usually what the server rendered) and does not load until `Refetch` |
| `hx.ResourceFor(source, fetch)` | loads when the UI is in the browser, and again each time the `source` signal changes. A load in progress is canceled by the next one |

"When the UI is in the browser" means: right away in a normal render, after the hydration when the page was rendered by a server, and never while rendering on a server. So a `Resource` can sit in code shared by both.

For the first paint to have data, give the server's data to `ResourceWith`. A resource without an initial value is empty in the HTML the server writes.

A search box:

```go
query := hx.Signal("")
results := hx.ResourceFor(query, func(ctx context.Context, q string) ([]Item, error) {
	return hxjson.Get[[]Item](ctx, "/api/search?q="+url.QueryEscape(q))
})
```

Each keystroke starts a search and cancels the previous one.

## One endpoint for both: HTML and JSON

The server decides what to answer by what the browser asks for. The WASM sends `Accept: application/json`; a plain form does not:

```go
func save(w http.ResponseWriter, r *http.Request) {
	// ... read the fields, validate (the same function the browser uses) ...
	json := strings.Contains(r.Header.Get("Accept"), "application/json")

	if len(errs) > 0 {
		if json {
			writeJSON(w, 422, Result{Errors: errs}) // the WASM shows them next to the fields
			return
		}
		renderPage(w, 422, State{Values: values, Errors: errs}) // the page, again, with the errors
		return
	}
	// ... store it ...
	if json {
		writeJSON(w, 200, Result{OK: true, Contact: &contact})
		return
	}
	http.Redirect(w, r, "/?saved=1", http.StatusSeeOther)
}
```

Reading the form is `r.ParseMultipartForm(32 << 20)` and then `r.FormValue("name")`, `r.FormFile("avatar")`, or `r.MultipartForm.File["attachments"]` for several files. An empty file input sends an empty part with no filename: skip those.

## CSRF, cookies, CORS

- **Cookies.** Requests to the origin of the page always send the cookies. Requests to another origin do not, unless you set `Credentials: true` (and the server allows it).
- **CSRF.** Keep a token in a cookie and in the page (the `State` you share is a good place). A form sends it in a hidden field and a JSON request in a header such as `X-CSRF-Token`. The server compares them with the cookie. The example does it.
- **CORS.** If the API is on another origin, the server must allow it (`Access-Control-Allow-Origin`, and `Access-Control-Allow-Credentials` if you send cookies). Otherwise the browser blocks the answer and `Do` fails with a network error. Serving the API and the page from the same origin avoids it.

## What does not work, and what to expect

- **On a server, `Do` fails with `ErrNotInBrowser`.** The types (`Request`, `Body`, `Action`...) build everywhere so code can be shared, but a request only happens in the browser.
- **No streaming.** The whole answer is read into memory, and the whole body is sent at once (the browser does stream a `File` from disk).
- **Download progress** needs the server to send `Content-Length`. Without it `Progress.Total` is 0.
- **`Header` of a response** only has what the browser lets you read.
- **Events before hydration.** A form works before the WASM loads, because it is HTML. A button that only has an `OnClick` does nothing until the hydration.

## Testing

`Action`, `Resource`, the bodies and `hxjson` are plain Go and are tested with `go test`. A request itself needs a browser: [`examples/forms/check.mjs`](../examples/forms/check.mjs) starts the server and a headless Chromium and checks the whole thing, with JavaScript off and on: forms, files with their hashes, validation, CSRF, progress, cancel, timeout and errors. Run it with `make check` (Node 22+ and Chromium).
