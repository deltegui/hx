//go:build js && wasm

package main

import (
	"fmt"

	"github.com/deltegui/hx"
)

func domPage() hx.INode {
	return hx.Div().Class("content").Body(
		classCard(),
		attributeCard(),
		styleCard(),
		handlerCard(),
		svgCard(),
	)
}

func classCard() hx.INode {
	info := hx.Signal("")
	box := hx.Div().Class("swatch", "hl").Text("class box")

	report := func(what string) {
		info.Set(fmt.Sprintf("%s -> HaveClass(hl) = %v", what, box.HaveClass("hl")))
	}

	return card("Classes",
		note("'remove + add' does both in the same tick: the class must stay."),
		box,
		row(
			btn("add", func() { box.Class("hl"); report("add") }),
			btn("remove", func() { box.RemoveClass("hl"); report("remove") }),
			btn("remove + add (same tick)", func() {
				box.RemoveClass("hl")
				box.Class("hl")
				report("remove+add")
			}),
		),
		live(info.Get),
	)
}

func attributeCard() hx.INode {
	target := hx.Button().Text("target button")

	return card("Attributes",
		note("'disabled' is an attribute with an empty value. It can be removed and added again with the same value."),
		row(target),
		row(
			btn("disable", func() { target.Attribute("disabled", "") }),
			btn("enable", func() { target.RemoveAttribute("disabled") }),
			btn("title A then B then A", func() {
				target.Attribute("title", "A")
				target.Attribute("title", "B")
				target.Attribute("title", "A")
			}),
		),
		note("Hover the target after the last button: its title is A."),
	)
}

func styleCard() hx.INode {
	size := hx.Signal("18")
	rotation := hx.Signal("0")
	color := hx.Signal("#3b6cf6")

	box := hx.Div().Class("swatch").Text("styled box")
	box.Effect(func() {
		box.Style("font-size", size.Get()+"px")
		box.Style("color", color.Get())
	})

	colorInput := hx.Input().Type("color").BindValue(color).BindOnInput(color)

	spinner := hx.Div().Class("swatch").Style("width", "6rem").Text("spin")
	spinner.Effect(func() {
		spinner.Style("transform", "rotate("+rotation.Get()+"deg)")
	})

	return card("Styles",
		note("Effects write styles; 'remove color' removes the style and 'restore' sets the same value again."),
		box,
		row(slider(size, "10", "40"), colorInput),
		row(
			btn("remove color", func() { box.RemoveStyle("color") }),
			btn("restore", func() { box.Style("color", color.Get()) }),
		),
		spinner,
		row(slider(rotation, "0", "360")),
	)
}

// handlerCard installs a new click handler many times over the same button.
// There is one DOM listener, so a click always counts once.
func handlerCard() hx.INode {
	clicks := hx.Signal(0)
	last := hx.Signal(0)
	generation := 0

	target := hx.Button().Text("click me")
	install := func() {
		generation++
		g := generation
		target.OnClick(func(hx.EventContext) {
			clicks.Update(func(n int) int { return n + 1 })
			last.Set(g)
		})
	}
	install()

	return card("Replacing event handlers",
		note("Replace the handler several times, then click the target: +1 per click, and the latest handler answers."),
		row(target, btn("replace handler", install)),
		row(
			live(func() string { return fmt.Sprintf("clicks: %d", clicks.Get()) }),
			live(func() string { return fmt.Sprintf("handler #%d answered", last.Get()) }),
		),
	)
}

func svgCard() hx.INode {
	hue := hx.Signal("40")
	angle := hx.Signal("0")

	path := hx.Path().
		Attribute("d", "M50 5 L61 38 L96 38 L68 59 L79 92 L50 72 L21 92 L32 59 L4 38 L39 38 Z").
		Style("transform-box", "fill-box").
		Style("transform-origin", "center")
	path.Effect(func() {
		path.Attribute("fill", fmt.Sprintf("hsl(%s 80%% 55%%)", hue.Get()))
		path.Style("transform", "rotate("+angle.Get()+"deg)")
	})

	svg := hx.Svg().Attribute("viewBox", "0 0 100 100").Attribute("width", "140").Attribute("height", "140").Body(path)

	return card("SVG",
		note("Svg and Path are created in the SVG namespace, so they render. The effect drives fill and rotation."),
		svg,
		row(slider(hue, "0", "360"), slider(angle, "0", "360")),
	)
}
