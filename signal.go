package hx

import (
	"log"
	"runtime"
	"sync"
)

var (
	currentEffect *Effect
	currentRoot   *Root
	untrack       bool
	mu            sync.Mutex
)

func accessEffect(action func(*Effect)) {
	mu.Lock()
	defer mu.Unlock()
	action(currentEffect)
}

// subscribe registers the current effect as a subscriber of subscribers.
// The subscription is removed when the effect cleans up.
func subscribe(subscribers map[*Effect]struct{}) {
	accessEffect(func(currentEffect *Effect) {
		if currentEffect == nil {
			if !untrack {
				_, file, line, ok := runtime.Caller(4)
				if ok {
					log.Printf("Cannot call Get if there is no effect (%s:%d)", file, line)
				} else {
					log.Printf("Cannot call Get if there is no effect")
				}
			}
			return
		}
		if _, ok := subscribers[currentEffect]; ok {
			return
		}
		subscribers[currentEffect] = struct{}{}
		currentEffect.cleanUps = append(currentEffect.cleanUps, func() {
			delete(subscribers, currentEffect)
		})
	})
}

// notify schedules every subscriber. Subscribers are copied first because
// running an effect unsubscribes and subscribes it again.
func notify(subscribers map[*Effect]struct{}) {
	if len(subscribers) == 0 {
		return
	}
	effects := make([]*Effect, 0, len(subscribers))
	for effect := range subscribers {
		effects = append(effects, effect)
	}
	for _, effect := range effects {
		effect.schedule()
	}
}

// SignalT is a reactive value. Create it with Signal.
type SignalT[T any] struct {
	value       T
	subscribers map[*Effect]struct{}
}

// Signal creates a reactive value. Reading it with Get inside an effect makes the effect run again every time the value changes.
func Signal[T any](initial T) *SignalT[T] {
	return &SignalT[T]{
		value:       initial,
		subscribers: make(map[*Effect]struct{}),
	}
}

// Get returns the value. Inside an effect (EffectFunc, node.Effect, Bind*, Computed) it subscribes the effect to the signal.
// Outside of an effect it just returns the value, and logs a warning unless it runs inside Untrack or an event handler.
func (signal *SignalT[T]) Get() T {
	subscribe(signal.subscribers)
	return signal.value
}

// Set changes the value and runs, right away, every effect that read the signal.
// It always notifies, even if the new value equals the old one.
func (signal *SignalT[T]) Set(v T) {
	signal.value = v
	signal.notify()
}

// Update sets the value to fn(current value). It does not subscribe the current effect.
func (signal *SignalT[T]) Update(fn func(T) T) {
	signal.Set(fn(signal.value))
}

func (signal *SignalT[T]) notify() {
	notify(signal.subscribers)
}

// Effect is a function that runs again when any signal it read changes.
type Effect struct {
	fn          func()
	isScheduled bool
	disposed    bool
	childs      []*Effect
	cleanUps    []func()
}

// EffectFunc runs fn now and again each time a signal read by fn changes.
// An effect created inside another effect is its child and is disposed when the parent runs again.
// An effect created inside CreateRoot is owned by the root. Any other effect lives forever unless you call Dispose.
// For effects tied to a node use VNode.Effect instead.
func EffectFunc(fn func()) *Effect {
	e := &Effect{
		fn:       fn,
		childs:   make([]*Effect, 0),
		cleanUps: make([]func(), 0),
	}
	mu.Lock()
	if currentEffect != nil {
		currentEffect.childs = append(currentEffect.childs, e)
	} else if currentRoot != nil {
		currentRoot.effects = append(currentRoot.effects, e)
	}
	mu.Unlock()
	e.run()
	return e
}

func (e *Effect) run() {
	mu.Lock()
	prev := currentEffect
	currentEffect = e
	mu.Unlock()

	defer func() {
		mu.Lock()
		currentEffect = prev
		mu.Unlock()
	}()

	e.clean()
	e.fn()
}

