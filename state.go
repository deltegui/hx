package hx

import (
	"html"
	"strings"
)

// StateScript returns a <script> element that carries json to the browser. Put
// it in the page after the HTML of the UI, and read it there with ReadState.
// The json is escaped so it can never close the script element.
func StateScript(id, json string) string {
	escaped := strings.NewReplacer("<", "\\u003c", ">", "\\u003e", "&", "\\u0026", "\u2028", "\\u2028", "\u2029", "\\u2029").Replace(json)
	return `<script type="application/json" id="` + html.EscapeString(id) + `">` + escaped + `</script>`
}
