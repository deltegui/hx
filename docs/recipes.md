# Recipes

Complete examples you can copy. Each function returns a node, so mount it with `hx.NewFromId("app").Body(recipe())`. All of them are checked to compile against the current API.

## Counter

```go
func counter() hx.INode {
	count := hx.Signal(0)
	text := hx.Computed(func() string { return fmt.Sprintf("Count: %d", count.Get()) })

	return hx.Div().Body(
		hx.P().BindText(text),
		hx.Button().Text("-").OnClick(func(hx.EventContext) {
			count.Update(func(n int) int { return n - 1 })
		}),
		hx.Button().Text("+").OnClick(func(hx.EventContext) {
			count.Update(func(n int) int { return n + 1 })
		}),
	)
}
```

## Todo list

Input, add with a button or Enter, remove, and a message when empty.

```go
func todoList() hx.INode {
	todos := hx.Signal([]string{})
	draft := hx.Signal("")
	isEmpty := hx.Computed(func() bool { return len(todos.Get()) == 0 })

	add := func(hx.EventContext) {
		text := draft.Get()
		if text == "" {
			return
		}
		todos.Update(func(l []string) []string { return append(slices.Clone(l), text) })
		draft.Set("")
	}

	list := hx.Ul()
	list.Effect(func() {
		list.Body(hx.Each(todos, func(i int, todo string) hx.INode {
			return hx.Li().Body(
				hx.Span().Text(todo+" "),
				hx.Button().Text("x").OnClick(func(hx.EventContext) {
					todos.Update(func(l []string) []string {
						return slices.Delete(slices.Clone(l), i, i+1)
					})
				}),
			)
		}))
	})

	empty := hx.Div()
	empty.Effect(func() {
		empty.Body(hx.If(isEmpty, hx.P().Text("Nothing to do!")))
	})

	return hx.Div().Body(
		hx.Input().Placeholder("New todo").BindValue(draft).BindOnInput(draft).
			OnKeyUp(func(ctx hx.EventContext) {
				if key, ok := ctx.Event.(*dom.KeyboardEvent); ok && key.Key() == "Enter" {
					add(ctx)
				}
			}),
		hx.Button().Text("Add").OnClick(add),
		empty,
		list,
	)
}
```

`dom` is `honnef.co/go/js/dom/v2`.

## Keyed list: rows that keep their state

Every row has an input for notes. Moving, adding or removing rows does not lose what you typed, because each row is built once per ID.

```go
type Task struct {
	ID    int
	Title string
}

func tasks() hx.INode {
	items := hx.Signal([]Task{{1, "write docs"}, {2, "ship it"}, {3, "relax"}})
	nextID := 3

	move := func(id, by int) {
		items.Update(func(l []Task) []Task {
			l = slices.Clone(l)
			i := slices.IndexFunc(l, func(t Task) bool { return t.ID == id })
			j := i + by
			if i < 0 || j < 0 || j >= len(l) {
				return l
			}
			l[i], l[j] = l[j], l[i]
			return l
		})
	}

	list := hx.Ul().Body(hx.EachKeyed(items,
		func(t Task) int { return t.ID },
		func(id int, task hx.Gettable[Task], index hx.Gettable[int]) hx.INode {
			title := hx.Computed(func() string {
				return fmt.Sprintf("%d. %s ", index.Get()+1, task.Get().Title)
			})
			return hx.Li().Body(
				hx.Span().BindText(title),
				hx.Input().Placeholder("notes"),
				hx.Button().Text("up").OnClick(func(hx.EventContext) { move(id, -1) }),
				hx.Button().Text("down").OnClick(func(hx.EventContext) { move(id, +1) }),
				hx.Button().Text("x").OnClick(func(hx.EventContext) {
					items.Update(func(l []Task) []Task {
						return slices.DeleteFunc(slices.Clone(l), func(t Task) bool { return t.ID == id })
					})
				}),
			)
		}))

	return hx.Div().Body(
		list,
		hx.Button().Text("Add task").OnClick(func(hx.EventContext) {
			nextID++
			items.Update(func(l []Task) []Task {
				return append(slices.Clone(l), Task{nextID, fmt.Sprintf("task %d", nextID)})
			})
		}),
	)
}
```

## Live search filter

A text box that filters a list. The filtered list is a `Computed`-like value kept in a signal updated by an effect.

```go
func search() hx.INode {
	all := []string{"apple", "banana", "cherry", "grape", "lemon"}
	query := hx.Signal("")

	list := hx.Ul()
	list.Effect(func() {
		q := strings.ToLower(query.Get())
		var nodes []hx.INode
		for _, item := range all {
			if strings.Contains(item, q) {
				nodes = append(nodes, hx.Li().Text(item))
			}
		}
		list.BodyList(nodes)
	})

	return hx.Div().Body(
		hx.Input().Placeholder("Search").BindValue(query).BindOnInput(query),
		list,
	)
}
```

