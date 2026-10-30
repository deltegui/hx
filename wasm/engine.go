// Package wasm is the engine of the browser: it keeps the tree of nodes, applies
// the changes to the DOM once per frame and brings the HTML of a server to life
// (Hydrate). Import it, usually with a blank import, to use Hix in a page:
//
//	import "github.com/deltegui/hx/wasm"
//
//	wasm.NewFromId("app").Body(hx.H1().Text("Hello"))
//
// The part that talks to the DOM only builds for GOOS=js GOARCH=wasm. The rest
// also builds for a server, which is how the nodes are tested.
package wasm

import "github.com/deltegui/hx"

func init() { Use() }

// Use makes this the engine in use. Importing the package already does, so it is
// only needed to go back to this engine after another one was chosen, in a test.
func Use() { hx.UseEngine(engine{}) }

type mountMode int

const (
	mountNow     mountMode = iota // OnMount runs the function right away
	mountCollect                  // OnMount keeps the function until the UI is hydrated
)

var (
	currentMountMode = mountNow
	mountQueue       []func()
)

type engine struct{}

func (engine) Element(tag, namespace string) hx.INode {
	if namespace != "" {
		return newVNodeNS(tag, namespace)
	}
	return newVNode(tag)
}

func (engine) Fragment() hx.INode { return newVNode(noopIdNode) }

func (engine) Raw(html string) hx.INode {
	n := newVNode(rawIdNode)
	n.rawHTML = html
	return n
}

func (engine) OnMount(fn func()) {
	if currentMountMode == mountCollect {
		mountQueue = append(mountQueue, fn)
		return
	}
	fn()
}

func (engine) IsServer() bool { return isServer() }

func (engine) Tracks() bool { return true }
