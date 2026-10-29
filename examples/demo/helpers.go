//go:build js && wasm

package main

import (
	"fmt"
	"slices"
	"syscall/js"

	"github.com/deltegui/hx"
)

func card(title string, children ...hx.INode) hx.INode {
	return hx.Section().Class("card").Body(
		append([]hx.INode{hx.H2().Text(title)}, children...)...,
	)
}

func wideCard(title string, children ...hx.INode) hx.INode {
	return card(title, children...).Class("wide")
}

func note(text string) hx.INode { return hx.P().Class("note").Text(text) }

func row(children ...hx.INode) hx.INode { return hx.Div().Class("row").Body(children...) }

func btn(label string, onClick func()) hx.INode {
	return hx.Button().Text(label).OnClick(func(hx.EventContext) { onClick() })
}

// live is a span whose text follows whatever f reads.
func live(f func() string) hx.INode {
	return hx.Span().BindText(hx.Computed(f))
}

func pill(f func() string) hx.INode {
	return hx.Span().Class("pill").BindText(hx.Computed(f))
}

// slider is a range input bound to a string signal.
func slider(sig *hx.SignalT[string], min, max string) hx.INode {
	return hx.Input().Type("range").Attribute("min", min).Attribute("max", max).BindValue(sig).BindOnInput(sig)
}

// logger is a small reactive log that shows the last lines.
type logger struct {
	lines *hx.SignalT[[]string]
}

func newLog() *logger { return &logger{lines: hx.Signal([]string{})} }

func (l *logger) Add(format string, args ...any) {
	line := fmt.Sprintf(format, args...)
	l.lines.Update(func(lines []string) []string {
		lines = append(slices.Clone(lines), line)
		if len(lines) > 12 {
			lines = lines[len(lines)-12:]
		}
		return lines
	})
}

func (l *logger) Clear() { l.lines.Set([]string{}) }

func (l *logger) View() hx.INode {
	ul := hx.Ul().Class("log")
	ul.Effect(func() {
		ul.Body(hx.Each(l.lines, func(i int, line string) hx.INode {
			return hx.Li().Text(line)
		}))
	})
	return ul
}

func now() float64 { return js.Global().Get("performance").Call("now").Float() }

// nextFrame blocks until the browser has run the next animation frame.
// Call it from a goroutine, never from an event handler.
func nextFrame() {
	done := make(chan struct{})
	var cb js.Func
	cb = js.FuncOf(func(js.Value, []js.Value) any {
		close(done)
		cb.Release()
		return nil
	})
	js.Global().Call("requestAnimationFrame", cb)
	<-done
}

func settle() {
	nextFrame()
	nextFrame()
}
