//go:build js && wasm

package main

import (
	"encoding/json"
	"log"

	"github.com/deltegui/hx"
	"github.com/deltegui/hx/wasm"
	"hxssr/app"
)

func main() {
	// The server wrote its state in the page, so both start from the same place.
	var state app.State
	if err := json.Unmarshal([]byte(wasm.ReadState("hx-state")), &state); err != nil {
		log.Printf("could not read the state of the page: %v", err)
	}

	wasm.Hydrate("app", func() hx.INode { return app.App(state) })

	select {}
}
