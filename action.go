package hx

import (
	"context"
	"errors"
	"sync"
)

// ActionT is something the user triggers that takes a while and can fail:
// saving a form, deleting a record. It keeps the state for the UI: whether it
// is running, the last error and the last result. Create it with Action.
type ActionT[A, R any] struct {
	run func(ctx context.Context, arg A) (R, error)

	pending *SignalT[bool]
	err     *SignalT[error]
	value   *SignalT[R]

	mu         sync.Mutex
	generation int
	cancel     context.CancelFunc

	onSuccess func(result R, arg A)
	onError   func(err error, arg A)

	settled func() // called when a run ends, for the tests to wait on
}

// Action creates an action that runs fn when Run is called.
//
// fn runs on its own goroutine and gets a context that is canceled if the action
// is canceled or run again. It can wait for the network:
//
//	save := hx.Action(func(ctx context.Context, c Contact) (Saved, error) {
//		return hxjson.Post[Saved](ctx, "/api/contacts", c)
//	})
//	save.Run(contact)
//
// The state is in signals, so the UI follows it:
//
//	button.BindText(...save.Pending()...)
func Action[A, R any](fn func(ctx context.Context, arg A) (R, error)) *ActionT[A, R] {
	var zero R
	return &ActionT[A, R]{
		run:     fn,
		pending: Signal(false),
		err:     Signal[error](nil),
		value:   Signal(zero),
	}
}

// Run starts the action. If it was already running, that run is canceled and
// its result is ignored: the last call wins.
//
// The error is cleared when it starts; the last result is kept until there is a
// new one, so the UI can keep showing it while it reloads.
func (a *ActionT[A, R]) Run(arg A) {
	a.mu.Lock()
	a.generation++
	generation := a.generation
	if a.cancel != nil {
		a.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.mu.Unlock()

	Batch(func() {
		a.err.Set(nil)
		a.pending.Set(true)
	})

	go func() {
		defer func() {
			a.mu.Lock()
			settled := a.settled
			a.mu.Unlock()
			if settled != nil {
				settled()
			}
		}()
		result, err := a.run(ctx, arg)

		a.mu.Lock()
		current := generation == a.generation
		if current {
			a.cancel = nil
		}
		a.mu.Unlock()
		cancel()
		if !current {
			return // it was canceled or run again, nobody wants this result
		}

		// The state of the end of the run changes in one batch, so the UI updates
		// once and never sees "not running" without the answer. The callback
		// comes after, when the UI has already reacted, and it is a batch of its own.
		if err != nil {
			Batch(func() {
				a.err.Set(err)
				a.pending.Set(false)
			})
			if a.onError != nil {
				Batch(func() { a.onError(err, arg) })
			}
			return
		}
		Batch(func() {
			a.value.Set(result)
			a.pending.Set(false)
		})
		if a.onSuccess != nil {
			Batch(func() { a.onSuccess(result, arg) })
		}
	}()
}

// Cancel stops a run that is in progress. Its result is ignored and the action
// is not pending anymore. It does nothing if the action is not running.
func (a *ActionT[A, R]) Cancel() {
	a.mu.Lock()
	running := a.cancel != nil
	if running {
		a.generation++
		a.cancel()
		a.cancel = nil
	}
	a.mu.Unlock()
	if running {
		a.pending.Set(false)
	}
}

// Reset cancels a run in progress and forgets the error and the result.
func (a *ActionT[A, R]) Reset() {
	a.Cancel()
	var zero R
	Batch(func() {
		a.err.Set(nil)
		a.value.Set(zero)
	})
}

// OnSuccess sets a function that is called, after the state has been updated,
// every time a run finishes without error. It runs on the goroutine of the action.
func (a *ActionT[A, R]) OnSuccess(fn func(result R, arg A)) *ActionT[A, R] {
	a.onSuccess = fn
	return a
}

// OnError sets a function that is called, after the state has been updated,
// every time a run fails. A run that was canceled does not call it.
func (a *ActionT[A, R]) OnError(fn func(err error, arg A)) *ActionT[A, R] {
	a.onError = fn
	return a
}

func (a *ActionT[A, R]) setSettled(fn func()) {
	a.mu.Lock()
	a.settled = fn
	a.mu.Unlock()
}

// Pending tells if the action is running.
func (a *ActionT[A, R]) Pending() Gettable[bool] { return a.pending }

// Error is the error of the last run, or nil.
func (a *ActionT[A, R]) Error() Gettable[error] { return a.err }

// Value is the result of the last run that succeeded.
func (a *ActionT[A, R]) Value() Gettable[R] { return a.value }

// IsCanceled tells if err comes from a canceled request.
func IsCanceled(err error) bool { return errors.Is(err, context.Canceled) }

// IsTimeout tells if err comes from a request that took too long.
func IsTimeout(err error) bool { return errors.Is(err, context.DeadlineExceeded) }
