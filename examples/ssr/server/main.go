//go:build !(js && wasm)

// Command server renders the page with Hix and serves it with the WASM that
// hydrates it.
package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strings"

	"github.com/deltegui/hx"
	"github.com/deltegui/hx/ssr"
	"hxssr/app"
)

// The script counts what changes inside #app once it has been parsed. A
// hydration that works must not change anything but the status text.
const observer = `<script>
window.__mutations = {childList: 0, attributes: 0, characterData: 0};
new MutationObserver(function (records) {
  records.forEach(function (r) { window.__mutations[r.type]++; });
}).observe(document.getElementById("app"), {childList: true, subtree: true, attributes: true, characterData: true});
</script>`

const loader = `<script src="/static/wasm_exec.js"></script>
<script>
const go = new Go();
WebAssembly.instantiateStreaming(fetch("/static/main.wasm"), go.importObject)
  .then((result) => go.run(result.instance));
</script>`

func page(state app.State, tamper func(string) string) string {
	body := ssr.Render(func() hx.INode { return app.App(state) })
	if tamper != nil {
		body = tamper(body)
	}
	data, _ := json.Marshal(state)
	return `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Hix server rendering</title>
<style>body{font-family:system-ui,sans-serif;max-width:40rem;margin:2rem auto;padding:0 1rem}li{margin:.2rem 0}</style>
</head>
<body>
<div id="app">` + body + `</div>
` + hx.StateScript("hx-state", string(data)) + `
` + observer + `
` + loader + `
</body></html>`
}

// tampers breaks the HTML on purpose so the hydration has to recover.
var tampers = map[string]func(string) string{
	"tag":   func(h string) string { return strings.Replace(h, `<p id="count">`, `<div id="count">`, 1) },
	"count": func(h string) string { return strings.Replace(h, `<li>`, ``, 1) },
	"text":  func(h string) string { return strings.Replace(h, `count: 3`, `count: 99`, 1) },
	"class": func(h string) string { return strings.Replace(h, `class="page"`, ``, 1) },
	"empty": func(h string) string { return `` },
	"attr":  func(h string) string { return strings.Replace(h, `placeholder="New todo"`, `placeholder="Old"`, 1) },
	"id":    func(h string) string { return strings.Replace(h, `id="draft"`, `id="other"`, 1) },
}

func main() {
	addr := flag.String("addr", ":8090", "address to listen on")
	flag.Parse()

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		state := app.State{Count: 3, Todos: []app.Todo{{ID: 1, Title: "write docs"}, {ID: 2, Title: "ship it"}}}
		if r.URL.Query().Get("empty") == "1" {
			state.Todos = nil
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(page(state, tampers[r.URL.Query().Get("break")])))
	})

	log.Printf("listening on http://localhost%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