// clean disposes the child effects and removes the subscriptions
// so the effect is ready to run again.
func (e *Effect) clean() {
	for _, child := range e.childs {
		child.Dispose()
	}
	e.childs = []*Effect{}

	for _, cleanfn := range e.cleanUps {
		cleanfn()
	}
	e.cleanUps = make([]func(), 0)
}

// Dispose stops the effect for good: it is unsubscribed from every
// signal and it will never run again. Child effects are disposed too.
func (e *Effect) Dispose() {
	e.disposed = true
	e.clean()
}

func (e *Effect) schedule() {
	if e.isScheduled || e.disposed {
		return
	}
	e.isScheduled = true
	defer func() { e.isScheduled = false }()
	e.run()
}

// Root owns the effects that are created inside CreateRoot, outside of any
// other effect. Disposing the root disposes all of them.
type Root struct {
	effects  []*Effect
	cleanUps []func()
	disposed bool
}

// CreateRoot runs fn in a new Root. Effects created by fn that do not belong to
// another effect are owned by the root and live until dispose is called.
// Effects created outside a root and outside another effect are never disposed
// automatically.
func CreateRoot[T any](fn func(dispose func()) T) T {
	root := &Root{}

	mu.Lock()
	prev := currentRoot
	currentRoot = root
	mu.Unlock()

	defer func() {
		mu.Lock()
		currentRoot = prev
		mu.Unlock()
	}()

	return fn(root.Dispose)
}

// Dispose disposes every effect of the root and runs its cleanups.
// Calling it more than once does nothing.
func (r *Root) Dispose() {
	if r.disposed {
		return
	}
	r.disposed = true
	for _, effect := range r.effects {
		effect.Dispose()
	}
	r.effects = nil
	for _, cleanfn := range r.cleanUps {
		cleanfn()
	}
	r.cleanUps = nil
}

// OnCleanup registers fn to run the next time the current effect re-runs or is
// disposed, or when the current root is disposed. Outside of both it does nothing.
func OnCleanup(fn func()) {
	mu.Lock()
	defer mu.Unlock()
	if currentEffect != nil {
		currentEffect.cleanUps = append(currentEffect.cleanUps, fn)
	} else if currentRoot != nil {
		currentRoot.cleanUps = append(currentRoot.cleanUps, fn)
	}
}

// ComputedT is a value derived from signals. Create it with Computed.
type ComputedT[T comparable] struct {
	value           T
	dependentEffect *Effect
	subscribers     map[*Effect]struct{}
}

// Computed creates a value derived from other signals or computeds. It is recalculated when its dependencies change and only notifies its own subscribers when the result is different.
func Computed[T comparable](fn func() T) *ComputedT[T] {
	c := &ComputedT[T]{
		subscribers: make(map[*Effect]struct{}),
	}

	c.dependentEffect = EffectFunc(func() {
		newVal := fn()
		if newVal != c.value {
			c.value = newVal
			c.notify()
		}
	})

	return c
}

// Get returns the current value and subscribes the current effect, like Signal.Get.
func (c *ComputedT[T]) Get() T {
	subscribe(c.subscribers)
	return c.value
}

func (c *ComputedT[T]) notify() {
	notify(c.subscribers)
}

// Untrack runs fn without subscribing the current effect to the signals fn reads.
func Untrack(fn func()) {
	mu.Lock()
	prev := currentEffect
	prevUntrack := untrack
	untrack = true
	currentEffect = nil
	mu.Unlock()

	defer func() {
		mu.Lock()
		currentEffect = prev
		untrack = prevUntrack
		mu.Unlock()
	}()

	fn()
}

// UntrackGet reads a signal without subscribing the current effect.
func UntrackGet[T any](gettable Gettable[T]) T {
	var value T
	Untrack(func() {
		value = gettable.Get()
	})
	return value
}

// Gettable is anything that can be read: a Signal, a Computed or a Value.
type Gettable[T any] interface {
	Get() T
}

// Settable is anything that can be written: a Signal or a Value.
type Settable[T any] interface {
	Set(T)
}
