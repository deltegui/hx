package hx

import "strings"

// H1 creates a <h1> element.
func H1() INode { return current().Element("H1", "") }

// H2 creates a <h2> element.
func H2() INode { return current().Element("H2", "") }

// H3 creates a <h3> element.
func H3() INode { return current().Element("H3", "") }

// H4 creates a <h4> element.
func H4() INode { return current().Element("H4", "") }

// H5 creates a <h5> element.
func H5() INode { return current().Element("H5", "") }

// H6 creates a <h6> element.
func H6() INode { return current().Element("H6", "") }

// Div creates a <div> element.
func Div() INode { return current().Element("DIV", "") }

// P creates a <p> element.
func P() INode { return current().Element("P", "") }

// Button creates a <button> element.
func Button() INode { return current().Element("BUTTON", "") }

// I creates a <i> element.
func I() INode { return current().Element("I", "") }

// A creates a <a> element.
func A() INode { return current().Element("A", "") }

// Span creates a <span> element.
func Span() INode { return current().Element("SPAN", "") }

// Strong creates a <strong> element.
func Strong() INode { return current().Element("STRONG", "") }

// Em creates a <em> element.
func Em() INode { return current().Element("EM", "") }

// Small creates a <small> element.
func Small() INode { return current().Element("SMALL", "") }

// Img creates a <img> element.
func Img() INode { return current().Element("IMG", "") }

// Input creates a <input> element.
func Input() INode { return current().Element("INPUT", "") }

// Label creates a <label> element.
func Label() INode { return current().Element("LABEL", "") }

// Form creates a <form> element.
func Form() INode { return current().Element("FORM", "") }

// Select creates a <select> element.
func Select() INode { return current().Element("SELECT", "") }

// Option creates a <option> element.
func Option() INode { return current().Element("OPTION", "") }

// TextArea creates a <textarea> element.
func TextArea() INode { return current().Element("TEXTAREA", "") }

// Ul creates a <ul> element.
func Ul() INode { return current().Element("UL", "") }

// Ol creates a <ol> element.
func Ol() INode { return current().Element("OL", "") }

// Li creates a <li> element.
func Li() INode { return current().Element("LI", "") }

// Table creates a <table> element.
func Table() INode { return current().Element("TABLE", "") }

// THead creates a <thead> element.
func THead() INode { return current().Element("THEAD", "") }

// TBody creates a <tbody> element.
func TBody() INode { return current().Element("TBODY", "") }

// TFoot creates a <tfoot> element.
func TFoot() INode { return current().Element("TFOOT", "") }

// Tr creates a <tr> element.
func Tr() INode { return current().Element("TR", "") }

// Th creates a <th> element.
func Th() INode { return current().Element("TH", "") }

// Td creates a <td> element.
func Td() INode { return current().Element("TD", "") }

// Nav creates a <nav> element.
func Nav() INode { return current().Element("NAV", "") }

// Header creates a <header> element.
func Header() INode { return current().Element("HEADER", "") }

// Footer creates a <footer> element.
func Footer() INode { return current().Element("FOOTER", "") }

// Main creates a <main> element.
func Main() INode { return current().Element("MAIN", "") }

// Section creates a <section> element.
func Section() INode { return current().Element("SECTION", "") }

// Article creates a <article> element.
func Article() INode { return current().Element("ARTICLE", "") }

// Aside creates a <aside> element.
func Aside() INode { return current().Element("ASIDE", "") }

// Video creates a <video> element.
func Video() INode { return current().Element("VIDEO", "") }

// Audio creates a <audio> element.
func Audio() INode { return current().Element("AUDIO", "") }

// Source creates a <source> element.
func Source() INode { return current().Element("SOURCE", "") }

// Canvas creates a <canvas> element.
func Canvas() INode { return current().Element("CANVAS", "") }

// Svg creates an <svg> element in the SVG namespace.
func Svg() INode { return current().Element("svg", SvgNamespace) }

// Path creates an SVG <path> element.
func Path() INode { return current().Element("path", SvgNamespace) }

// Br creates a <br> element.
func Br() INode { return current().Element("BR", "") }

// Hr creates a <hr> element.
func Hr() INode { return current().Element("HR", "") }

// Code creates a <code> element.
func Code() INode { return current().Element("CODE", "") }

// Pre creates a <pre> element.
func Pre() INode { return current().Element("PRE", "") }

// Noop is a fragment: it renders its children but no element of its own.
func Noop() INode { return current().Fragment() }

// Raw is HTML that is written as it is, with no escaping, where the node is. It
// has no element of its own, like Noop: what it holds becomes siblings of the
// nodes around it.
//
// Raw is for HTML you trust: markup you made, or text that has been through a
// sanitizer. NEVER give it something a user typed, or the page is open to script
// injection. For text use Text, which escapes it.
//
// On a server the HTML goes between two comments, <!--hx-raw--> and
// <!--/hx-raw-->, so the browser knows where it ends when it hydrates the page;
// it adopts it as it is and never touches it. A script inside it runs when the
// server's page loads, and does not when the browser creates the node. Setting
// attributes, classes, styles, events or children on a Raw does nothing.
func Raw(html string) INode { return current().Raw(html) }

// El creates a node with any tag, for the ones that have no constructor.
func El(tag string) INode { return current().Element(strings.ToUpper(tag), "") }

// SvgNamespace is the namespace of the SVG elements.
const SvgNamespace = "http://www.w3.org/2000/svg"
