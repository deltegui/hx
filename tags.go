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

// Animate creates an SVG <animate> element.
func Animate() INode { return current().Element("animate", SvgNamespace) }

// AnimateMotion creates an SVG <animateMotion> element.
func AnimateMotion() INode { return current().Element("animateMotion", SvgNamespace) }

// AnimateTransform creates an SVG <animateTransform> element.
func AnimateTransform() INode { return current().Element("animateTransform", SvgNamespace) }

// Circle creates an SVG <circle> element.
func Circle() INode { return current().Element("circle", SvgNamespace) }

// ClipPath creates an SVG <clipPath> element.
func ClipPath() INode { return current().Element("clipPath", SvgNamespace) }

// Defs creates an SVG <defs> element.
func Defs() INode { return current().Element("defs", SvgNamespace) }

// Desc creates an SVG <desc> element.
func Desc() INode { return current().Element("desc", SvgNamespace) }

// Ellipse creates an SVG <ellipse> element.
func Ellipse() INode { return current().Element("ellipse", SvgNamespace) }

// FeBlend creates an SVG <feBlend> element.
func FeBlend() INode { return current().Element("feBlend", SvgNamespace) }

// FeColorMatrix creates an SVG <feColorMatrix> element.
func FeColorMatrix() INode { return current().Element("feColorMatrix", SvgNamespace) }

// FeComponentTransfer creates an SVG <feComponentTransfer> element.
func FeComponentTransfer() INode { return current().Element("feComponentTransfer", SvgNamespace) }

// FeComposite creates an SVG <feComposite> element.
func FeComposite() INode { return current().Element("feComposite", SvgNamespace) }

// FeConvolveMatrix creates an SVG <feConvolveMatrix> element.
func FeConvolveMatrix() INode { return current().Element("feConvolveMatrix", SvgNamespace) }

// FeDiffuseLighting creates an SVG <feDiffuseLighting> element.
func FeDiffuseLighting() INode { return current().Element("feDiffuseLighting", SvgNamespace) }

// FeDisplacementMap creates an SVG <feDisplacementMap> element.
func FeDisplacementMap() INode { return current().Element("feDisplacementMap", SvgNamespace) }

// FeDistantLight creates an SVG <feDistantLight> element.
func FeDistantLight() INode { return current().Element("feDistantLight", SvgNamespace) }

// FeDropShadow creates an SVG <feDropShadow> element.
func FeDropShadow() INode { return current().Element("feDropShadow", SvgNamespace) }

// FeFlood creates an SVG <feFlood> element.
func FeFlood() INode { return current().Element("feFlood", SvgNamespace) }

// FeFuncA creates an SVG <feFuncA> element.
func FeFuncA() INode { return current().Element("feFuncA", SvgNamespace) }

// FeFuncB creates an SVG <feFuncB> element.
func FeFuncB() INode { return current().Element("feFuncB", SvgNamespace) }

// FeFuncG creates an SVG <feFuncG> element.
func FeFuncG() INode { return current().Element("feFuncG", SvgNamespace) }

// FeFuncR creates an SVG <feFuncR> element.
func FeFuncR() INode { return current().Element("feFuncR", SvgNamespace) }

// FeGaussianBlur creates an SVG <feGaussianBlur> element.
func FeGaussianBlur() INode { return current().Element("feGaussianBlur", SvgNamespace) }

// FeImage creates an SVG <feImage> element.
func FeImage() INode { return current().Element("feImage", SvgNamespace) }

// FeMerge creates an SVG <feMerge> element.
func FeMerge() INode { return current().Element("feMerge", SvgNamespace) }

// FeMergeNode creates an SVG <feMergeNode> element.
func FeMergeNode() INode { return current().Element("feMergeNode", SvgNamespace) }

// FeMorphology creates an SVG <feMorphology> element.
func FeMorphology() INode { return current().Element("feMorphology", SvgNamespace) }

// FeOffset creates an SVG <feOffset> element.
func FeOffset() INode { return current().Element("feOffset", SvgNamespace) }

// FePointLight creates an SVG <fePointLight> element.
func FePointLight() INode { return current().Element("fePointLight", SvgNamespace) }

// FeSpecularLighting creates an SVG <feSpecularLighting> element.
func FeSpecularLighting() INode { return current().Element("feSpecularLighting", SvgNamespace) }

