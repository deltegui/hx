//go:build js && wasm

package main

import (
	"fmt"
	"maps"
	"math/rand"
	"slices"
	"strings"

	"github.com/deltegui/hx"
)

func listsPage() hx.INode {
	return hx.Div().Class("content").Body(
		keyedCard(),
		sandwichCard(),
		reorderCard(),
		mapsCard(),
		textChildrenCard(),
	)
}

// sandwichCard changes a Noop that sits between two fixed siblings. The new
// items must appear between them, not at the end.
func sandwichCard() hx.INode {
	items := hx.Signal([]string{"b", "c"})
	next := 0

	mid := hx.Noop()
	mid.Effect(func() {
		nodes := []hx.INode{}
		for _, item := range items.Get() {
			nodes = append(nodes, hx.Li().Text(item))
		}
		mid.BodyList(nodes)
	})

	push := func(front bool) {
		next++
		name := fmt.Sprintf("n%d", next)
		items.Update(func(l []string) []string {
			if front {
				return append([]string{name}, l...)
			}
			return append(slices.Clone(l), name)
		})
	}

	return card("Noop between siblings",
		note("Only the middle Noop changes. Its items must stay between the two grey edges."),
		hx.Ul().Class("sandwich").Body(
			hx.Li().Class("edge").Text("first (fixed)"),
			mid,
			hx.Li().Class("edge").Text("last (fixed)"),
		),
		row(
			btn("add at start", func() { push(true) }),
			btn("add at end", func() { push(false) }),
			btn("remove first", func() {
				items.Update(func(l []string) []string {
					if len(l) == 0 {
						return l
					}
					return slices.Clone(l[1:])
				})
			}),
			btn("clear", func() { items.Set([]string{}) }),
		),
	)
}

// reorderCard passes the same nodes to Body again in another order. Nodes are
// reused (no duplicates) and keep their own click handlers and counters.
func reorderCard() hx.INode {
	nodes := []hx.INode{}
	for _, name := range []string{"A", "B", "C", "D", "E"} {
		count := hx.Signal(0)
		chip := hx.Button().Class("chip")
		chip.BindText(hx.Computed(func() string { return fmt.Sprintf("%s: %d", name, count.Get()) }))
		chip.OnClick(func(hx.EventContext) { count.Update(func(n int) int { return n + 1 }) })
		nodes = append(nodes, chip)
	}
	holder := hx.Div().Class("row")
	holder.BodyList(nodes)

	return card("Reusing nodes in Body",
		note("Click some chips to count, then shuffle. The same nodes are moved, so counters and handlers survive and each click counts once."),
		holder,
		row(
			btn("shuffle", func() {
				rand.Shuffle(len(nodes), func(i, j int) { nodes[i], nodes[j] = nodes[j], nodes[i] })
				holder.BodyList(nodes)
			}),
			btn("reverse", func() {
				slices.Reverse(nodes)
				holder.BodyList(nodes)
			}),
		),
	)
}

func mapsCard() hx.INode {
	scores := hx.Signal(map[string]int{"carol": 3, "alice": 1, "bob": 2})
	extra := 0

	unsorted := hx.Ul()
	unsorted.Effect(func() {
		unsorted.Body(hx.EachMap(scores, func(k string, v int) hx.INode {
			return hx.Li().Text(fmt.Sprintf("%s: %d", k, v))
		}))
	})
	sorted := hx.Ul()
	sorted.Effect(func() {
		sorted.Body(hx.EachMapSorted(scores, func(k string, v int) hx.INode {
			return hx.Li().Text(fmt.Sprintf("%s: %d", k, v))
		}))
	})

	return card("EachMap vs EachMapSorted",
		note("EachMap has no defined order (it can change on every render). EachMapSorted is ordered by key."),
		row(
			btn("random +1", func() {
				scores.Update(func(m map[string]int) map[string]int {
					m = maps.Clone(m)
					keys := slices.Collect(maps.Keys(m))
					m[keys[rand.Intn(len(keys))]]++
					return m
				})
			}),
			btn("add key", func() {
				extra++
				scores.Update(func(m map[string]int) map[string]int {
					m = maps.Clone(m)
					m[fmt.Sprintf("user%d", extra)] = extra
					return m
				})
			}),
		),
		row(hx.Div().Body(hx.Strong().Text("EachMap"), unsorted), hx.Div().Body(hx.Strong().Text("EachMapSorted"), sorted)),
	)
}

// textChildrenCard has a node with bound text and children at the same time.
func textChildrenCard() hx.INode {
	words := []string{"alpha", "bravo", "charlie", "delta", "echo"}
	label := hx.Signal("alpha")

	box := hx.Div().Class("child")
	box.BindText(label)
	box.Body(hx.Span().Class("badge").Text("child node"))

	return card("Text and children together",
		note("Changing the text does not remove the child node."),
		box,
		row(btn("change text", func() { label.Set(words[rand.Intn(len(words))]) })),
	)
}

type fruit struct {
	id   int
	name string
}

// keyedCard shows the same list twice. Type something in the inputs, then press
// the buttons: Each rebuilds the rows (the text is lost), EachKeyed keeps them.
func keyedCard() hx.INode {
	items := hx.Signal([]fruit{{1, "apple"}, {2, "banana"}, {3, "cherry"}, {4, "grape"}})
	nextID := 4

	plain := hx.Ul()
	plain.Effect(func() {
		plain.Body(hx.Each(items, func(i int, f fruit) hx.INode {
			return hx.Li().Body(hx.Span().Textf("%d. %s ", i+1, f.name), hx.Input().Placeholder("type here"))
		}))
	})

	keyed := hx.Ul().Body(hx.EachKeyed(items,
		func(f fruit) int { return f.id },
		func(id int, f hx.Gettable[fruit], index hx.Gettable[int]) hx.INode {
			label := hx.Computed(func() string { return fmt.Sprintf("%d. %s ", index.Get()+1, f.Get().name) })
			return hx.Li().Body(hx.Span().BindText(label), hx.Input().Placeholder("type here"))
		}))

	change := func(fn func([]fruit) []fruit) {
		items.Update(func(l []fruit) []fruit { return fn(slices.Clone(l)) })
	}

	return wideCard("Each vs EachKeyed",
		note("Type in some inputs, then press a button. Each rebuilds every row, so the text is lost. EachKeyed builds a row once per key and keeps its DOM element, so the text and the focus stay."),
		row(
			btn("add at start", func() {
				nextID++
				change(func(l []fruit) []fruit { return append([]fruit{{nextID, fmt.Sprintf("fruit %d", nextID)}}, l...) })
			}),
			btn("remove first", func() {
				change(func(l []fruit) []fruit {
					if len(l) == 0 {
						return l
					}
					return l[1:]
				})
			}),
			btn("shuffle", func() {
				change(func(l []fruit) []fruit { rand.Shuffle(len(l), func(i, j int) { l[i], l[j] = l[j], l[i] }); return l })
			}),
			btn("reverse", func() { change(func(l []fruit) []fruit { slices.Reverse(l); return l }) }),
			btn("rename first", func() {
				change(func(l []fruit) []fruit {
					if len(l) > 0 {
						l[0].name = strings.ToUpper(l[0].name)
					}
					return l
				})
			}),
		),
		row(
			hx.Div().Body(hx.Strong().Text("Each (rebuilds)"), plain),
			hx.Div().Body(hx.Strong().Text("EachKeyed (reuses)"), keyed),
		),
	)
}
