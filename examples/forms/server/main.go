//go:build !(js && wasm)

// Command server renders the page with Hix and handles the forms. The same
// endpoints answer a form sent by the browser itself (HTML) and the one sent by
// the WASM (JSON).
package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/deltegui/hx"
	"hxforms/app"
)

const head = `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Hix forms</title>
<style>
body{font-family:system-ui,sans-serif;max-width:44rem;margin:2rem auto;padding:0 1rem}
label{display:block;margin:.5rem 0}label>span{display:inline-block;min-width:6rem}
input:not([type=checkbox]):not([type=file]),select,textarea{font:inherit;padding:.25rem .4rem;min-width:16rem}
.error{color:#b00020;margin:.1rem 0 .4rem 6rem;min-height:1.1em}.banner{color:#0a6b2d;font-weight:600}
.contact{margin:.8rem 0;padding:.5rem;border:1px solid #ccc;border-radius:6px;list-style:none}.contact .files{margin:.2rem 0;font-size:.85rem;color:#555}
ul{padding:0}.row{display:flex;gap:.4rem;flex-wrap:wrap}
</style>
</head>
<body>
<noscript><p>JavaScript is off. The forms still work: the server handles them.</p></noscript>
`

// The script counts what changes inside #app once it has been parsed.
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

type server struct {
	mu       sync.Mutex
	contacts []app.Contact
	nextID   int
	stats    map[string]int
}

func (s *server) count(name string) {
	s.mu.Lock()
	s.stats[name]++
	s.mu.Unlock()
}

func (s *server) list() []app.Contact {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]app.Contact, len(s.contacts))
	copy(out, s.contacts)
	return out
}

// csrf returns the token of this browser, and gives one if it has none.
func csrf(w http.ResponseWriter, r *http.Request) string {
	if cookie, err := r.Cookie("csrf"); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	raw := make([]byte, 16)
	rand.Read(raw)
	token := hex.EncodeToString(raw)
	http.SetCookie(w, &http.Cookie{Name: "csrf", Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
	return token
}

// validCSRF checks the token in the cookie against the one in the form field or in the header.
func validCSRF(r *http.Request) bool {
	cookie, err := r.Cookie("csrf")
	if err != nil || cookie.Value == "" {
		return false
	}
	sent := r.Header.Get("X-CSRF-Token")
	if sent == "" {
		sent = r.FormValue("csrf")
	}
	return sent == cookie.Value
}

func wantsJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (s *server) page(w http.ResponseWriter, status int, state app.State) {
	body := hx.RenderToString(func() hx.INode { return app.App(state) })
	data, _ := json.Marshal(state)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprint(w, head, `<div id="app">`, body, `</div>
`, hx.StateScript("hx-state", string(data)), `
`, observer, `
`, loader, `
</body></html>`)
}

func (s *server) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	saved, _ := strconv.Atoi(r.URL.Query().Get("saved"))
	s.page(w, http.StatusOK, app.State{Contacts: s.list(), Saved: saved, CSRF: csrf(w, r)})
}

// fileInfo reads an uploaded file to know its size and its hash. A real
// application would store it.
func fileInfo(header *multipart.FileHeader) (app.FileInfo, error) {
	file, err := header.Open()
	if err != nil {
		return app.FileInfo{}, err
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return app.FileInfo{}, err
	}
	return app.FileInfo{
		Name: header.Filename,
		Size: size,
		Type: header.Header.Get("Content-Type"),
		Hash: hex.EncodeToString(hash.Sum(nil))[:12],
	}, nil
}

func (s *server) saveContact(w http.ResponseWriter, r *http.Request) {
	json := wantsJSON(r)
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, "the form could not be read: "+err.Error(), http.StatusBadRequest)
			return
		}
	} else if err := r.ParseForm(); err != nil {
		http.Error(w, "the form could not be read", http.StatusBadRequest)
		return
	}
	if !validCSRF(r) {
		s.count("rejected")
		http.Error(w, "invalid CSRF token", http.StatusForbidden)
		return
	}

	values := app.Values{
		Name:       r.FormValue("name"),
		Email:      r.FormValue("email"),
		Message:    r.FormValue("message"),
		Country:    r.FormValue("country"),
		Newsletter: r.FormValue("newsletter") == "yes",
	}
	errs := app.Validate(values)
	// A rule only the server can check.
	if _, ok := errs["email"]; !ok && strings.HasSuffix(strings.TrimSpace(values.Email), ".invalid") {
		errs["email"] = "That domain does not accept mail"
	}

	contact := app.Contact{
		Name: strings.TrimSpace(values.Name), Email: strings.TrimSpace(values.Email),
		Message: values.Message, Country: values.Country, Newsletter: values.Newsletter,
	}
	if r.MultipartForm != nil {
		for _, header := range r.MultipartForm.File["avatar"] {
			if header.Filename == "" { // an empty file input sends an empty part
				continue
			}
			info, err := fileInfo(header)
			if err != nil {
				http.Error(w, "the avatar could not be read", http.StatusBadRequest)
				return
			}
			contact.Avatar = &info
		}
		for _, header := range r.MultipartForm.File["attachments"] {
			if header.Filename == "" {
				continue
			}
			info, err := fileInfo(header)
			if err != nil {
				http.Error(w, "an attachment could not be read", http.StatusBadRequest)
				return
			}
			contact.Attachments = append(contact.Attachments, info)
		}
	}

	if len(errs) > 0 {
		if json {
			writeJSON(w, http.StatusUnprocessableEntity, app.Result{Errors: errs})
			return
		}
		s.count("classic-rejected")
		// Without JavaScript the page is rendered again with what was typed and what is wrong.
		s.page(w, http.StatusUnprocessableEntity, app.State{Contacts: s.list(), Values: values, Errors: errs, CSRF: csrf(w, r)})
		return
	}

	s.mu.Lock()
	s.nextID++
	contact.ID = s.nextID
	s.contacts = append([]app.Contact{contact}, s.contacts...)
	s.mu.Unlock()

	if json {
		s.count("json-saves")
		writeJSON(w, http.StatusOK, app.Result{OK: true, Contact: &contact})
		return
	}
	s.count("classic-saves")
	// Post, redirect, get: reloading the page does not send the form again.
	http.Redirect(w, r, fmt.Sprintf("/?saved=%d", contact.ID), http.StatusSeeOther)
}

