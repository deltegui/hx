//go:build js && wasm

package wasm

import (
	"github.com/deltegui/hx"
	"log"
	"strings"
	"syscall/js"

	"honnef.co/go/js/dom/v2"
)

// DiffRenderer applies the changes of the nodes to the browser DOM. It batches them and runs once per animation frame.
type DiffRenderer struct {
	mountpoint  dom.Element
	markNodes   map[*VNode]struct{}
	toPlace     map[*VNode]struct{} // DOM owners whose children must be put in order
	selects     []*VNode            // selects that get their value when the rest of the render is done
	rendered    []*VNode            // nodes whose OnRendered runs when the frame is done
	scheduled   bool
	rafCallback js.Func
}

func newRenderer(element dom.Element) *DiffRenderer {
	r := &DiffRenderer{
		mountpoint: element,
		markNodes:  map[*VNode]struct{}{},
		toPlace:    map[*VNode]struct{}{},
		scheduled:  false,
	}
	r.createRaf()
	return r
}

// Mark records that the node has changes to render.
func (r *DiffRenderer) Mark(element *VNode) {
	r.markNodes[element] = struct{}{}
}

func (r *DiffRenderer) createRaf() {
	r.rafCallback = js.FuncOf(func(this js.Value, args []js.Value) any {
		defer func() { r.scheduled = false }()
		r.render()
		return nil
	})
}

// ScheduleRender asks for a render in the next animation frame. Several calls in the same frame render once.
func (r *DiffRenderer) ScheduleRender() {
	if !r.scheduled {
		r.scheduled = true
		js.Global().Call("requestAnimationFrame", r.rafCallback)
	}
}

func (renderer *DiffRenderer) render() {
	rootLCA := renderer.GetMarkedCommonAncestor()
	if rootLCA != nil {
		renderer.syncNodes(rootLCA)
		renderer.placeAll()
	}
	for node := range renderer.markNodes {
		node.render()
		delete(renderer.markNodes, node)
	}
	// A select takes its value once its options have theirs, and the order the
	// nodes are rendered in does not guarantee that.
	selects := renderer.selects
	renderer.selects = nil
	for _, node := range selects {
		if node.haveDomElement {
			node.updateValue()
		}
	}
	renderer.flushRendered()
}

// flushRendered runs the OnRendered functions of the nodes that were rendered
// and are still in the page.
func (renderer *DiffRenderer) flushRendered() {
	queue := renderer.rendered
	renderer.rendered = nil
	for _, node := range queue {
		if node.attached || node.deactivated || !node.haveDomElement {
			continue
		}
		node.attached = true
		for _, fn := range node.onRendered {
			hx.Untrack(fn)
		}
	}
}

// GetMarkedCommonAncestor returns the lowest node that contains every marked node, or nil if none is marked.
func (renderer *DiffRenderer) GetMarkedCommonAncestor() *VNode {
	if len(renderer.markNodes) == 0 {
		return nil
	}

	var path []*VNode
	isFirstMark := true

	for markedNode := range renderer.markNodes {
		if isFirstMark {
			path = renderer.generatePathToRoot(markedNode)
			isFirstMark = false
			continue
		}

		path = renderer.generateCommonPathToRoot(path, markedNode)
	}
	return path[0]
}

func (renderer *DiffRenderer) generatePathToRoot(source *VNode) []*VNode {
	path := []*VNode{source}
	parent := source.father
	for parent != nil {
		path = append(path, parent)
		parent = parent.father
	}
	return path
}

// generateCommonPathToRoot returns the part of currentPath that starts at the
// lowest common ancestor of the path and markedNode. markedNode itself can be
// that ancestor.
func (renderer *DiffRenderer) generateCommonPathToRoot(currentPath []*VNode, markedNode *VNode) []*VNode {
	for node := markedNode; node != nil; node = node.father {
		for pathNodeIndex, pathNode := range currentPath {
			if pathNode == node {
				return currentPath[pathNodeIndex:]
			}
		}
	}
	return currentPath
}

// domParentOf returns the closest ancestor that owns a DOM element.
// Noop nodes do not own one, so they are skipped.
func domParentOf(element *VNode) *VNode {
	for parent := element.father; parent != nil; parent = parent.father {
		if parent.haveDomElement {
			return parent
		}
	}
	return nil
}

// collectTops returns the DOM elements that the live children of node put
// directly inside the DOM element of its owner. Noop children are transparent.
func collectTops(node *VNode, tops []dom.Element) []dom.Element {
	for _, child := range node.children {
		if child == nil {
			continue
		}
		if child.tag == noopIdNode {
			tops = collectTops(child, tops)
		} else if child.haveDomElement {
			tops = append(tops, child.domElement)
		}
	}
	return tops
}

