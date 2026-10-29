//go:build js && wasm

package main

import (
	"fmt"
	"slices"

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

	keyedRows := hx.Signal([]int{})
	keyedResult := hx.Signal("-")
	keyedTimed := func(label string, fn func()) {
		start := now()
		fn()
		go func() {
			settle()
			keyedResult.Set(fmt.Sprintf("%s: %.0f ms (including 2 frames)", label, now()-start))
		}()
	}
	keyedFill := func(n int) {
		list := make([]int, n)
		for i := range list {
			list[i] = i
		}
		keyedRows.Set(list)
	}
	keyedList := hx.Ul().Class("big").Body(hx.EachKeyed(keyedRows,
		func(n int) int { return n },
		func(k int, _ hx.Gettable[int], index hx.Gettable[int]) hx.INode {
			return hx.Li().BindText(hx.Computed(func() string { return fmt.Sprintf("row %d at %d", k, index.Get()) }))
		}))

	return hx.Div().Class("content").Body(
		wideCard("EachKeyed with many rows",
			note("The same rows are reused. Reversing moves elements, rotating moves one, 'append' only adds."),
			row(
				btn("render 1000", func() { keyedTimed("render 1000", func() { keyedFill(1000) }) }),
				btn("render 5000", func() { keyedTimed("render 5000", func() { keyedFill(5000) }) }),
				btn("reverse", func() {
					keyedTimed("reverse", func() {
						keyedRows.Update(func(l []int) []int { l = append([]int(nil), l...); slices.Reverse(l); return l })
					})
				}),
				btn("last to first", func() {
					keyedTimed("last to first", func() {
						keyedRows.Update(func(l []int) []int {
							if len(l) < 2 {
								return l
							}
							return append([]int{l[len(l)-1]}, l[:len(l)-1]...)
						})
					})
				}),
				btn("append 100", func() {
					keyedTimed("append 100", func() {
						keyedRows.Update(func(l []int) []int {
							out := append([]int(nil), l...)
							for i := 0; i < 100; i++ {
								out = append(out, len(l)+i+100000)
							}
							return out
						})
					})
				}),
				btn("clear", func() { keyedTimed("clear", func() { keyedRows.Set(nil) }) }),
			),
			row(
				live(func() string { return fmt.Sprintf("rows: %d", len(keyedRows.Get())) }),
				live(keyedResult.Get),
			),
			keyedList,
		),
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
