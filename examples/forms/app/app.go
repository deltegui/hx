package app

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/deltegui/hx"
	"github.com/deltegui/hx/hxjson"
)

// App builds the whole page. The server renders it with hx.RenderToString and
// the browser brings it to life with hx.Hydrate.
//
// The forms work without JavaScript: they are normal HTML forms that the server
// understands. When the WASM is running they send the same fields with
// hx.Do instead, and the page does not reload.
func App(st State) hx.INode {
	// The contacts are a resource: they start with what the server rendered and
	// "Refresh" loads them again.
	contacts := hx.ResourceWith(st.Contacts, func(ctx context.Context) ([]Contact, error) {
		return hxjson.Get[[]Contact](ctx, "/api/contacts")
	})
	status := hx.Signal("server rendered")
	banner := hx.Signal("")
	if st.Saved > 0 {
		banner.Set(fmt.Sprintf("Saved contact #%d", st.Saved))
	}

	hx.OnMount(func() { status.Set("interactive") })

	return hx.Div().Id("root").Body(
		hx.H1().Text("Hix forms"),
		hx.P().Id("status").BindText(status),
		hx.P().Id("banner").Class("banner").BindText(banner),
		contactForm(st, contacts, banner),
		contactList(st, contacts),
		lab(),
	)
}

func countryName(code string) string {
	for _, c := range Countries {
		if c.Code == code {
			return c.Name
		}
	}
	return code
}

func contactForm(st State, contacts *hx.ResourceT[[]Contact], banner *hx.SignalT[string]) hx.INode {
	selected := st.Values.Country
	if selected == "" {
		selected = DefaultCountry
	}
	name := hx.Signal(st.Values.Name)
	email := hx.Signal(st.Values.Email)
	message := hx.Signal(st.Values.Message)
	country := hx.Signal(selected)
	newsletter := hx.Signal(st.Values.Newsletter)
	errs := hx.Signal(st.Errors)
	progress := hx.Signal(0.0)
	failure := hx.Signal("")

	// A normal form: this is all that works without JavaScript.
	form := hx.Form().Id("contact-form").
		Attribute("method", "post").
		Attribute("action", "/contacts").
		Attribute("enctype", "multipart/form-data").
		Attribute("novalidate", "")

	send := hx.Action(func(ctx context.Context, _ struct{}) (Result, error) {
		progress.Set(0)
		response, err := hx.Do(ctx, hx.Request{
			Method: "POST",
			URL:    "/contacts",
			Header: map[string]string{"Accept": "application/json"},
			Body:   hx.FormOf(form), // every field of the form, files included
			OnProgress: func(p hx.Progress) {
				if p.Upload {
					progress.Set(p.Fraction())
				}
			},
		})
		if err != nil {
			return Result{}, err
		}
		// A 422 carries the errors of each field as JSON: that is an answer, not a failure.
		var result Result
		if err := hxjson.Decode(response, &result); err != nil {
			if httpErr := response.Err(); httpErr != nil {
				return Result{}, httpErr
			}
			return Result{}, err
		}
		return result, nil
	})
	send.OnSuccess(func(result Result, _ struct{}) {
		errs.Set(result.Errors)
		if !result.OK {
			return
		}
		contacts.Set(append([]Contact{*result.Contact}, hx.UntrackGet(contacts.Value())...))
		resetForm(form)
		name.Set("")
		email.Set("")
		message.Set("")
		country.Set(DefaultCountry)
		newsletter.Set(false)
		banner.Set(fmt.Sprintf("Saved contact #%d", result.Contact.ID))
	})
	send.OnError(func(err error, _ struct{}) {
		failure.Set("Could not save: " + err.Error())
	})

	form.OnSubmit(func(ctx hx.EventContext) {
		preventDefault(ctx) // the browser does not send the form, hx.Do does
		banner.Set("")
		failure.Set("")
		values := Values{
			Name: name.Get(), Email: email.Get(), Message: message.Get(),
			Country: country.Get(), Newsletter: newsletter.Get(),
		}
		if problems := Validate(values); len(problems) > 0 {
			errs.Set(problems) // the same rules as the server, without a round trip
			return
		}
		errs.Set(nil)
		send.Run(struct{}{})
	})

	fieldError := func(key string) hx.INode {
		return hx.P().Class("error").Id("error-" + key).
			BindText(hx.Computed(func() string { return errs.Get()[key] }))
	}

	options := make([]hx.INode, 0, len(Countries))
	for _, c := range Countries {
		option := hx.Option().Value(c.Code).Text(c.Name)
		if c.Code == selected {
			option.Selected()
		}
		options = append(options, option)
	}

	bar := hx.El("progress").Id("progress").Attribute("max", "1")
	bar.Effect(func() { bar.Attribute("value", fmt.Sprintf("%.2f", progress.Get())) })

	submit := hx.Button().Id("submit").Type("submit")
	submit.Effect(func() {
		if send.Pending().Get() {
			submit.Text("Saving...")
			submit.Attribute("disabled", "")
		} else {
			submit.Text("Save contact")
			submit.RemoveAttribute("disabled")
		}
	})

	form.Body(
		hx.Input().Type("hidden").Attribute("name", "csrf").Attribute("value", st.CSRF),

		hx.Label().Body(hx.Span().Text("Name"),
			hx.Input().Id("name").Attribute("name", "name").BindValue(name).BindOnInput(name)),
		fieldError("name"),

		hx.Label().Body(hx.Span().Text("Email"),
			hx.Input().Id("email").Attribute("name", "email").BindValue(email).BindOnInput(email)),
		fieldError("email"),

		hx.Label().Body(hx.Span().Text("Country"),
			hx.Select().Id("country").Attribute("name", "country").BindValue(country).BindOnChange(country).Body(options...)),
		fieldError("country"),

		hx.Label().Body(hx.Span().Text("Message"),
			hx.TextArea().Id("message").Attribute("name", "message").BindValue(message).BindOnInput(message)),
		fieldError("message"),

		hx.Label().Body(
			hx.Input().Type("checkbox").Id("newsletter").Attribute("name", "newsletter").Attribute("value", "yes").
				BindChecked(newsletter).BindOnChecked(newsletter),
			hx.Span().Text(" Send me the newsletter")),

		hx.Label().Body(hx.Span().Text("Avatar"),
			hx.Input().Type("file").Id("avatar").Attribute("name", "avatar").Attribute("accept", ".png,.jpg,.txt")),
		hx.Label().Body(hx.Span().Text("Attachments"),
			hx.Input().Type("file").Id("attachments").Attribute("name", "attachments").Attribute("multiple", "")),

		bar,
		submit,
		hx.P().Class("error").Id("failure").BindText(failure),
	)

	return hx.Section().Id("new").Body(hx.H2().Text("New contact"), form)
}

