//go:build js && wasm

package main

import (
	"fmt"

	"github.com/deltegui/hx"
)

func stressPage() hx.INode {
	items := hx.Signal([]*hx.SignalT[int]{})
	result := hx.Signal("-")

	// timed runs fn and then waits for the browser to paint the result.
	timed := func(label string, fn func()) {
		start := now()
		fn()
		go func() {
			settle()
			result.Set(fmt.Sprintf("%s: %.0f ms (including 2 frames)", label, now()-start))
		}()
	}

	fill := func(n int) {
		list := make([]*hx.SignalT[int], n)
		for i := range list {
			list[i] = hx.Signal(0)
		}
		items.Set(list)
	}

	list := hx.Ul().Class("big")
	list.Effect(func() {
		list.Body(hx.Each(items, func(i int, s *hx.SignalT[int]) hx.INode {
			li := hx.Li()
			li.BindText(hx.Computed(func() string { return fmt.Sprintf("item %d = %d", i, s.Get()) }))
			return li
		}))
	})

	return hx.Div().Class("content").Body(
		wideCard("Many nodes, many bindings",
			note("Each item has its own signal, a Computed and a bound text. Every rebuild disposes the previous items."),
			row(
				btn("render 100", func() { timed("render 100", func() { fill(100) }) }),
				btn("render 1000", func() { timed("render 1000", func() { fill(1000) }) }),
				btn("render 5000", func() { timed("render 5000", func() { fill(5000) }) }),
				btn("increment all", func() {
					timed("increment all", func() {
						for _, s := range items.Get() {
							s.Update(func(n int) int { return n + 1 })
						}
					})
				}),
				btn("clear", func() { timed("clear", func() { items.Set(nil) }) }),
			),
			row(
				live(func() string { return fmt.Sprintf("items: %d", len(items.Get())) }),
				live(result.Get),
			),
			list,
		),
	)
}
