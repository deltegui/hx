package ssr_test

import (
	"testing"

	"github.com/deltegui/hx"
	"github.com/deltegui/hx/ssr"
	"github.com/deltegui/hx/wasm"
)

// Hydration compares the tree the server wrote with the tree the browser builds,
// so the two engines must write the same HTML for the same UI. The browser
// engine writes it with its StringRenderer in tests.
func TestEnginesWriteTheSameHTML(t *testing.T) {
	cases := map[string]func() hx.INode{
		"attributes and classes": func() hx.INode {
			return hx.Div().Id("x").Class("b", "a", "").Style("color", "red").Style("a", "1").Attribute("title", `a"b<c`).Text("a<b & c")
		},
		"a class with a space is ignored": func() hx.INode {
			return hx.Div().Class("a b", "c", "")
		},
		"removed things": func() hx.INode {
			return hx.Div().Class("a", "b").RemoveClass("a").Style("x", "1").RemoveStyle("x").Attribute("k", "v").RemoveAttribute("k")
		},
		"void elements and inputs": func() hx.INode {
			return hx.Div().Body(hx.Input().Type("text").Value("v").Placeholder("p"), hx.Br(), hx.Img().Src("a.png"), hx.Hr())
		},
		"checkbox": func() hx.INode {
			on := hx.Signal(true)
			return hx.Div().Body(hx.Input().Type("checkbox").BindChecked(on), hx.Input().Type("checkbox"), hx.Input().Type("checkbox").Checked(true))
		},
		"textarea, pre and select": func() hx.INode {
			v := hx.Signal("\nline")
			return hx.Div().Body(hx.TextArea().BindValue(v), hx.Pre().Text("\nx"),
				hx.Select().Body(hx.Option().Value("a").Text("A"), hx.Option().Value("b").Selected().Text("B")))
		},
		"fragments and nil": func() hx.INode {
			return hx.Ul().Body(hx.Noop().Body(hx.Li().Text("1"), nil, hx.Li().Text("2")), nil, hx.Noop())
		},
		"raw html": func() hx.INode {
			return hx.Div().Body(hx.P().Text("a"), hx.Raw("<b>x</b> and <i>y</i>"), hx.Raw(""), hx.Span().Text("z"))
		},
		"raw in a table and in a fragment": func() hx.INode {
			return hx.Table().Body(hx.TBody().Body(hx.Noop().Body(hx.Raw("<tr><td>1</td></tr>")), hx.Raw("<tr><td>2</td></tr>")))
		},
		"svg": func() hx.INode {
			return hx.Svg().Attribute("viewBox", "0 0 1 1").Body(hx.Path().Attribute("d", "M0 0"))
		},
		"bindings": func() hx.INode {
			name := hx.Signal("Ada")
			label := hx.Computed(func() string { return "Hi " + name.Get() })
			return hx.Div().Body(hx.P().BindText(label), hx.Input().BindValue(name).BindOnInput(name), hx.Button().Text("x").OnClick(func(hx.EventContext) {}))
		},
		"node effect": func() hx.INode {
			items := hx.Signal([]string{"a", "b"})
			list := hx.Ul()
			list.Effect(func() {
				list.Body(hx.Each(items, func(i int, v string) hx.INode { return hx.Li().Textf("%d:%s", i, v) }))
			})
			return list
		},
		"conditionals": func() hx.INode {
			return hx.Div().Body(hx.IfFn(hx.Value(false), func() hx.INode { return hx.P().Text("hidden") }), hx.If(hx.Value(true), hx.P().Text("shown")),
				hx.ShowFn(hx.Value(false), func() hx.INode { return hx.P().Text("a") }, func() hx.INode { return hx.P().Text("b") }))
		},
		"dynamic": func() hx.INode {
			n := hx.Signal(2)
			return hx.Ul().Body(hx.Dynamic(func() hx.INode {
				items := []hx.INode{}
				for i := 0; i < n.Get(); i++ {
					items = append(items, hx.Li().Textf("row %d", i))
				}
				return hx.Noop().BodyList(items)
			}))
		},
		"keyed list": func() hx.INode {
			items := hx.Signal([]int{3, 1, 2})
			return hx.Ul().Body(hx.EachKeyed(items, func(i int) int { return i },
				func(k int, _ hx.Gettable[int], idx hx.Gettable[int]) hx.INode {
					return hx.Li().BindText(hx.Computed(func() string { return string(rune('a'+k)) + string(rune('0'+idx.Get())) }))
				}))
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			wasm.Use()
			var r wasm.StringRenderer
			wasm.NewWithoutMount("DIV", &r)
			r.Render(build().(*wasm.VNode))
			fromBrowser := r.String()

			ssr.Use()
			fromServer := ssr.Render(build)

			if fromBrowser != fromServer {
				t.Fatalf("\nbrowser engine: %s\nserver engine:  %s", fromBrowser, fromServer)
			}
		})
	}
	ssr.Use()
}
