package hx

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