func fileSummary(c Contact) string {
	var parts []string
	if c.Avatar != nil {
		parts = append(parts, fmt.Sprintf("avatar %s (%d bytes, %s)", c.Avatar.Name, c.Avatar.Size, c.Avatar.Hash))
	}
	if len(c.Attachments) > 0 {
		names := make([]string, len(c.Attachments))
		for i, f := range c.Attachments {
			names[i] = fmt.Sprintf("%s (%d bytes, %s)", f.Name, f.Size, f.Hash)
		}
		parts = append(parts, fmt.Sprintf("%d attachments: %s", len(c.Attachments), strings.Join(names, ", ")))
	}
	if len(parts) == 0 {
		return "no files"
	}
	return strings.Join(parts, "; ")
}

func contactList(st State, contacts *hx.ResourceT[[]Contact]) hx.INode {
	remove := hx.Action(func(ctx context.Context, id int) (int, error) {
		response, err := hx.Do(ctx, hx.Request{
			Method: "DELETE",
			URL:    fmt.Sprintf("/api/contacts/%d", id),
			Header: map[string]string{"Accept": "application/json", "X-CSRF-Token": st.CSRF},
		})
		if err != nil {
			return id, err
		}
		return id, response.Err()
	})
	remove.OnSuccess(func(id int, _ int) {
		contacts.Set(slices.DeleteFunc(slices.Clone(hx.UntrackGet(contacts.Value())), func(c Contact) bool { return c.ID == id }))
	})
	removeFailure := hx.Signal("")
	remove.OnError(func(err error, _ int) { removeFailure.Set("Could not delete: " + err.Error()) })

	isEmpty := hx.Computed(func() bool { return len(contacts.Value().Get()) == 0 })
	empty := hx.Div().Id("no-contacts")
	empty.Effect(func() { empty.Body(hx.If(isEmpty, hx.P().Text("No contacts yet"))) })

	list := hx.Ul().Id("contacts").Body(hx.EachKeyed(contacts.Value(),
		func(c Contact) int { return c.ID },
		func(id int, contact hx.Gettable[Contact], _ hx.Gettable[int]) hx.INode {
			summary := hx.Computed(func() string {
				c := contact.Get()
				return fmt.Sprintf("%s <%s> from %s", c.Name, c.Email, countryName(c.Country))
			})
			files := hx.Computed(func() string { return fileSummary(contact.Get()) })

			// The delete button is a form too: it works without JavaScript.
			deleteForm := hx.Form().Class("delete").
				Attribute("method", "post").
				Attribute("action", fmt.Sprintf("/contacts/%d/delete", id)).
				OnSubmit(func(ctx hx.EventContext) {
					preventDefault(ctx)
					removeFailure.Set("")
					remove.Run(id)
				}).
				Body(
					hx.Input().Type("hidden").Attribute("name", "csrf").Attribute("value", st.CSRF),
					hx.Button().Type("submit").Class("delete-button").Text("Delete"),
				)

			return hx.Li().Class("contact").Id(fmt.Sprintf("contact-%d", id)).Body(
				hx.Strong().BindText(summary),
				hx.P().Class("files").BindText(files),
				deleteForm,
			)
		}))

	refresh := hx.Button().Id("refresh").Type("button")
	refresh.Effect(func() {
		if contacts.Loading().Get() {
			refresh.Text("Refreshing...")
		} else {
			refresh.Text("Refresh from the server")
		}
	})
	refresh.OnClick(func(hx.EventContext) { contacts.Refetch() })

	return hx.Section().Id("list").Body(
		hx.H2().Text("Contacts"),
		refresh,
		hx.P().Class("error").Id("list-error").BindText(hx.Computed(func() string {
			if err := contacts.Error().Get(); err != nil {
				return "Could not refresh: " + err.Error()
			}
			return removeFailure.Get()
		})),
		empty,
		list,
	)
}

