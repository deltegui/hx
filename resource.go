package hx

import "context"

// ResourceT is data that comes from a server. It loads when the UI is in the
// page and keeps the state in signals. Create it with Resource, ResourceWith or ResourceFor.
type ResourceT[T any] struct {
	value   *SignalT[T]
	loading *SignalT[bool]
	err     *SignalT[error]
	refetch func()
	cancel  func()
	settled func(func()) // sets the function the action calls when a load ends, for the tests
}

// Resource loads data with fetch as soon as the UI is in the browser.
//
// While the UI is rendered on a server nothing is loaded and the resource has
// no value; use ResourceWith to give it one that comes from the server.
//
//	items := hx.Resource(func(ctx context.Context) ([]Item, error) {
//		return hxjson.Get[[]Item](ctx, "/api/items")
//	})
//	// items.Loading(), items.Error(), items.Value() are signals
//	// items.Refetch() loads again
func Resource[T any](fetch func(ctx context.Context) (T, error)) *ResourceT[T] {
	r := newResource(fetch)
	OnMount(r.Refetch)
	return r
}

// ResourceWith is Resource with a value to start from (usually the one the
// server rendered, so the first paint already has it). It does not load until
// Refetch is called.
func ResourceWith[T any](initial T, fetch func(ctx context.Context) (T, error)) *ResourceT[T] {
	r := newResource(fetch)
	r.value.Set(initial)
	return r
}

// ResourceFor loads data that depends on source. It loads when the UI is in the
// browser and again each time source changes; a load that is still running is
// canceled by the next one.
//
//	results := hx.ResourceFor(query, func(ctx context.Context, q string) ([]Item, error) {
//		return hxjson.Get[[]Item](ctx, "/api/search?q="+url.QueryEscape(q))
//	})
func ResourceFor[S, T any](source Gettable[S], fetch func(ctx context.Context, source S) (T, error)) *ResourceT[T] {
	action := Action(fetch)
	r := &ResourceT[T]{
		value:   action.value,
		loading: action.pending,
		err:     action.err,
		cancel:  action.Cancel,
		settled: action.setSettled,
	}
	r.refetch = func() { action.Run(UntrackGet(source)) }
	OnMount(func() {
		EffectFunc(func() { action.Run(source.Get()) })
	})
	return r
}

func newResource[T any](fetch func(ctx context.Context) (T, error)) *ResourceT[T] {
	action := Action(func(ctx context.Context, _ struct{}) (T, error) { return fetch(ctx) })
	return &ResourceT[T]{
		value:   action.value,
		loading: action.pending,
		err:     action.err,
		refetch: func() { action.Run(struct{}{}) },
		cancel:  action.Cancel,
		settled: action.setSettled,
	}
}

// Value is the last data that was loaded.
func (r *ResourceT[T]) Value() Gettable[T] { return r.value }

// Loading tells if a load is in progress.
func (r *ResourceT[T]) Loading() Gettable[bool] { return r.loading }

// Error is the error of the last load, or nil.
func (r *ResourceT[T]) Error() Gettable[error] { return r.err }

// Refetch loads the data again. The current value stays until the new one arrives.
func (r *ResourceT[T]) Refetch() { r.refetch() }

// Cancel stops the load that is in progress.
func (r *ResourceT[T]) Cancel() { r.cancel() }

// Set replaces the value without loading, for example after an action changed the data.
func (r *ResourceT[T]) Set(v T) { r.value.Set(v) }