// longestIncreasing returns, for the sequence seq, which of its values belong
// to a longest strictly increasing subsequence. The values must be smaller than size.
func longestIncreasing(seq []int, size int) []bool {
	keep := make([]bool, size)
	if len(seq) == 0 {
		return keep
	}
	tails := make([]int, 0, len(seq)) // index in seq of the smallest tail of each length
	prev := make([]int, len(seq))
	for i, v := range seq {
		lo, hi := 0, len(tails)
		for lo < hi {
			mid := (lo + hi) / 2
			if seq[tails[mid]] < v {
				lo = mid + 1
			} else {
				hi = mid
			}
		}
		if lo > 0 {
			prev[i] = tails[lo-1]
		} else {
			prev[i] = -1
		}
		if lo == len(tails) {
			tails = append(tails, i)
		} else {
			tails[lo] = i
		}
	}
	for i := tails[len(tails)-1]; i >= 0; i = prev[i] {
		keep[seq[i]] = true
	}
	return keep
}

// positionKey is the property used to read back the order of the elements in the DOM.
const positionKey = "__hxpos"

// placeChildren puts the DOM elements of the children of owner in the order of
// the tree while moving as few as possible. The elements that are already in
// the right relative order (a longest increasing subsequence of their DOM
// positions) are not touched, so they keep focus, scroll and selection; the
// others are inserted before the next element.
func placeChildren(owner *VNode) {
	tops := collectTops(owner, nil)
	if len(tops) == 0 {
		return
	}
	parent := owner.domElement.Underlying()

	raws := make([]js.Value, len(tops))
	for i, top := range tops {
		raws[i] = top.Underlying()
		raws[i].Set(positionKey, i)
	}

	// Order in which the elements of the tree currently are in the DOM.
	inDom := make([]int, 0, len(tops))
	for child := parent.Get("firstChild"); !child.IsNull(); child = child.Get("nextSibling") {
		if position := child.Get(positionKey); position.Type() == js.TypeNumber {
			inDom = append(inDom, position.Int())
		}
	}
	stay := longestIncreasing(inDom, len(tops))

	next := js.Null()
	for i := len(raws) - 1; i >= 0; i-- {
		if !stay[i] {
			parent.Call("insertBefore", raws[i], next)
		}
		next = raws[i]
	}
}

func (renderer *DiffRenderer) placeAll() {
	for owner := range renderer.toPlace {
		placeChildren(owner)
		delete(renderer.toPlace, owner)
	}
}

// removeDom removes from the DOM everything that node renders.
func removeDom(node *VNode) {
	if node.tag == noopIdNode {
		for _, child := range node.children {
			if child != nil {
				removeDom(child)
			}
		}
		return
	}
	if !node.haveDomElement {
		return
	}
	removeFromParent(node.domElement)
}

// detach removes a deleted node from the DOM and frees its resources.
func (renderer *DiffRenderer) detach(element *VNode) {
	if element.father == nil {
		return
	}
	removeDom(element)
	element.deactivate()
	delete(renderer.markNodes, element)
	element.father = nil
}

func (renderer *DiffRenderer) syncChildren(element *VNode) {
	childs := make([]*VNode, 0, len(element.children))
	for _, child := range element.children {
		if child == nil {
			continue
		}
		if renderer.syncNodes(child) {
			childs = append(childs, child)
		}
	}
	element.children = childs
}

// syncNoopNode syncs a Noop. A Noop renders nothing by itself, so when its
// children change the DOM owner that holds them has to put them in order.
func (renderer *DiffRenderer) syncNoopNode(element *VNode) bool {
	if element.status == changeDeleted {
		renderer.detach(element)
		return false
	}
	element.status = unchanged
	renderer.syncChildren(element)
	if element.isDirty(flagChildren) {
		element.dirtyFlags[flagChildren] = false // a Noop is never rendered by itself
		if owner := domParentOf(element); owner != nil && len(element.children) > 0 {
			renderer.toPlace[owner] = struct{}{}
		}
	}
	return true
}

func (renderer *DiffRenderer) createDomElement(element *VNode) {
	if element.haveDomElement || len(element.tag) == 0 {
		return
	}
	element.domElement = newDomElement(element.tag, element.namespace)
	element.haveDomElement = true
}

// syncNodes creates the DOM elements of new nodes, removes the ones of deleted
// nodes, and records which DOM owners have to put their children in order. It
// returns false if the node was deleted.
func (renderer *DiffRenderer) syncNodes(element *VNode) bool {
	if element.tag == noopIdNode {
		return renderer.syncNoopNode(element)
	}

	if element.status == changeDeleted {
		renderer.detach(element)
		return false
	}
	if element.status == changeNew {
		renderer.createDomElement(element)
		element.status = unchanged
		element.reactivate()
	}

	renderer.syncChildren(element)
	if element.isDirty(flagChildren) && element.haveDomElement && len(element.children) > 0 {
		renderer.toPlace[element] = struct{}{}
	}
	return true
}

