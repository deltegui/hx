package ssr_test

import (
	"strings"
	"testing"

	"github.com/deltegui/hx"
	"github.com/deltegui/hx/ssr"
)

// Every constructor makes the element it is named after, and the void ones
// have no closing tag.
func TestTagConstructors(t *testing.T) {
	cases := map[string]func() hx.INode{
		"fieldset": hx.Fieldset, "legend": hx.Legend, "details": hx.Details, "summary": hx.Summary,
		"dialog": hx.Dialog, "figure": hx.Figure, "figcaption": hx.Figcaption, "blockquote": hx.Blockquote,
		"dl": hx.Dl, "dt": hx.Dt, "dd": hx.Dd, "progress": hx.Progress, "meter": hx.Meter,
		"datalist": hx.Datalist, "optgroup": hx.Optgroup, "output": hx.Output, "iframe": hx.Iframe,
		"picture": hx.Picture, "mark": hx.Mark, "kbd": hx.Kbd, "time": hx.Time, "abbr": hx.Abbr,
		"b": hx.B, "u": hx.U, "sub": hx.Sub, "sup": hx.Sup, "caption": hx.Caption, "colgroup": hx.Colgroup,
		"hgroup": hx.Hgroup, "address": hx.Address, "cite": hx.Cite, "q": hx.Q, "samp": hx.Samp,
		"var": hx.Var, "del": hx.Del, "ins": hx.Ins, "noscript": hx.Noscript, "template": hx.Template,
		"title": hx.Title, "script": hx.Script, "style": hx.Style,
	}
	for tag, mk := range cases {
		got := ssr.Render(func() hx.INode { return mk().Text("x") })
		if want := "<" + tag + ">x</" + tag + ">"; got != want {
			t.Errorf("%s: got %q, want %q", tag, got, want)
		}
	}
	for tag, mk := range map[string]func() hx.INode{"col": hx.Col, "meta": hx.Meta, "link": hx.Link, "wbr": hx.Wbr} {
		got := ssr.Render(mk)
		if !strings.HasPrefix(got, "<"+tag) || strings.Contains(got, "</"+tag) {
			t.Errorf("%s (void): got %q", tag, got)
		}
	}
}
