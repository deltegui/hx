package hx

// The tests of the core run with an engine that has no nodes: signals, actions
// and resources need an engine only for OnMount. The nodes are tested in
// hx/wasm and hx/ssr.
func init() { UseEngine(testEngine{}) }

type testEngine struct{}

// stubNode is a node whose methods are not there: a test can pass it around but
// cannot use it.
type stubNode struct{ INode }

func (testEngine) Element(tag, namespace string) INode { return stubNode{} }
func (testEngine) Fragment() INode                     { return stubNode{} }
func (testEngine) OnMount(fn func())                   { fn() }
func (testEngine) IsServer() bool                      { return false }
func (testEngine) Tracks() bool                        { return true }
