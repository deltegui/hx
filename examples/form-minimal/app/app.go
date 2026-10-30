// Package app is the UI. The server and the browser both use it, so it must not
// import syscall/js.
package app

import (
	"context"
	"net/mail"
	"strings"

	"github.com/deltegui/hx"
	"github.com/deltegui/hx/hxhttp"
	"github.com/deltegui/hx/hxjson"
)

// State is what the server tells the page. The browser gets a copy.
type State struct {
	Email   string `json:"email"`   // what was typed, after a send that failed
	Problem string `json:"problem"` // what is wrong with it
	Done    bool   `json:"done"`    // subscribed (after a send without JavaScript)
}

// Result is the JSON answer of POST /subscribe.
type Result struct {
	OK      bool   `json:"ok"`
	Problem string `json:"problem,omitempty"`
}

// Validate is used by the server, and by the browser to tell the user sooner.
func Validate(email string) string {
	address, err := mail.ParseAddress(strings.TrimSpace(email))
	if err != nil || address.Address != strings.TrimSpace(email) {
		return "That is not an email"
	}
	return ""
}

func App(st State) hx.INode {
	email := hx.Signal(st.Email)
	problem := hx.Signal(st.Problem)
	done := hx.Signal(st.Done)

	// A normal form. Without WASM this is all that is needed: the browser sends it.
	form := hx.Form().Attribute("method", "post").Attribute("action", "/subscribe")

	// With WASM the same fields are sent without leaving the page.
	send := hx.Action(func(ctx context.Context, _ struct{}) (Result, error) {
		response, err := hxhttp.Do(ctx, hxhttp.Request{
			Method: "POST",
			URL:    "/subscribe",
			Header: map[string]string{"Accept": "application/json"},
			Body:   hxhttp.FormOf(form),
		})
		if err != nil {
			return Result{}, err
		}
		var result Result
		return result, hxjson.Decode(response, &result)
	})
	send.OnSuccess(func(result Result, _ struct{}) {
		problem.Set(result.Problem)
		done.Set(result.OK)
		if result.OK {
			email.Set("")
		}
	})
	send.OnError(func(err error, _ struct{}) { problem.Set("Could not send it: " + err.Error()) })

	form.OnSubmit(func(ctx hx.EventContext) {
		preventDefault(ctx) // the browser does not send it, hxhttp.Do does
		if p := Validate(email.Get()); p != "" {
			problem.Set(p)
			return
		}
		problem.Set("")
		send.Run(struct{}{})
	})

	button := hx.Button().Id("subscribe").Type("submit")
	button.Effect(func() {
		if send.Pending().Get() {
			button.Text("Sending...")
			button.Attribute("disabled", "")
		} else {
			button.Text("Subscribe")
			button.RemoveAttribute("disabled")
		}
	})

	thanks := hx.P().Id("thanks")
	thanks.Effect(func() { thanks.Body(hx.If(done, hx.Span().Text("Thanks, you are subscribed!"))) })

	return hx.Div().Body(
		hx.H1().Text("Newsletter"),
		form.Body(
			hx.Label().Body(hx.Span().Text("Email "),
				hx.Input().Id("email").Attribute("name", "email").BindValue(email).BindOnInput(email)),
			button,
		),
		hx.P().Id("problem").BindText(problem),
		thanks,
	)
}
