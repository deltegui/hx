package hx

import (
	"cmp"
	"sort"
)

// discard frees a node that was built but is not going to be used, so its
// bindings and effects do not live until the effect that built it re-runs.
// If the node ends up in the tree later it is activated again.
func discard(node INode) {
	if node != nil {
		node.AsVNode().deactivate()
	}
}

// If returns child when the condition is true, otherwise nil. The child is
// already built, so when the condition is false it is discarded.
func If(condition Gettable[bool], child INode) INode {
	if condition.Get() {
		return child
	}
	discard(child)
	return nil
}

// Show returns ifPath when the condition is true, otherwise elsePath. The other
// path is already built, so it is discarded.
func Show(condition Gettable[bool], ifPath, elsePath INode) INode {
	if condition.Get() {
		discard(elsePath)
		return ifPath
	}
	discard(ifPath)
	return elsePath
}

// IfFn is like If, but the child is only built when the condition is true.
// Prefer it when building the child is expensive or has side effects.
func IfFn(condition Gettable[bool], build func() INode) INode {
	if condition.Get() {
		return build()
	}
	return nil
}

// ShowFn is like Show, but only the path that is going to be used is built.
// elsePath can be nil.
func ShowFn(condition Gettable[bool], ifPath, elsePath func() INode) INode {
	if condition.Get() {
		return ifPath()
	}
	if elsePath == nil {
		return nil
	}
	return elsePath()
}

// Each renders a node for every item of the slice. The list is rebuilt when the signal changes, so call it inside an effect, for example list.Effect(func() { list.Body(hx.Each(items, render)) }).
func Each[T any](src Gettable[[]T], renderOne func(index int, value T) INode) INode {
	list := src.Get()
	result := make([]INode, len(list))
	for i, element := range list {
		result[i] = renderOne(i, element)
	}
	return Noop().Body(result...)
}

// EachMap renders every entry of the map. The order of the entries is not
// defined, use EachMapSorted if it matters.
func EachMap[K comparable, T any](src Gettable[map[K]T], renderOne func(index K, value T) INode) INode {
	list := src.Get()
	result := make([]INode, len(list))
	i := 0
	for key, element := range list {
		result[i] = renderOne(key, element)
		i++
	}
	return Noop().Body(result...)
}

// EachMapSorted is like EachMap but renders the entries ordered by key.
func EachMapSorted[K cmp.Ordered, T any](src Gettable[map[K]T], renderOne func(index K, value T) INode) INode {
	list := src.Get()
	keys := make([]K, 0, len(list))
	for key := range list {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	result := make([]INode, len(keys))
	for i, key := range keys {
		result[i] = renderOne(key, list[key])
	}
	return Noop().Body(result...)
}
