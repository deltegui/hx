//go:build js && wasm

package main

import (
	"encoding/json"
	"log"

	"github.com/deltegui/hx"
	"github.com/deltegui/hx/wasm"
	"hxforms/app"
)

func main() {
	var state app.State
	if err := json.Unmarshal([]byte(wasm.ReadState("hx-state")), &state); err != nil {
		log.Printf("could not read the state of the page: %v", err)
	}

	wasm.Hydrate("app", func() hx.INode { return app.App(state) })

	select {}
}
