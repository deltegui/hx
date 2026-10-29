package hx

import "sort"

// ValueWrapper holds a plain value that satisfies Gettable and Settable. It is not reactive.
type ValueWrapper[T any] struct {
	inner T
}

// Value wraps a plain value so it can be used where a Gettable is expected, for example hx.If(hx.Value(true), node).
func Value[T any](t T) ValueWrapper[T] {
	return ValueWrapper[T]{
		inner: t,
	}
}

// Get returns the wrapped value.
func (v ValueWrapper[T]) Get() T {
	return v.inner
}

// Set replaces the wrapped value.
func (v *ValueWrapper[T]) Set(t T) {
	v.inner = t
}

type diffValue[T comparable] struct {
	value     T
	nextValue T
	status    changeStatus
}

func (sv diffValue[T]) Value() T {
	return sv.nextValue
}
func (sv diffValue[T]) Status() changeStatus {
	return sv.status
}

func (sv *diffValue[T]) reset() {
	sv.nextValue = sv.value
	sv.status = unchanged
}

func (sv *diffValue[T]) assign(val T, status changeStatus) bool {
	if val == sv.value {
		sv.reset()
		return false
	}
	if val == sv.nextValue && sv.status != changeDeleted {
		return true
	}
	sv.nextValue = val
	sv.status = status
	return true
}

func (sv *diffValue[T]) equals(val T) bool {
	return sv.nextValue == val || sv.value == val
}

func (sv *diffValue[T]) tick() {
	sv.value = sv.nextValue
	sv.status = unchanged
}

// setEntry assigns value to key, keeping track of what changed since the last
// render. It returns false if nothing changed.
func setEntry(entries map[string]diffValue[string], key, value string) bool {
	current, ok := entries[key]
	switch {
	case !ok:
		entries[key] = diffValue[string]{nextValue: value, status: changeNew}
	case current.status == changeNew:
		current.nextValue = value
		entries[key] = current
	case current.status != changeDeleted && current.nextValue == value:
		return false
	default:
		current.assign(value, changeModified)
		entries[key] = current
	}
	return true
}

// removeEntry marks key as deleted. It returns false if there was nothing to remove.
func removeEntry(entries map[string]diffValue[string], key string) bool {
	current, ok := entries[key]
	if !ok || current.status == changeDeleted {
		return false
	}
	if current.status == changeNew {
		// It was never rendered, so there is nothing to remove from the DOM.
		delete(entries, key)
		return true
	}
	current.status = changeDeleted
	entries[key] = current
	return true
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