// FeSpotLight creates an SVG <feSpotLight> element.
func FeSpotLight() INode { return current().Element("feSpotLight", SvgNamespace) }

// FeTile creates an SVG <feTile> element.
func FeTile() INode { return current().Element("feTile", SvgNamespace) }

// FeTurbulence creates an SVG <feTurbulence> element.
func FeTurbulence() INode { return current().Element("feTurbulence", SvgNamespace) }

// Filter creates an SVG <filter> element.
func Filter() INode { return current().Element("filter", SvgNamespace) }

// ForeignObject creates an SVG <foreignObject> element.
func ForeignObject() INode { return current().Element("foreignObject", SvgNamespace) }

// G creates an SVG <g> element.
func G() INode { return current().Element("g", SvgNamespace) }

// Image creates an SVG <image> element.
func Image() INode { return current().Element("image", SvgNamespace) }

// Line creates an SVG <line> element.
func Line() INode { return current().Element("line", SvgNamespace) }

// LinearGradient creates an SVG <linearGradient> element.
func LinearGradient() INode { return current().Element("linearGradient", SvgNamespace) }

// Marker creates an SVG <marker> element.
func Marker() INode { return current().Element("marker", SvgNamespace) }

// Mask creates an SVG <mask> element.
func Mask() INode { return current().Element("mask", SvgNamespace) }

// Metadata creates an SVG <metadata> element.
func Metadata() INode { return current().Element("metadata", SvgNamespace) }

// Mpath creates an SVG <mpath> element.
func Mpath() INode { return current().Element("mpath", SvgNamespace) }

// Pattern creates an SVG <pattern> element.
func Pattern() INode { return current().Element("pattern", SvgNamespace) }

// Polygon creates an SVG <polygon> element.
func Polygon() INode { return current().Element("polygon", SvgNamespace) }

// Polyline creates an SVG <polyline> element.
func Polyline() INode { return current().Element("polyline", SvgNamespace) }

// RadialGradient creates an SVG <radialGradient> element.
func RadialGradient() INode { return current().Element("radialGradient", SvgNamespace) }

// Rect creates an SVG <rect> element.
func Rect() INode { return current().Element("rect", SvgNamespace) }

// Set creates an SVG <set> element.
func Set() INode { return current().Element("set", SvgNamespace) }

// Stop creates an SVG <stop> element.
func Stop() INode { return current().Element("stop", SvgNamespace) }

// Switch creates an SVG <switch> element.
func Switch() INode { return current().Element("switch", SvgNamespace) }

// Symbol creates an SVG <symbol> element.
func Symbol() INode { return current().Element("symbol", SvgNamespace) }

// Text creates an SVG <text> element.
func Text() INode { return current().Element("text", SvgNamespace) }

// TextPath creates an SVG <textPath> element.
func TextPath() INode { return current().Element("textPath", SvgNamespace) }

// Tspan creates an SVG <tspan> element.
func Tspan() INode { return current().Element("tspan", SvgNamespace) }

// Use creates an SVG <use> element.
func Use() INode { return current().Element("use", SvgNamespace) }

// View creates an SVG <view> element.
func View() INode { return current().Element("view", SvgNamespace) }

// SvgA creates an SVG <a> element.
func SvgA() INode { return current().Element("a", SvgNamespace) }

// SvgTitle creates an SVG <title> element.
func SvgTitle() INode { return current().Element("title", SvgNamespace) }

// SvgStyle creates an SVG <style> element.
func SvgStyle() INode { return current().Element("style", SvgNamespace) }

// SvgScript creates an SVG <script> element.
func SvgScript() INode { return current().Element("script", SvgNamespace) }

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

// Fieldset creates a <fieldset> element.
func Fieldset() INode { return current().Element("FIELDSET", "") }

// Legend creates a <legend> element.
func Legend() INode { return current().Element("LEGEND", "") }

// Details creates a <details> element.
func Details() INode { return current().Element("DETAILS", "") }

// Summary creates a <summary> element.
func Summary() INode { return current().Element("SUMMARY", "") }

// Dialog creates a <dialog> element.
func Dialog() INode { return current().Element("DIALOG", "") }

// Figure creates a <figure> element.
func Figure() INode { return current().Element("FIGURE", "") }

