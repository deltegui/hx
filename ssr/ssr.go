// Package ssr is the engine of the server: it builds the UI as a plain tree and
// writes it as HTML. Import it where the page is rendered:
//
//	import "github.com/deltegui/hx/ssr"
//
//	html := ssr.Render(func() hx.INode { return app.App(state) })
//
// Nothing is tracked and no browser code is linked, so a signal is a plain
// value, an effect runs once, event handlers are ignored and any number of UIs
// can be rendered at the same time. The browser can bring the HTML to life with
// wasm.Hydrate, as long as it builds the same UI with the same initial state.
//
// The engine registers itself when the package is imported, so a program that
// renders on a server must not also import hx/wasm.
package ssr

import (
	"strings"

	"github.com/deltegui/hx"
)

func init() { Use() }

// Use makes this the engine in use. Importing the package already does, so it is
// only needed to go back to this engine after another one was chosen, in a test.
func Use() { hx.UseEngine(engine{}) }

type engine struct{}

func (engine) Element(tag, namespace string) hx.INode { return &node{tag: tag, namespace: namespace} }
func (engine) Fragment() hx.INode                     { return &node{fragment: true} }
func (engine) Raw(html string) hx.INode               { return &node{raw: true, html: html} }
func (engine) OnMount(func())                         {}
func (engine) IsServer() bool                         { return true }
func (engine) Tracks() bool                           { return false }

// Render builds the UI by calling build and returns its HTML. Functions
// registered with hx.OnMount are ignored. It can be called from many
// goroutines at the same time.
func Render(build func() hx.INode) string {
	root := build()
	if root == nil {
		return ""
	}
	var w writer
	w.node(root)
	return w.String()
}

// Node renders a node that is already built.
func Node(root hx.INode) string {
	var w writer
	w.node(root)
	return w.String()
}

type writer struct{ strings.Builder }
