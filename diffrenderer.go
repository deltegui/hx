package hx

import (
	"log"
	"strings"
	"syscall/js"

	"honnef.co/go/js/dom/v2"
)

type DiffRenderer struct {
	mountpoint  dom.Element
	markNodes   map[*VNode]struct{}
	scheduled   bool
	rafCallback js.Func
}

func newRenderer(element dom.Element) *DiffRenderer {
	r := &DiffRenderer{
		mountpoint: element,
		markNodes:  map[*VNode]struct{}{},
		scheduled:  false,
	}
	r.createRaf()
	return r
}

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

func (r *DiffRenderer) ScheduleRender() {
	if !r.scheduled {
		r.scheduled = true
		js.Global().Call("requestAnimationFrame", r.rafCallback)
	}
}

func (renderer *DiffRenderer) render() {
	rootLCA := renderer.GetMarkedCommonAncestor()
	if rootLCA != nil {
		renderer.syncNodes(rootLCA, nil)
	}
	for node := range renderer.markNodes {
		node.render()
		delete(renderer.markNodes, node)
	}
}

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

// firstAttachedDom returns the first DOM node that node renders and that is
// already in the DOM, or nil.
func firstAttachedDom(node *VNode) dom.Node {
	if node == nil || node.status == changeNew || node.status == changeDeleted {
		return nil
	}
	if node.tag != noopIdNode {
		if node.haveDomElement {
			return node.domElement
		}
		return nil
	}
	for _, child := range node.children {
		if first := firstAttachedDom(child); first != nil {
			return first
		}
	}
	return nil
}

// nextAttachedDom returns the DOM node that follows everything node renders,
// inside the DOM parent of node. It returns nil if node is the last one.
func nextAttachedDom(node *VNode) dom.Node {
	for current := node; current.father != nil; current = current.father {
		parent := current.father
		found := false
		for _, sibling := range parent.children {
			if sibling == current {
				found = true
				continue
			}
			if !found {
				continue
			}
			if first := firstAttachedDom(sibling); first != nil {
				return first
			}
		}
		if parent.haveDomElement {
			return nil
		}
	}
	return nil
}

func insertDom(parent *VNode, child dom.Node, anchor dom.Node) {
	if parent == nil || !parent.haveDomElement {
		return
	}
	if anchor == nil {
		parent.domElement.AppendChild(child)
		return
	}
	parent.domElement.InsertBefore(child, anchor)
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
	if parent := node.domElement.ParentNode(); parent != nil {
		parent.RemoveChild(node.domElement)
	}
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

func (renderer *DiffRenderer) syncChildren(element *VNode, anchor dom.Node) {
	childs := make([]*VNode, 0, len(element.children))
	for _, child := range element.children {
		if child == nil {
			continue
		}
		if renderer.syncNodes(child, anchor) {
			childs = append(childs, child)
		}
	}
	element.children = childs
}

// syncNoopNode syncs a Noop. A Noop renders nothing by itself, its children are
// inserted in its DOM parent before anchor (or at the end if anchor is nil).
func (renderer *DiffRenderer) syncNoopNode(element *VNode, anchor dom.Node) bool {
	if element.status == changeDeleted {
		renderer.detach(element)
		return false
	}
	if element.status == unchanged {
		anchor = nextAttachedDom(element)
	}
	if element.status == changeNew {
		element.status = unchanged
	}
	renderer.syncChildren(element, anchor)
	return true
}

func (renderer *DiffRenderer) createDomElement(element *VNode) {
	if element.haveDomElement || len(element.tag) == 0 {
		return
	}
	document := dom.GetWindow().Document()
	if element.namespace != "" {
		element.domElement = document.CreateElementNS(element.namespace, element.tag)
	} else {
		element.domElement = document.CreateElement(element.tag)
	}
	element.haveDomElement = true
}

func (renderer *DiffRenderer) syncNodes(element *VNode, anchor dom.Node) bool {
	if element.tag == noopIdNode {
		return renderer.syncNoopNode(element, anchor)
	}

	if element.status == changeDeleted {
		renderer.detach(element)
		return false
	}
	if element.status == changeNew {
		renderer.createDomElement(element)
		element.status = unchanged
		element.reactivate()
		if element.haveDomElement {
			insertDom(domParentOf(element), element.domElement, anchor)
		}
	}

	renderer.syncChildren(element, nil)

	return true
}

func (element *VNode) render() {
	if element.domElement == nil {
		return
	}

	if element.text.status != unchanged {
		element.updateText()
	}
	if element.value.status != unchanged {
		element.updateValue()
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
			Untrack(func() {
				listener.handler(EventContext{
					Target: element.Owner,
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
	if element.textNode == nil {
		element.textNode = dom.GetWindow().Document().CreateTextNode(text)
		element.domElement.InsertBefore(element.textNode, element.domElement.FirstChild())
	} else {
		element.textNode.SetTextContent(text)
	}
	element.text.tick()
}

func (element *VNode) updateValue() {
	jsVal := element.domElement.Underlying()
	val := element.value.Value()
	jsVal.Set("value", val)
	element.value.tick()
}