func (element *VNode) render() {
	if element.domElement == nil {
		return
	}

	if element.text.status != unchanged {
		element.updateText()
	}
	if element.tag == "SELECT" {
		element.deferSelectValue()
	} else if element.value.status != unchanged {
		element.updateValue()
	}
	if element.checked.status != unchanged {
		element.updateChecked()
	}
	if element.isDirty(flagEventListeners) {
		element.updateEventListeners()
	}
	if element.isDirty(flagClasses) {
		element.updateClasses()
	}
	if element.isDirty(flagAttributes) {
		element.updateAttributes()
	}
	if element.isDirty(flagStyles) {
		element.updateStyles()
	}
	if element.id.status != unchanged {
		element.updateId()
	}

	if element.isDirty(flagChildren) {
		element.updateChildren()
	}

	element.clearDirty()
	element.queueRendered()
}

func (element *VNode) updateId() {
	element.domElement.SetID(element.id.Value())
	element.id.status = unchanged
}

func (element *VNode) updateAttributes() {
	for _, attribute := range sortedKeys(element.attributes) {
		value := element.attributes[attribute]
		switch value.status {
		case changeModified, changeNew:
			element.domElement.SetAttribute(attribute, value.Value())
		case changeDeleted:
			element.domElement.RemoveAttribute(attribute)
			delete(element.attributes, attribute)
			continue
		default:
			continue
		}
		value.tick()
		element.attributes[attribute] = value
	}
}

func (element *VNode) updateStyles() {
	stylesString := strings.Builder{}
	for _, style := range sortedKeys(element.styles) {
		value := element.styles[style]
		if value.status == changeDeleted {
			delete(element.styles, style)
			continue
		}
		stylesString.WriteString(style)
		stylesString.WriteString(":")
		stylesString.WriteString(value.Value())
		stylesString.WriteString("; ")
		value.tick()
		element.styles[style] = value
	}
	if stylesString.Len() > 0 {
		element.domElement.SetAttribute("style", stylesString.String())
	} else {
		element.domElement.RemoveAttribute("style")
	}
}

func (element *VNode) updateClasses() {
	for _, class := range sortedKeys(element.classes) {
		status := element.classes[class]
		if strings.Contains(class, " ") {
			log.Printf("Warning: ignoring class with spaces: %s", class)
			continue
		}
		switch status {
		case changeDeleted:
			element.domElement.Class().Remove(class)
			delete(element.classes, class)
		case changeNew:
			element.domElement.Class().Add(class)
			element.classes[class] = unchanged
		default:
			element.classes[class] = unchanged
		}
	}
}

func (element *VNode) updateEventListeners() {
	for event, listener := range element.eventListeners {
		if listener.attached {
			continue
		}
		listener.attached = true
		listener.wrapper = element.domElement.AddEventListener(string(event), false, func(e dom.Event) {
			hx.Untrack(func() {
				listener.handler(hx.EventContext{
					Target: element,
					Event:  e,
				})
			})
			element.scheludeRender()
		})
	}
}

// releaseListeners removes the DOM listeners of element, freeing the js
// functions that back them.
func (element *VNode) releaseListeners() {
	for event, listener := range element.eventListeners {
		if !listener.attached {
			continue
		}
		listener.attached = false
		if element.haveDomElement {
			element.domElement.RemoveEventListener(string(event), false, listener.wrapper)
		}
	}
}

// reviveListeners schedules the registration of the listeners that
// releaseListeners removed.
func (element *VNode) reviveListeners() {
	for _, listener := range element.eventListeners {
		if !listener.attached {
			element.setDirty(flagEventListeners)
			break
		}
	}
}

func (element *VNode) updateChildren() {
	for index, child := range element.children {
		if child == nil {
			continue
		}
		element.children[index].render()
	}
}

// updateText keeps the text in its own text node, so changing it does not
// remove the DOM nodes of the children.
func (element *VNode) updateText() {
	text := element.text.Value()
	if element.textNode.IsUndefined() {
		element.textNode = prependText(element.domElement, text)
	} else {
		element.textNode.Set("nodeValue", text)
	}
	element.text.tick()
}

// deferSelectValue leaves the value of a select for the end of the render. It is
// written again when the options change, because the old value may now match.
func (element *VNode) deferSelectValue() {
	changed := element.value.status != unchanged
	if !changed && (!element.isDirty(flagChildren) || element.value.Value() == "") {
		return
	}
	if renderer, ok := element.renderer.(*DiffRenderer); ok {
		renderer.selects = append(renderer.selects, element)
	} else {
		element.updateValue()
	}
}

func (element *VNode) updateChecked() {
	jsVal := element.domElement.Underlying()
	on := element.checked.Value()
	if jsVal.Get("checked").Truthy() != on {
		jsVal.Set("checked", on)
	}
	element.checked.tick()
}

func (element *VNode) updateValue() {
	jsVal := element.domElement.Underlying()
	val := element.value.Value()
	if jsVal.Get("value").String() != val {
		jsVal.Set("value", val)
	}
	element.value.tick()
}
