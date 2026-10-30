//go:build js && wasm

package main

import (
	"strings"
	"syscall/js"

	"github.com/deltegui/hx"
	"github.com/deltegui/hx/wasm"
)

type page struct {
	id, title string
	build     func() hx.INode
}

func pages() []page {
	return []page{
		{"reactivity", "Reactivity", reactivityPage},
		{"lifecycle", "Lifecycle", lifecyclePage},
		{"lists", "Lists", listsPage},
		{"dom", "DOM details", domPage},
		{"forms", "Forms", formsPage},
		{"stress", "Stress", stressPage},
		{"checks", "Checks", checksPage},
	}
}

func hashPage() (id string, auto bool) {
	hash := strings.TrimPrefix(js.Global().Get("location").Get("hash").String(), "#")
	if strings.HasSuffix(hash, "-auto") {
		return strings.TrimSuffix(hash, "-auto"), true
	}
	return hash, false
}

func main() {
	root := wasm.NewFromId("app")

	hx.CreateRoot(func(dispose func()) struct{} {
		all := pages()
		start, auto := hashPage()
		if start == "" {
			start = all[0].id
		}
		autoRunChecks = auto
		current := hx.Signal(start)

		nav := hx.Nav().Class("tabs")
		nav.Effect(func() {
			active := current.Get()
			buttons := make([]hx.INode, 0, len(all))
			for _, p := range all {
				b := hx.Button().Text(p.title)
				if p.id == active {
					b.Class("active")
				}
				b.OnClick(func(hx.EventContext) {
					current.Set(p.id)
					js.Global().Get("location").Set("hash", p.id)
				})
				buttons = append(buttons, b)
			}
			nav.BodyList(buttons)
		})

		// The whole page is rebuilt when the tab changes. The effects and
		// bindings of the old page are released automatically.
		content := hx.Main().Class("content")
		content.Effect(func() {
			active := current.Get()
			for _, p := range all {
				if p.id == active {
					content.Body(p.build())
					return
				}
			}
		})

		root.Body(
			hx.Header().Class("hero").Body(
				hx.Img().Src("logo.svg").Attribute("alt", "Hix logo").Class("logo"),
				hx.Div().Body(
					hx.H1().Text("Hix"),
					hx.P().Text("A small, reactive frontend library for Go (WASM). It aims to be small, simple, and predictable."),
					hx.P().Class("links").Body(
						hx.A().Href("https://github.com/deltegui/hx").Text("GitHub"),
						hx.Span().Text(" · "),
						hx.A().Href("https://github.com/deltegui/hx/tree/main/examples/demo").Text("Source of this demo"),
					),
					hx.P().Class("note").Text("Signals, effects, roots, node-owned bindings, lists, DOM details, forms and in-browser checks."),
				),
			),
			nav,
			content,
		)
		return struct{}{}
	})

	select {} // keep the WASM program alive
}
