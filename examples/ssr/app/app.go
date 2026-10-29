// Package app is the UI, shared by the server and the browser. It must not use
// syscall/js, so it builds for both.
package app

import (
	"fmt"
	"slices"
	"strings"

	"github.com/deltegui/hx"
)

// Todo is one item of the list.
type Todo struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

// State is what the server knows when it renders the page. The browser gets a
// copy of it, so both start from the same place.
type State struct {
	Count int    `json:"count"`
	Todos []Todo `json:"todos"`
}

// App builds the whole page. The server renders it with hx.RenderToString and
// the browser brings it to life with hx.Hydrate.
func App(initial State) hx.INode {
	count := hx.Signal(initial.Count)
	todos := hx.Signal(initial.Todos)
	nextID := len(initial.Todos) + 1
	draft := hx.Signal("")
	color := hx.Signal("green")
	status := hx.Signal("server rendered")
	notes := hx.Signal("Notes\nwith two lines")

	// Only runs in the browser, after hydration.
	hx.OnMount(func() { status.Set("interactive") })

	countText := hx.Computed(func() string { return fmt.Sprintf("count: %d", count.Get()) })
	isEmpty := hx.Computed(func() bool { return len(todos.Get()) == 0 })

	add := func(hx.EventContext) {
		title := strings.TrimSpace(draft.Get())
		if title == "" {
			return
		}
		id := nextID
		nextID++
		todos.Update(func(l []Todo) []Todo { return append(slices.Clone(l), Todo{ID: id, Title: title}) })
		draft.Set("")
	}

	empty := hx.Div().Id("empty")
	empty.Effect(func() {
		empty.Body(hx.If(isEmpty, hx.P().Text("Nothing to do")))
	})

	list := hx.Ul().Id("list").Body(hx.EachKeyed(todos,
		func(t Todo) int { return t.ID },
		func(id int, todo hx.Gettable[Todo], index hx.Gettable[int]) hx.INode {
			return hx.Li().Body(
				hx.Span().BindText(hx.Computed(func() string {
					return fmt.Sprintf("%d. %s ", index.Get()+1, todo.Get().Title)
				})),
				hx.Button().Class("remove").Text("x").OnClick(func(hx.EventContext) {
					todos.Update(func(l []Todo) []Todo {
						return slices.DeleteFunc(slices.Clone(l), func(t Todo) bool { return t.ID == id })
					})
				}),
			)
		}))

	return hx.Div().Id("root").Class("page").Body(
		hx.H1().Text("Hix server rendering"),
		hx.P().Id("status").BindText(status),

		hx.Section().Id("counter").Body(
			hx.P().Id("count").BindText(countText),
			hx.Button().Id("inc").Text("+1").OnClick(func(hx.EventContext) {
				count.Update(func(n int) int { return n + 1 })
			}),
		),

		hx.Section().Id("todos").Body(
			hx.Input().Id("draft").Placeholder("New todo").BindValue(draft).BindOnInput(draft).
				OnKeyUp(func(ctx hx.EventContext) {
					if isEnter(ctx) {
						add(ctx)
					}
				}),
			hx.Button().Id("add").Text("Add").OnClick(add),
			empty,
			list,
		),

		hx.Section().Id("controls").Body(
			hx.Select().Id("color").BindOnChange(color).Body(
				hx.Option().Value("red").Text("Red"),
				hx.Option().Value("green").Selected().Text("Green"),
				hx.Option().Value("blue").Text("Blue"),
			),
			hx.Span().Id("picked").BindText(hx.Computed(func() string { return "color: " + color.Get() })),
			hx.TextArea().Id("notes").BindValue(notes).BindOnInput(notes),
		),

		hx.Section().Id("misc").Body(
			hx.Table().Body(
				hx.TBody().Body(
					hx.Tr().Body(hx.Th().Text("Item"), hx.Th().Text("Qty")),
					hx.Tr().Body(hx.Td().Text("apples"), hx.Td().Text("3")),
				),
			),
			hx.Svg().Attribute("viewBox", "0 0 10 10").Attribute("width", "40").Body(
				hx.Path().Attribute("d", "M0 0 L10 0 L5 10 Z").Attribute("fill", "tomato"),
			),
			hx.Img().Attribute("alt", "pixel").Attribute("src", "data:image/gif;base64,R0lGODlhAQABAAAAACw="),
			hx.A().Href("#top").Text("a & b <link>"),
		),
	)
}
