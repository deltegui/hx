package ssr

import (
	"fmt"
	"html"
	"log"
	"sort"
	"strings"

	"github.com/deltegui/hx"
)

var voidElements = map[string]bool{
	"AREA": true, "BASE": true, "BR": true, "COL": true,
	"EMBED": true, "HR": true, "IMG": true, "INPUT": true,
	"LINK": true, "META": true, "PARAM": true, "SOURCE": true,
	"TRACK": true, "WBR": true,
}

// node is what the server needs of a node: the state it has when the UI is built.
// The maps are made when the first entry is written, since most nodes have few.
type node struct {
	tag, namespace string
	fragment       bool
	id, text       string
	value          string
	checked        bool
	classes        map[string]struct{}
	styles         map[string]string
	attributes     map[string]string
	children       []*node
}

func asNode(n hx.INode) *node {
	if v, ok := n.(*node); ok {
		return v
	}
	panic("ssr: the node was not made by the server engine; a program cannot mix engines")
}

// Content

func (n *node) Body(children ...hx.INode) hx.INode { return n.BodyList(children) }

func (n *node) BodyList(children []hx.INode) hx.INode {
	n.children = make([]*node, 0, len(children))
	for _, child := range children {
		if child != nil {
			n.children = append(n.children, asNode(child))
		}
	}
	return n
}

func (n *node) Text(text string) hx.INode { n.text = text; return n }

func (n *node) Textf(format string, args ...any) hx.INode {
	return n.Text(fmt.Sprintf(format, args...))
}

func (n *node) BindText(signal hx.Gettable[string]) hx.INode { return n.Text(signal.Get()) }

// Attributes

func (n *node) Id(id string) hx.INode { n.id = id; return n }

func (n *node) Attribute(key, value string) hx.INode {
	if key == "" {
		return n
	}
	if n.attributes == nil {
		n.attributes = map[string]string{}
	}
	n.attributes[key] = value
	return n
}

func (n *node) RemoveAttribute(key string) hx.INode { delete(n.attributes, key); return n }

func (n *node) GetAttribute(key string) string { return n.attributes[key] }

func (n *node) HaveAttribute(key, value string) bool {
	v, ok := n.attributes[key]
	return ok && v == value
}

func (n *node) Src(value string) hx.INode         { return n.Attribute("src", value) }
func (n *node) Href(value string) hx.INode        { return n.Attribute("href", value) }
func (n *node) Placeholder(value string) hx.INode { return n.Attribute("placeholder", value) }
func (n *node) Type(value string) hx.INode        { return n.Attribute("type", value) }
func (n *node) Selected() hx.INode                { return n.Attribute("selected", "selected") }

// Classes and styles

func (n *node) Class(classes ...string) hx.INode {
	for _, class := range classes {
		if class == "" {
			continue
		}
		if strings.ContainsAny(class, " \t\n\r") {
			log.Printf("hx: Class(%q): a class cannot have spaces, so it is ignored. Pass them separately: Class(\"a\", \"b\")", class)
			continue
		}
		if n.classes == nil {
			n.classes = map[string]struct{}{}
		}
		n.classes[class] = struct{}{}
	}
	return n
}

func (n *node) RemoveClass(classes ...string) hx.INode {
	for _, class := range classes {
		delete(n.classes, class)
	}
	return n
}

func (n *node) ClassIf(class string, on bool) hx.INode {
	if on {
		return n.Class(class)
	}
	return n.RemoveClass(class)
}

func (n *node) HaveClass(class string) bool { _, ok := n.classes[class]; return ok }

func (n *node) Style(key, value string) hx.INode {
	if n.styles == nil {
		n.styles = map[string]string{}
	}
	n.styles[key] = value
	return n
}

func (n *node) RemoveStyle(key string) hx.INode { delete(n.styles, key); return n }

func (n *node) GetStyle(key string) string { return n.styles[key] }

func (n *node) HaveStyle(key, value string) bool {
	v, ok := n.styles[key]
	return ok && v == value
}

// Form controls

