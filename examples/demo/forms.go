//go:build js && wasm

package main

import (
	"fmt"
	"strings"

	"github.com/deltegui/hx"
	"github.com/deltegui/hx/wasm"
	"honnef.co/go/js/dom/v2"
)

func formsPage() hx.INode {
	return hx.Div().Class("content").Body(
		mirrorCard(),
		textareaCard(),
		choicesCard(),
		keysCard(),
	)
}

func mirrorCard() hx.INode {
	text := hx.Signal("")

	first := hx.Input().Placeholder("type here").BindValue(text).BindOnInput(text)
	second := hx.Input().Placeholder("I mirror the first one").BindValue(text).BindOnInput(text)

	return card("Two inputs, one signal",
		note("Both inputs use BindValue and BindOnInput. 'clear' resets them from the signal."),
		row(first, second),
		row(
			btn("clear", func() { text.Set("") }),
			btn("shout", func() { text.Set(strings.ToUpper(text.Get())) }),
			live(func() string { return fmt.Sprintf("%d characters", len(text.Get())) }),
		),
	)
}

func textareaCard() hx.INode {
	text := hx.Signal("Edit me.\nBindValue keeps the textarea and the signal in sync.")

	area := hx.TextArea().BindValue(text).BindOnInput(text)

	return card("Textarea",
		area,
		row(
			live(func() string { return fmt.Sprintf("%d lines", strings.Count(text.Get(), "\n")+1) }),
			btn("append line", func() { text.Set(text.Get() + "\nnew line") }),
		),
	)
}

func choicesCard() hx.INode {
	color := hx.Signal("green")
	agree := hx.Signal(false)

	sel := hx.Select().BindOnChange(color).Body(
		hx.Option().Value("red").Text("Red"),
		hx.Option().Value("green").Selected().Text("Green"),
		hx.Option().Value("blue").Text("Blue"),
	)

	check := hx.Input().Type("checkbox").OnChange(func(ctx hx.EventContext) {
		agree.Set(wasm.EventOf(ctx).Target().Underlying().Get("checked").Bool())
	})

	return card("Select and checkbox",
		note("Plain On(EventChange) handlers read the value from the event target."),
		row(sel, live(func() string { return "color = " + color.Get() })),
		row(hx.Label().Body(check, hx.Span().Text(" I agree")), live(func() string { return fmt.Sprintf("agree = %v", agree.Get()) })),
	)
}

func keysCard() hx.INode {
	log := newLog()
	in := hx.Input().Placeholder("press keys").OnKeyUp(func(ctx hx.EventContext) {
		if key, ok := ctx.Event.(*dom.KeyboardEvent); ok {
			log.Add("keyup %q", key.Key())
		}
	})

	return card("Keyboard events",
		in,
		log.View(),
	)
}