func (s *server) remove(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.contacts {
		if c.ID == id {
			s.contacts = append(s.contacts[:i:i], s.contacts[i+1:]...)
			return true
		}
	}
	return false
}

func (s *server) deleteClassic(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil || !validCSRF(r) {
		s.count("rejected")
		http.Error(w, "invalid CSRF token", http.StatusForbidden)
		return
	}
	id, _ := strconv.Atoi(r.PathValue("id"))
	s.remove(id)
	s.count("classic-deletes")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *server) deleteJSON(w http.ResponseWriter, r *http.Request) {
	if !validCSRF(r) {
		s.count("rejected")
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid CSRF token"})
		return
	}
	id, _ := strconv.Atoi(r.PathValue("id"))
	if !s.remove(id) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "there is no such contact"})
		return
	}
	s.count("json-deletes")
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func main() {
	addr := flag.String("addr", ":8095", "address to listen on")
	flag.Parse()

	s := &server{stats: map[string]int{}}
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("GET /{$}", s.home)
	mux.HandleFunc("POST /contacts", s.saveContact)
	mux.HandleFunc("POST /contacts/{id}/delete", s.deleteClassic)
	mux.HandleFunc("DELETE /api/contacts/{id}", s.deleteJSON)
	mux.HandleFunc("GET /api/contacts", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.list())
	})
	mux.HandleFunc("GET /api/slow", func(w http.ResponseWriter, r *http.Request) {
		ms, _ := strconv.Atoi(r.URL.Query().Get("ms"))
		select {
		case <-time.After(time.Duration(ms) * time.Millisecond):
			fmt.Fprintf(w, "slept %d ms", ms)
		case <-r.Context().Done(): // the browser gave up
			s.count("slow-abandoned")
		}
	})
	mux.HandleFunc("POST /api/upload", func(w http.ResponseWriter, r *http.Request) {
		n, _ := io.Copy(io.Discard, r.Body)
		writeJSON(w, http.StatusOK, map[string]int64{"bytes": n})
	})
	mux.HandleFunc("POST /debug/reset", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.contacts, s.nextID, s.stats = nil, 0, map[string]int{}
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /debug/stats", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		writeJSON(w, http.StatusOK, s.stats)
	})

	log.Printf("listening on http://localhost%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
