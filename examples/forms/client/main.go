//go:build js && wasm

package main

import (
	"encoding/json"
	"log"

	"github.com/deltegui/hx"
	"hxforms/app"
)

func main() {
	var state app.State
	if err := json.Unmarshal([]byte(hx.ReadState("hx-state")), &state); err != nil {
		log.Printf("could not read the state of the page: %v", err)
	}

	hx.Hydrate("app", func() hx.INode { return app.App(state) })

	select {}
}
