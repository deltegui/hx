//go:build js && wasm

package main

import (
	"fmt"

	"github.com/deltegui/hx"
)

func lifecyclePage() hx.INode {
	return hx.Div().Class("content").Body(
		rootCard(),
		ownedCard(),
		cachedCard(),
	)
}

func rootCard() hx.INode {
	log := newLog()
	ping := hx.Signal(0)
	var dispose func()
	alive := hx.Signal(false)

	start := func() {
		if dispose != nil {
			return
		}
		dispose = hx.CreateRoot(func(dispose func()) func() {
			for i := 1; i <= 3; i++ {
				hx.EffectFunc(func() {
					log.Add("root effect %d saw ping=%d", i, ping.Get())
				})
			}
			hx.OnCleanup(func() { log.Add("root cleanup ran") })
			return dispose
		})
		alive.Set(true)
	}
	stop := func() {
		if dispose == nil {
			return
		}
		dispose()
		dispose = nil
		alive.Set(false)
	}

	return card("CreateRoot",
		note("Effects created inside a root belong to it. Dispose the root: they all stop, and OnCleanup runs."),
		row(
			btn("Create root", start),
			btn("Dispose root", stop),
			btn("ping", func() { ping.Update(func(n int) int { return n + 1 }) }),
			btn("clear log", log.Clear),
			pill(func() string {
				if alive.Get() {
					return "root alive"
				}
				return "no root"
			}),
		),
		log.View(),
	)
}

func ownedCard() hx.INode {
	log := newLog()
	shared := hx.Signal(0)
	sharedText := hx.Computed(func() string { return fmt.Sprintf("shared = %d", shared.Get()) })
	mounted := hx.Signal(true)
	built := 0

	newChild := func() hx.INode {
		built++
		n := built
		log.Add("child #%d built", n)
		child := hx.Div().Class("child")
		child.Effect(func() {
			log.Add("child #%d effect saw shared=%d", n, shared.Get())
		})
		child.Body(hx.Span().BindText(sharedText))
		return child
	}

	box := hx.Div()
	box.Effect(func() {
		// IfFn only builds the child when it is shown, so nothing is created
		// (and nothing is logged) while it is hidden.
		box.Body(hx.IfFn(mounted, newChild))
	})

	return card("Node-owned effects + IfFn",
		note("The child has an Effect and a Bind. It is built by IfFn: hide it and nothing is built or logged, press 'shared++' and nothing happens. Show it: a new child is built."),
		row(
			btn("Toggle child", func() { mounted.Update(func(b bool) bool { return !b }) }),
			btn("shared++", func() { shared.Update(func(n int) int { return n + 1 }) }),
			btn("clear log", log.Clear),
		),
		box,
		log.View(),
	)
}

func cachedCard() hx.INode {
	shared := hx.Signal(0)
	text := hx.Computed(func() string { return fmt.Sprintf("cached node, shared = %d", shared.Get()) })
	visible := hx.Signal(true)

	// This node is built once and shown and hidden many times. Its binding is
	// released when it is hidden and created again when it comes back.
	cached := hx.Div().Class("child")
	cached.BindText(text)
	hidden := hx.P().Class("note").Text("(hidden)")

	holder := hx.Div()
	holder.Effect(func() {
		holder.Body(hx.Show(visible, cached, hidden))
	})

	return card("Reusing a node with Show",
		note("Hide the node, press 'shared++' a few times, show it again: it catches up with the current value."),
		row(
			btn("Toggle", func() { visible.Update(func(b bool) bool { return !b }) }),
			btn("shared++", func() { shared.Update(func(n int) int { return n + 1 }) }),
		),
		holder,
	)
}
