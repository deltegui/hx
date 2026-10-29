//go:build js && wasm

package main

import (
	"fmt"
	"strconv"
	"syscall/js"

	"github.com/deltegui/hx"
)

func reactivityPage() hx.INode {
	return hx.Div().Class("content").Body(
		signalCard(),
		trackingCard(),
		temperatureCard(),
		cleanupCard(),
	)
}

func signalCard() hx.INode {
	a := hx.Signal(1)
	b := hx.Signal(2)
	sum := hx.Computed(func() int { return a.Get() + b.Get() })
	even := hx.Computed(func() bool { return sum.Get()%2 == 0 })

	return card("Signal + Computed",
		note("sum is a Computed of a and b. 'even' is a Computed of sum, so Computeds chain."),
		row(
			live(func() string { return fmt.Sprintf("a = %d", a.Get()) }),
			btn("a++", func() { a.Update(func(n int) int { return n + 1 }) }),
			live(func() string { return fmt.Sprintf("b = %d", b.Get()) }),
			btn("b++", func() { b.Update(func(n int) int { return n + 1 }) }),
		),
		hx.P().Class("big-number").BindText(hx.Computed(func() string {
			return fmt.Sprintf("a + b = %d", sum.Get())
		})),
		row(pill(func() string {
			if even.Get() {
				return "even"
			}
			return "odd"
		})),
	)
}

func trackingCard() hx.INode {
	a := hx.Signal(0)
	b := hx.Signal(0)
	runsBoth := hx.Signal(0)
	runsUntracked := hx.Signal(0)

	holder := hx.Div()
	holder.Effect(func() {
		a.Get()
		b.Get()
		runsBoth.Update(func(n int) int { return n + 1 })
	})
	holder.Effect(func() {
		a.Get()
		hx.UntrackGet[int](b) // b is read without subscribing
		runsUntracked.Update(func(n int) int { return n + 1 })
	})

	return card("Tracking and Untrack",
		note("Effect 1 reads a and b. Effect 2 reads a and UntrackGet(b). Change b: only effect 1 runs. Each Set runs an effect once."),
		row(
			btn("a++", func() { a.Update(func(n int) int { return n + 1 }) }),
			btn("b++", func() { b.Update(func(n int) int { return n + 1 }) }),
		),
		row(
			live(func() string { return fmt.Sprintf("effect 1 runs: %d", runsBoth.Get()) }),
			live(func() string { return fmt.Sprintf("effect 2 runs: %d", runsUntracked.Get()) }),
		),
		holder,
	)
}

func temperatureCard() hx.INode {
	celsius := hx.Signal("20")
	fahrenheit := hx.Computed(func() string {
		c, err := strconv.ParseFloat(celsius.Get(), 64)
		if err != nil {
			return "?"
		}
		return fmt.Sprintf("%.1f", c*9/5+32)
	})
	feeling := hx.Computed(func() string {
		c, _ := strconv.ParseFloat(celsius.Get(), 64)
		switch {
		case c < 5:
			return "freezing"
		case c < 18:
			return "cold"
		case c < 27:
			return "nice"
		}
		return "hot"
	})

	return card("Two-way binding + Computed",
		note("The slider and the number box share one signal (BindValue + BindOnInput)."),
		row(slider(celsius, "-20", "45"), numberBox(celsius)),
		row(
			live(func() string { return celsius.Get() + " °C" }),
			live(func() string { return "= " + fahrenheit.Get() + " °F" }),
			pill(feeling.Get),
		),
	)
}

func numberBox(sig *hx.SignalT[string]) hx.INode {
	return hx.Input().Type("number").BindValue(sig).BindOnInput(sig)
}

// cleanupCard shows OnCleanup: the interval is stopped and its js.Func released
// each time the effect re-runs, and when the page is closed.
func cleanupCard() hx.INode {
	running := hx.Signal(false)
	interval := hx.Signal("250")
	ticks := hx.Signal(0)
	alive := hx.Signal(0)

	holder := hx.Div()
	holder.Effect(func() {
		if !running.Get() {
			return
		}
		ms, _ := strconv.Atoi(interval.Get())
		if ms < 50 {
			ms = 50
		}
		cb := js.FuncOf(func(js.Value, []js.Value) any {
			ticks.Update(func(n int) int { return n + 1 })
			return nil
		})
		id := js.Global().Call("setInterval", cb, ms)
		alive.Update(func(n int) int { return n + 1 })
		hx.OnCleanup(func() {
			js.Global().Call("clearInterval", id)
			cb.Release()
			alive.Update(func(n int) int { return n - 1 })
		})
	})

	return card("Effect + OnCleanup",
		note("The effect starts a setInterval and registers OnCleanup to stop it. Change the interval while running: the old one is cleaned first. Switch tabs: it is cleaned too."),
		row(
			btn("Start / stop", func() { running.Update(func(b bool) bool { return !b }) }),
			slider(interval, "50", "1000"),
			live(func() string { return interval.Get() + " ms" }),
		),
		row(
			live(func() string { return fmt.Sprintf("running: %v", running.Get()) }),
			live(func() string { return fmt.Sprintf("ticks: %d", ticks.Get()) }),
			live(func() string { return fmt.Sprintf("intervals alive: %d", alive.Get()) }),
		),
		holder,
	)
}
