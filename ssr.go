package hx

import (
	"html"
	"strings"
	"sync"
)

// This file has what a server needs to produce the HTML of the UI, and what
// code shared by the server and the browser needs to tell them apart.

// renderMu serializes RenderToString. The tracking of effects uses global state,
// so two renders cannot run at the same time.
var renderMu sync.Mutex

type mountMode int

const (
	mountNow     mountMode = iota // OnMount runs the function right away
	mountCollect                  // OnMount keeps the function until the UI is hydrated
	mountDiscard                  // OnMount ignores the function (rendering on a server)
)

var (
	currentMountMode = mountNow
	mountQueue       []func()
)

// OnMount runs fn only in the browser, once the UI is on the page.
//
// While the UI is being rendered to a string on the server the function is
// ignored. While Hydrate is building the UI it waits until the page has been
// hydrated. In any other case it runs right away. Use it for what needs the
// browser: timers, local storage, focusing an input...
func OnMount(fn func()) {
	switch currentMountMode {
	case mountCollect:
		mountQueue = append(mountQueue, fn)
	case mountNow:
		fn()
	}
}

// RenderToString builds the UI by calling build and returns its HTML.
//
// build runs inside a root, so the effects it creates run once (to produce the
// initial state) and are released before RenderToString returns. Functions
// registered with OnMount are ignored. Calls are serialized: only one UI is
// rendered at a time.
//
// The browser can take this HTML and bring it to life with Hydrate, as long as
// it builds the same UI with the same initial state.
func RenderToString(build func() INode) string {
	renderMu.Lock()
	defer renderMu.Unlock()

	previousMode := currentMountMode
	currentMountMode = mountDiscard
	defer func() { currentMountMode = previousMode }()

	var result string
	CreateRoot(func(dispose func()) struct{} {
		defer dispose()
		node := build()
		if node == nil {
			return struct{}{}
		}
		var renderer StringRenderer
		renderer.Render(node)
		result = renderer.String()
		node.deactivate()
		return struct{}{}
	})
	return result
}

// StateScript returns a <script> element that carries json to the browser. Put
// it in the page after the HTML of the UI, and read it there with ReadState.
// The json is escaped so it can never close the script element.
func StateScript(id, json string) string {
	escaped := strings.NewReplacer("<", "\\u003c", ">", "\\u003e", "&", "\\u0026", "\u2028", "\\u2028", "\u2029", "\\u2029").Replace(json)
	return `<script type="application/json" id="` + html.EscapeString(id) + `">` + escaped + `</script>`
}
