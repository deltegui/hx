//go:build js && wasm

package main

import (
	"strings"
	"syscall/js"

	"github.com/deltegui/hx"
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
	root := hx.NewFromId("app")

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
			hx.Header().Body(
				hx.H1().Text("Hix demo"),
				hx.P().Text("Signals, effects, roots, node-owned bindings, lists, DOM details, forms and in-browser checks."),
			),
			nav,
			content,
		)
		return struct{}{}
	})

	select {} // keep the WASM program alive
}
