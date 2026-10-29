//go:build js && wasm

package main

import (
	"encoding/json"

	"github.com/deltegui/hx"
	"hxform/app"
)

func main() {
	var state app.State
	json.Unmarshal([]byte(hx.ReadState("state")), &state) // what the server wrote in the page

	hx.Hydrate("app", func() hx.INode { return app.App(state) })
	select {}
}
