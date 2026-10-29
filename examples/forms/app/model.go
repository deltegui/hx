// Package app is the UI of the example, shared by the server and the browser.
// It must not use syscall/js, so it builds for both.
package app

import (
	"net/mail"
	"strings"
)

// FileInfo is what the server keeps about an uploaded file.
type FileInfo struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Type string `json:"type"`
	Hash string `json:"hash"` // the start of its SHA-256
}

// Contact is a saved contact.
type Contact struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	Message     string     `json:"message"`
	Country     string     `json:"country"`
	Newsletter  bool       `json:"newsletter"`
	Avatar      *FileInfo  `json:"avatar,omitempty"`
	Attachments []FileInfo `json:"attachments,omitempty"`
}

// Values is what the form holds.
type Values struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	Message    string `json:"message"`
	Country    string `json:"country"`
	Newsletter bool   `json:"newsletter"`
}

// Errors maps a field to what is wrong with it.
type Errors map[string]string

// State is what the server knows when it renders the page. The browser gets a
// copy, so both start from the same place.
type State struct {
	Contacts []Contact `json:"contacts"`
	Values   Values    `json:"values"` // what the form has, after a send that failed
	Errors   Errors    `json:"errors"`
	Saved    int       `json:"saved"` // the contact that was just saved without JavaScript
	CSRF     string    `json:"csrf"`
}

// Result is the answer of POST /contacts when the browser asks for JSON.
type Result struct {
	OK      bool     `json:"ok"`
	Contact *Contact `json:"contact,omitempty"`
	Errors  Errors   `json:"errors,omitempty"`
}

// Country is an option of the select.
type Country struct{ Code, Name string }

// Countries are the options of the select.
var Countries = []Country{{"es", "Spain"}, {"fr", "France"}, {"de", "Germany"}, {"us", "United States"}}

// DefaultCountry is the one selected at the start.
const DefaultCountry = "es"

// Validate checks the values. The server uses it, and the browser too, to tell
// the user before sending anything.
func Validate(v Values) Errors {
	errs := Errors{}
	if strings.TrimSpace(v.Name) == "" {
		errs["name"] = "Write your name"
	}
	if address, err := mail.ParseAddress(strings.TrimSpace(v.Email)); err != nil || address.Address != strings.TrimSpace(v.Email) {
		errs["email"] = "That is not an email"
	}
	if len(v.Message) > 200 {
		errs["message"] = "The message is too long (200 characters at most)"
	}
	known := false
	for _, c := range Countries {
		known = known || c.Code == v.Country
	}
	if !known {
		errs["country"] = "Choose a country"
	}
	return errs
}