// lab shows the requests one by one: they are what an Action can run, and the
// ways in which a request can end.
func lab() hx.INode {
	uploaded := hx.Signal(0.0)

	run := hx.Action(func(ctx context.Context, which string) (string, error) {
		started := time.Now()
		switch which {
		case "slow":
			response, err := hx.Do(ctx, hx.Request{URL: "/api/slow?ms=3000"})
			if err != nil {
				return "", err
			}
			return "slow answer: " + response.Text(), response.Err()

		case "timeout":
			_, err := hx.Do(ctx, hx.Request{URL: "/api/slow?ms=1500", Timeout: 100 * time.Millisecond})
			return "", err

		case "missing":
			response, err := hx.Do(ctx, hx.Request{URL: "/api/missing"})
			if err != nil {
				return "", err
			}
			return "", response.Err()

		case "network":
			_, err := hx.Do(ctx, hx.Request{URL: "http://127.0.0.1:1/nothing-listens-here"})
			return "", err

		case "upload":
			uploaded.Set(0)
			data := make([]byte, 3<<20)
			for i := range data {
				data[i] = byte(i)
			}
			body := hx.NewMultipart().Field("note", "three megabytes").Bytes("blob", "big.bin", "application/octet-stream", data).Body()
			response, err := hx.Do(ctx, hx.Request{
				Method: "POST", URL: "/api/upload", Body: body,
				OnProgress: func(p hx.Progress) {
					if p.Upload {
						uploaded.Set(p.Fraction())
					}
				},
			})
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("uploaded, the server read: %s (%v)", strings.TrimSpace(response.Text()), time.Since(started).Round(time.Millisecond)), response.Err()
		}
		return "", fmt.Errorf("unknown request %q", which)
	})

	result := hx.Computed(func() string {
		switch err := run.Error().Get(); {
		case run.Pending().Get():
			return "running"
		case err == nil:
			if value := run.Value().Get(); value != "" {
				return value
			}
			return "idle"
		case hx.IsCanceled(err):
			return "canceled"
		case hx.IsTimeout(err):
			return "timeout"
		default:
			return "error: " + err.Error()
		}
	})

	button := func(id, label, which string) hx.INode {
		return hx.Button().Id(id).Type("button").Text(label).OnClick(func(hx.EventContext) { run.Run(which) })
	}

	return hx.Section().Id("lab").Body(
		hx.H2().Text("Requests"),
		hx.Div().Class("row").Body(
			button("lab-slow", "Slow (3 s)", "slow"),
			hx.Button().Id("lab-cancel").Type("button").Text("Cancel").OnClick(func(hx.EventContext) { run.Cancel() }),
			button("lab-timeout", "Timeout (100 ms)", "timeout"),
			button("lab-missing", "Missing (404)", "missing"),
			button("lab-network", "Network error", "network"),
			button("lab-upload", "Upload 3 MB", "upload"),
		),
		hx.P().Id("lab-result").BindText(result),
		hx.P().Id("lab-progress").BindText(hx.Computed(func() string { return fmt.Sprintf("upload: %.0f%%", uploaded.Get()*100) })),
	)
}
