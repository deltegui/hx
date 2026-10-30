// Package hxhttp makes requests from the UI. In the browser, Do uses
// XMLHttpRequest, with bodies for forms and files, progress, timeout and cancel.
//
// The types (Request, Response, Body, File...) build on every platform, so code
// shared with a server can use them; Do only works in the browser (see
// xhr_wasm.go) and on a server it returns ErrNotInBrowser.
//
// It does not import encoding/json or net/url on purpose: Go keeps the packages
// a program imports even when they are not used, and encoding/json alone adds
// more than 800 KB to a WASM file built with TinyGo. JSON is in the hxjson
// package.
package hxhttp

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/deltegui/hx"
)

// ErrNotInBrowser is what Do returns when the code runs on a server.
var ErrNotInBrowser = errors.New("hxhttp: requests can only be made in the browser")

// ErrInCallback is what Do returns when it is called inside an event handler, or
// any other callback of JavaScript. Do waits for the answer, and the page cannot
// deliver it until the callback returns, so waiting there would stop the Go
// program with a deadlock. Run it in a goroutine, or in an hx.Action.
var ErrInCallback = errors.New("hxhttp: Do waits for the answer and cannot do that inside an event handler: " +
	"the page could not deliver it, and Go would stop with a deadlock. Run it in a goroutine (go func() { ... }()) or in an hx.Action")

// Request describes a request for Do.
type Request struct {
	Method string            // GET if empty
	URL    string            // absolute, or relative to the page
	Header map[string]string // Content-Type is set from the body, unless it is here
	Body   *Body             // nil for no body

	// Timeout aborts the request after this long. Zero means no timeout.
	Timeout time.Duration

	// Credentials sends cookies on requests to other origins. Requests to the
	// origin of the page always send them.
	Credentials bool

	// OnProgress is called while the body is being uploaded and while the answer
	// is being downloaded. It runs on its own goroutine.
	OnProgress func(Progress)
}

// Progress is how much of an upload or a download has been transferred.
type Progress struct {
	Upload bool  // true for the body of the request, false for the answer
	Loaded int64 // bytes transferred
	Total  int64 // bytes in all, 0 if it is not known
}

// Fraction returns a number from 0 to 1, or 0 if the total is not known.
func (p Progress) Fraction() float64 {
	if p.Total <= 0 {
		return 0
	}
	return float64(p.Loaded) / float64(p.Total)
}

// Response is the answer to a request.
type Response struct {
	Status     int
	StatusText string
	Header     map[string]string // names in lowercase
	body       []byte
}

// OK tells if the status is 2xx.
func (r Response) OK() bool { return r.Status >= 200 && r.Status < 300 }

// Bytes returns the body of the answer.
func (r Response) Bytes() []byte { return r.body }

// Text returns the body of the answer as text.
func (r Response) Text() string { return string(r.body) }

// Err returns nil for a 2xx answer, and an *HTTPError for any other.
func (r Response) Err() error {
	if r.OK() {
		return nil
	}
	return &HTTPError{Status: r.Status, StatusText: r.StatusText, Body: r.Text()}
}

// HTTPError is the error for an answer whose status is not 2xx.
type HTTPError struct {
	Status     int
	StatusText string
	Body       string // the body of the answer, it often says what was wrong
}

func (e *HTTPError) Error() string {
	text := strings.TrimSpace(e.Body)
	if len(text) > 200 {
		text = text[:200] + "..."
	}
	if text == "" {
		return fmt.Sprintf("%d %s", e.Status, e.StatusText)
	}
	return fmt.Sprintf("%d %s: %s", e.Status, e.StatusText, text)
}

type bodyKind int

const (
	bodyText bodyKind = iota + 1
	bodyValues
	bodyForm
	bodyMultipart
)

// Body is what a request sends. Make one with TextBody, FormValues, FormOf or
// MultipartBody, or with hxjson.Body to send JSON.
type Body struct {
	kind        bodyKind
	contentType string
	text        string
	values      map[string][]string
	form        hx.INode
	multipart   *MultipartForm
}

// TextBody sends text with that content type, for example "application/json".
func TextBody(contentType, text string) *Body {
	return &Body{kind: bodyText, contentType: contentType, text: text}
}

// FormValues sends the values like a classic form without files
// (application/x-www-form-urlencoded). A url.Values can be passed as it is.
func FormValues(values map[string][]string) *Body { return &Body{kind: bodyValues, values: values} }

// FormOf sends what a <form> in the page would send: every field with a name,
// including the files chosen in file inputs (multipart/form-data). The node has
// to be a form that has been rendered.
func FormOf(form hx.INode) *Body { return &Body{kind: bodyForm, form: form} }

// MultipartBody sends the fields and files of a MultipartForm (multipart/form-data).
func MultipartBody(m *MultipartForm) *Body { return &Body{kind: bodyMultipart, multipart: m} }

// encode returns the bytes of the bodies that are plain data. FormOf and
// MultipartBody are built by the browser.
func (b *Body) encode() (contentType string, data []byte, err error) {
	switch b.kind {
	case bodyText:
		return b.contentType, []byte(b.text), nil
	case bodyValues:
		return "application/x-www-form-urlencoded", []byte(encodeValues(b.values)), nil
	}
	return "", nil, errors.New("hxhttp: this body is built by the browser")
}

// File is a file the user chose in a file input.
type File struct {
	Name string
	Size int64
	Type string // the MIME type the browser guessed, may be empty
	domFile
}

type part struct {
	name     string
	value    string
	file     *File
	filename string
	mime     string
	data     []byte
	kind     partKind
}

type partKind int

const (
	partField partKind = iota
	partFile
	partBytes
)

// MultipartForm builds a multipart/form-data body by hand, for when there is no
// <form> in the page.
type MultipartForm struct {
	parts []part
}

// NewMultipart creates an empty MultipartForm.
func NewMultipart() *MultipartForm { return &MultipartForm{} }

// Field adds a text field.
func (m *MultipartForm) Field(name, value string) *MultipartForm {
	m.parts = append(m.parts, part{kind: partField, name: name, value: value})
	return m
}

// File adds a file the user chose (see FilesOf). The browser reads it while it
// sends it, so big files are not copied into memory.
func (m *MultipartForm) File(name string, file File) *MultipartForm {
	m.parts = append(m.parts, part{kind: partFile, name: name, file: &file})
	return m
}

// Bytes adds a file made from data.
func (m *MultipartForm) Bytes(name, filename, contentType string, data []byte) *MultipartForm {
	m.parts = append(m.parts, part{kind: partBytes, name: name, filename: filename, mime: contentType, data: data})
	return m
}

// Body returns the body to send.
func (m *MultipartForm) Body() *Body { return MultipartBody(m) }

// encodeValues writes the values as a query string, sorted by name, like
// url.Values.Encode.
func encodeValues(values map[string][]string) string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	var out strings.Builder
	for _, name := range names {
		for _, value := range values[name] {
			if out.Len() > 0 {
				out.WriteByte('&')
			}
			out.WriteString(queryEscape(name))
			out.WriteByte('=')
			out.WriteString(queryEscape(value))
		}
	}
	return out.String()
}

// queryEscape escapes text for a query string: letters, digits and -_.~ stay,
// a space is +, anything else is %XX.
func queryEscape(text string) string {
	const hex = "0123456789ABCDEF"
	var out strings.Builder
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_', c == '.', c == '~':
			out.WriteByte(c)
		case c == ' ':
			out.WriteByte('+')
		default:
			out.WriteByte('%')
			out.WriteByte(hex[c>>4])
			out.WriteByte(hex[c&15])
		}
	}
	return out.String()
}