## Tabs

The page is rebuilt when the tab changes. Everything the old tab created is released.

```go
func tabs() hx.INode {
	pages := map[string]func() hx.INode{
		"one": func() hx.INode { return hx.P().Text("First page") },
		"two": func() hx.INode { return counter() },
	}
	names := []string{"one", "two"}
	current := hx.Signal("one")

	bar := hx.Nav()
	bar.Effect(func() {
		active := current.Get()
		buttons := make([]hx.INode, 0, len(names))
		for _, name := range names {
			buttons = append(buttons,
				hx.Button().Text(name).ClassIf("active", name == active).
					OnClick(func(hx.EventContext) { current.Set(name) }))
		}
		bar.BodyList(buttons)
	})

	content := hx.Main()
	content.Effect(func() {
		content.Body(pages[current.Get()]())
	})

	return hx.Div().Body(bar, content)
}
```

## Show and hide with a lazy branch

```go
func details() hx.INode {
	open := hx.Signal(false)

	box := hx.Div()
	box.Effect(func() {
		box.Body(hx.IfFn(open, func() hx.INode {
			return hx.P().Text("This paragraph only exists while it is open.")
		}))
	})

	return hx.Div().Body(
		hx.Button().Text("Toggle").OnClick(func(hx.EventContext) {
			open.Update(func(b bool) bool { return !b })
		}),
		box,
	)
}
```

## Timer that cleans up after itself

`OnCleanup` stops the interval whenever the effect re-runs or the node leaves the page.

```go
func timer() hx.INode {
	running := hx.Signal(false)
	seconds := hx.Signal(0)

	holder := hx.Div()
	holder.Effect(func() {
		if !running.Get() {
			return
		}
		tick := js.FuncOf(func(js.Value, []js.Value) any {
			seconds.Update(func(n int) int { return n + 1 })
			return nil
		})
		id := js.Global().Call("setInterval", tick, 1000)
		hx.OnCleanup(func() {
			js.Global().Call("clearInterval", id)
			tick.Release()
		})
	})

	return hx.Div().Body(
		hx.Span().BindText(hx.Computed(func() string { return fmt.Sprintf("%d s", seconds.Get()) })),
		hx.Button().Text("Start / stop").OnClick(func(hx.EventContext) {
			running.Update(func(b bool) bool { return !b })
		}),
		holder,
	)
}
```

## Form with validation

```go
func signup() hx.INode {
	email := hx.Signal("")
	valid := hx.Computed(func() bool { return strings.Contains(email.Get(), "@") })

	submit := hx.Button().Text("Sign up")
	submit.Effect(func() {
		if valid.Get() {
			submit.RemoveAttribute("disabled")
		} else {
			submit.Attribute("disabled", "")
		}
	})

	message := hx.P()
	message.Effect(func() {
		message.Text(map[bool]string{true: "Looks good", false: "Type a valid email"}[valid.Get()])
	})

	return hx.Form().OnSubmit(func(ctx hx.EventContext) {
		ctx.Event.PreventDefault()
		fmt.Println("signing up", email.Get())
	}).Body(
		hx.Input().Type("email").Placeholder("you@example.com").BindValue(email).BindOnInput(email),
		submit,
		message,
	)
}
```

## A reusable component with props

Components are functions. Pass signals when the parent owns the state.

```go
func field(label string, value *hx.SignalT[string]) hx.INode {
	return hx.Label().Body(
		hx.Span().Text(label+" "),
		hx.Input().BindValue(value).BindOnInput(value),
	)
}

func profile() hx.INode {
	first, last := hx.Signal("Ada"), hx.Signal("Lovelace")
	full := hx.Computed(func() string { return first.Get() + " " + last.Get() })

	return hx.Div().Body(
		field("First", first),
		field("Last", last),
		hx.H2().BindText(full),
	)
}
```

## Everything in a root

```go
func main() {
	hx.CreateRoot(func(dispose func()) struct{} {
		hx.NewFromId("app").Body(todoList(), tasks(), timer(), tabs())
		return struct{}{}
	})
	select {}
}
```

## An SVG you can control

```go
func star() hx.INode {
	angle := hx.Signal("0")

	path := hx.Path().
		Attribute("d", "M50 5 L61 38 L96 38 L68 59 L79 92 L50 72 L21 92 L32 59 L4 38 L39 38 Z").
		Attribute("fill", "gold").
		Style("transform-box", "fill-box").
		Style("transform-origin", "center")
	path.Effect(func() {
		path.Style("transform", "rotate("+angle.Get()+"deg)")
	})

	return hx.Div().Body(
		hx.Svg().Attribute("viewBox", "0 0 100 100").Attribute("width", "120").Body(path),
		hx.Input().Type("range").Attribute("min", "0").Attribute("max", "360").
			BindValue(angle).BindOnInput(angle),
	)
}
```
