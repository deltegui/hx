package hx

import (
	"html"
	"strings"
)

var voidElements = map[string]bool{
	"AREA": true, "BASE": true, "BR": true, "COL": true,
	"EMBED": true, "HR": true, "IMG": true, "INPUT": true,
	"LINK": true, "META": true, "PARAM": true, "SOURCE": true,
	"TRACK": true, "WBR": true,
}

// StringRenderer writes a tree of nodes as HTML. Use the zero value, call Render and then String.
type StringRenderer struct {
	buff strings.Builder
}

// ScheduleRender does nothing: a StringRenderer renders when you call Render.
func (ssr *StringRenderer) ScheduleRender() {}

// Mark does nothing: a StringRenderer renders when you call Render.
func (ssr *StringRenderer) Mark(element *VNode) {}

// Render appends the HTML of the node and its children. Noop nodes and removed entries are skipped.
func (ssr *StringRenderer) Render(current *VNode) {
	if current.status == changeDeleted {
		return
	}
	if current.tag == noopIdNode {
		ssr.writeChildren(current)
		return
	}
	ssr.writeOpenTag(current)
	if !voidElements[current.tag] {
		ssr.writeBody(current)
		ssr.writeCloseTag(current)
	}
}

func (ssr *StringRenderer) writeChildren(current *VNode) {
	for _, child := range current.children {
		if child != nil {
			ssr.Render(child)
		}
	}
}

func (ssr *StringRenderer) writeBody(current *VNode) {
	ssr.buff.WriteString(html.EscapeString(current.text.nextValue))
	ssr.writeChildren(current)
}

func (ssr *StringRenderer) writeCloseTag(current *VNode) {
	ssr.buff.WriteString("</")
	ssr.buff.WriteString(current.tag)
	ssr.buff.WriteString(">")
}

func (ssr *StringRenderer) writeOpenTag(current *VNode) {
	ssr.buff.WriteRune('<')
	ssr.buff.WriteString(current.tag)
	ssr.writeAttributes(current)

	if voidElements[current.tag] {
		ssr.buff.WriteString("/>")
	} else {
		ssr.buff.WriteRune('>')
	}
}

func (ssr *StringRenderer) writeAttribute(name, value string) {
	ssr.buff.WriteRune(' ')
	ssr.buff.WriteString(name)
	ssr.buff.WriteString(`="`)
	ssr.buff.WriteString(html.EscapeString(value))
	ssr.buff.WriteString(`"`)
}

func (ssr *StringRenderer) writeAttributes(current *VNode) {
	if id := current.id.nextValue; id != "" {
		ssr.writeAttribute("id", id)
	}

	classes := make([]string, 0, len(current.classes))
	for _, cls := range sortedKeys(current.classes) {
		if current.classes[cls] != changeDeleted {
			classes = append(classes, cls)
		}
	}
	if len(classes) > 0 {
		ssr.writeAttribute("class", strings.Join(classes, " "))
	}

	var styles strings.Builder
	for _, styleName := range sortedKeys(current.styles) {
		styleValue := current.styles[styleName]
		if styleValue.status == changeDeleted {
			continue
		}
		styles.WriteString(styleName)
		styles.WriteRune(':')
		styles.WriteString(styleValue.Value())
		styles.WriteRune(';')
	}
	if styles.Len() > 0 {
		ssr.writeAttribute("style", styles.String())
	}

	if value := current.value.nextValue; value != "" && current.tag != "TEXTAREA" {
		ssr.writeAttribute("value", value)
	}

	for _, attrName := range sortedKeys(current.attributes) {
		attrValue := current.attributes[attrName]
		if attrValue.status == changeDeleted {
			continue
		}
		ssr.writeAttribute(attrName, attrValue.Value())
	}
}

// String returns the HTML written so far.
func (ssr *StringRenderer) String() string {
	return ssr.buff.String()
}

// Reset discards the HTML written so far.
func (ssr *StringRenderer) Reset() {
	ssr.buff.Reset()
}
