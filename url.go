package hx

import "strings"

// UnsafeURL is what an attribute that holds a URL gets when the URL uses a scheme
// that is not allowed (see SafeAttribute). It is a valid URL that goes nowhere.
const UnsafeURL = "about:invalid#hx-unsafe-url"

// urlAttributes are the attributes whose value the browser follows or loads.
var urlAttributes = map[string]bool{
	"href": true, "src": true, "action": true, "formaction": true,
	"xlink:href": true, "poster": true, "cite": true, "data": true,
	"background": true, "manifest": true, "ping": true,
}

var allowedSchemes = map[string]bool{"http": true, "https": true, "mailto": true, "tel": true}

// AllowURLScheme lets URLs with that scheme (for example "whatsapp") through
// SafeAttribute. Call it at startup, before rendering: it is not safe to call
// while pages are being rendered.
func AllowURLScheme(scheme string) { allowedSchemes[strings.ToLower(scheme)] = true }

// SafeAttribute returns the value an attribute may have. Both engines call it in
// Attribute, Href and Src, so a URL from a user cannot run code: in attributes
// that hold a URL (href, src, action, formaction...), a value with a scheme that
// is not http, https, mailto or tel (javascript:, data:, vbscript:...) becomes
// UnsafeURL. Relative URLs are always fine. Any other attribute is returned as it is.
func SafeAttribute(key, value string) string {
	if value == "" || !urlAttributes[strings.ToLower(key)] {
		return value
	}
	if urlSchemeOK(value) {
		return value
	}
	return UnsafeURL
}

// urlSchemeOK follows the browser: it ignores leading spaces and control
// characters and the tabs and line breaks inside the scheme, which is how
// "java\tscript:" gets past a naive check.
func urlSchemeOK(url string) bool {
	var scheme strings.Builder
	start := true
	for i := 0; i < len(url); i++ {
		c := url[i]
		if start && c <= ' ' {
			continue
		}
		start = false
		switch {
		case c == '\t' || c == '\n' || c == '\r':
			continue
		case c == ':':
			return allowedSchemes[strings.ToLower(scheme.String())]
		case c == '/' || c == '?' || c == '#':
			return true // a path, a query or a fragment came first: it is relative
		}
		scheme.WriteByte(c)
	}
	return true // no scheme
}