// Figcaption creates a <figcaption> element.
func Figcaption() INode { return current().Element("FIGCAPTION", "") }

// Blockquote creates a <blockquote> element.
func Blockquote() INode { return current().Element("BLOCKQUOTE", "") }

// Dl creates a <dl> element.
func Dl() INode { return current().Element("DL", "") }

// Dt creates a <dt> element.
func Dt() INode { return current().Element("DT", "") }

// Dd creates a <dd> element.
func Dd() INode { return current().Element("DD", "") }

// Progress creates a <progress> element.
func Progress() INode { return current().Element("PROGRESS", "") }

// Meter creates a <meter> element.
func Meter() INode { return current().Element("METER", "") }

// Datalist creates a <datalist> element.
func Datalist() INode { return current().Element("DATALIST", "") }

// Optgroup creates a <optgroup> element.
func Optgroup() INode { return current().Element("OPTGROUP", "") }

// Output creates a <output> element.
func Output() INode { return current().Element("OUTPUT", "") }

// Iframe creates a <iframe> element.
func Iframe() INode { return current().Element("IFRAME", "") }

// Picture creates a <picture> element.
func Picture() INode { return current().Element("PICTURE", "") }

// Mark creates a <mark> element.
func Mark() INode { return current().Element("MARK", "") }

// Kbd creates a <kbd> element.
func Kbd() INode { return current().Element("KBD", "") }

// Time creates a <time> element.
func Time() INode { return current().Element("TIME", "") }

// Abbr creates a <abbr> element.
func Abbr() INode { return current().Element("ABBR", "") }

// B creates a <b> element.
func B() INode { return current().Element("B", "") }

// U creates a <u> element.
func U() INode { return current().Element("U", "") }

// Sub creates a <sub> element.
func Sub() INode { return current().Element("SUB", "") }

// Sup creates a <sup> element.
func Sup() INode { return current().Element("SUP", "") }

// Caption creates a <caption> element.
func Caption() INode { return current().Element("CAPTION", "") }

// Colgroup creates a <colgroup> element.
func Colgroup() INode { return current().Element("COLGROUP", "") }

// Col creates a <col> element.
func Col() INode { return current().Element("COL", "") }

// Hgroup creates a <hgroup> element.
func Hgroup() INode { return current().Element("HGROUP", "") }

// Html creates a <html> element.
// Only useful on the server and when hydrating: the browser already has its <html>.
func Html() INode { return current().Element("HTML", "") }

// Head creates a <head> element.
// Only useful on the server and when hydrating: the browser already has its <head>.
func Head() INode { return current().Element("HEAD", "") }

// Body creates a <body> element.
// Only useful on the server and when hydrating: the browser already has its <body>.
func Body() INode { return current().Element("BODY", "") }

// Title creates a <title> element.
func Title() INode { return current().Element("TITLE", "") }

// Meta creates a <meta> element.
func Meta() INode { return current().Element("META", "") }

// Link creates a <link> element.
func Link() INode { return current().Element("LINK", "") }

// Script creates a <script> element.
// A script made as a node runs in the browser; one inside Raw does not.
func Script() INode { return current().Element("SCRIPT", "") }

// Style creates a <style> element.
func Style() INode { return current().Element("STYLE", "") }

// Template creates a <template> element.
func Template() INode { return current().Element("TEMPLATE", "") }

// Address creates a <address> element.
func Address() INode { return current().Element("ADDRESS", "") }

// Cite creates a <cite> element.
func Cite() INode { return current().Element("CITE", "") }

// Q creates a <q> element.
func Q() INode { return current().Element("Q", "") }

// Samp creates a <samp> element.
func Samp() INode { return current().Element("SAMP", "") }

// Var creates a <var> element.
func Var() INode { return current().Element("VAR", "") }

// Del creates a <del> element.
func Del() INode { return current().Element("DEL", "") }

// Ins creates a <ins> element.
func Ins() INode { return current().Element("INS", "") }

// Wbr creates a <wbr> element.
func Wbr() INode { return current().Element("WBR", "") }

// Noscript creates a <noscript> element.
func Noscript() INode { return current().Element("NOSCRIPT", "") }

// SvgNamespace is the namespace of the SVG elements.
const SvgNamespace = "http://www.w3.org/2000/svg"
