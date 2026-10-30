//go:build js && wasm

package main

import (
	"encoding/json"

	"github.com/deltegui/hx"
	"github.com/deltegui/hx/wasm"
	"hxform/app"
)

func main() {
	var state app.State
	json.Unmarshal([]byte(wasm.ReadState("state")), &state) // what the server wrote in the page

	wasm.Hydrate("app", func() hx.INode { return app.App(state) })
	select {}
}