func (n *node) Value(value string) hx.INode { n.value = value; return n }

func (n *node) Checked(on bool) hx.INode { n.checked = on; return n }

func (n *node) BindValue(signal hx.Gettable[string]) hx.INode {
	v := signal.Get()
	n.Value(v)
	if n.tag == "TEXTAREA" {
		n.Text(v)
	}
	return n
}

func (n *node) BindChecked(signal hx.Gettable[bool]) hx.INode { return n.Checked(signal.Get()) }

// Events do nothing on a server, and neither do the bindings that write to a signal.

func (n *node) BindOnInput(hx.Settable[string]) hx.INode    { return n }
func (n *node) BindOnChange(hx.Settable[string]) hx.INode   { return n }
func (n *node) BindOnChecked(hx.Settable[bool]) hx.INode    { return n }
func (n *node) On(hx.Event, func(hx.EventContext)) hx.INode { return n }
func (n *node) OnClick(func(hx.EventContext)) hx.INode      { return n }
func (n *node) OnInput(func(hx.EventContext)) hx.INode      { return n }
func (n *node) OnChange(func(hx.EventContext)) hx.INode     { return n }
func (n *node) OnKeyUp(func(hx.EventContext)) hx.INode      { return n }
func (n *node) OnKeyDown(func(hx.EventContext)) hx.INode    { return n }
func (n *node) OnSubmit(func(hx.EventContext)) hx.INode     { return n }
func (n *node) OnFocus(func(hx.EventContext)) hx.INode      { return n }
func (n *node) OnBlur(func(hx.EventContext)) hx.INode       { return n }

// Life of the node: the UI is built once and never changes, so an effect runs
// once and the hooks of the browser never run.

func (n *node) Effect(fn func()) hx.INode  { fn(); return n }
func (n *node) OnRendered(func()) hx.INode { return n }
func (n *node) OnRemoved(func()) hx.INode  { return n }
func (n *node) BindWith(run, stop func())  { run() }

// Writing

func (w *writer) node(root hx.INode) { w.element(asNode(root)) }

func (w *writer) element(n *node) {
	if n.fragment {
		w.children(n)
		return
	}
	tag := strings.ToLower(n.tag)
	w.WriteByte('<')
	w.WriteString(tag)
	w.attributes(n)
	if voidElements[strings.ToUpper(n.tag)] {
		w.WriteString("/>")
		return
	}
	w.WriteByte('>')
	// The HTML parser drops a newline right after <textarea> and <pre>, so one
	// more is written to keep the text the same when the browser reads it.
	if (n.tag == "TEXTAREA" || n.tag == "PRE") && strings.HasPrefix(n.text, "\n") {
		w.WriteByte('\n')
	}
	w.WriteString(html.EscapeString(n.text))
	w.children(n)
	w.WriteString("</")
	w.WriteString(tag)
	w.WriteByte('>')
}

func (w *writer) children(n *node) {
	for _, child := range n.children {
		w.element(child)
	}
}

func (w *writer) attribute(name, value string) {
	w.WriteByte(' ')
	w.WriteString(name)
	w.WriteString(`="`)
	w.WriteString(html.EscapeString(value))
	w.WriteByte('"')
}

func (w *writer) attributes(n *node) {
	if n.id != "" {
		w.attribute("id", n.id)
	}
	if len(n.classes) > 0 {
		w.attribute("class", strings.Join(sortedKeys(n.classes), " "))
	}
	if len(n.styles) > 0 {
		var styles strings.Builder
		for _, name := range sortedKeys(n.styles) {
			styles.WriteString(name)
			styles.WriteByte(':')
			styles.WriteString(n.styles[name])
			styles.WriteByte(';')
		}
		w.attribute("style", styles.String())
	}
	if n.value != "" && n.tag != "TEXTAREA" && n.tag != "SELECT" {
		w.attribute("value", n.value)
	}
	if n.checked {
		w.attribute("checked", "")
	}
	for _, name := range sortedKeys(n.attributes) {
		w.attribute(name, n.attributes[name])
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
