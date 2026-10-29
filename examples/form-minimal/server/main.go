//go:build !(js && wasm)

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/deltegui/hx"
	"hxform/app"
)

var (
	mu          sync.Mutex
	subscribers []string
)

func page(w http.ResponseWriter, status int, state app.State) {
	body := hx.RenderToString(func() hx.INode { return app.App(state) })
	data, _ := json.Marshal(state)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprintf(w, `<!doctype html>
<html><head><meta charset="utf-8"><title>Newsletter</title></head>
<body>
<div id="app">%s</div>
%s
<script src="/static/wasm_exec.js"></script>
<script>
  const go = new Go();
  WebAssembly.instantiateStreaming(fetch("/static/main.wasm"), go.importObject).then((r) => go.run(r.instance));
</script>
</body></html>`, body, hx.StateScript("state", string(data)))
}

func subscribe(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(r.FormValue("email")) // the same for a plain form and for FormOf
	problem := app.Validate(email)                   // the server always checks
	wantsJSON := strings.Contains(r.Header.Get("Accept"), "application/json")

	switch {
	case problem != "" && wantsJSON:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprintf(w, `{"ok":false,"problem":%q}`, problem)

	case problem != "": // no JavaScript: the same page again, with what was typed and what is wrong
		page(w, http.StatusUnprocessableEntity, app.State{Email: email, Problem: problem})

	default:
		mu.Lock()
		subscribers = append(subscribers, email)
		mu.Unlock()
		if wantsJSON {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true}`)
			return
		}
		http.Redirect(w, r, "/?done=1", http.StatusSeeOther) // post, redirect, get
	}
}

func main() {
	addr := flag.String("addr", ":8097", "address to listen on")
	flag.Parse()

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		page(w, http.StatusOK, app.State{Done: r.URL.Query().Get("done") == "1"})
	})
	mux.HandleFunc("POST /subscribe", subscribe)
	mux.HandleFunc("GET /subscribers", func(w http.ResponseWriter, r *http.Request) { // for the checks
		mu.Lock()
		defer mu.Unlock()
		json.NewEncoder(w).Encode(subscribers)
	})

	log.Printf("listening on http://localhost%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
