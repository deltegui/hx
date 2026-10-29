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

type SignalT[T any] struct {
	value       T
	subscribers map[*Effect]struct{}
}

func Signal[T any](initial T) *SignalT[T] {
	return &SignalT[T]{
		value:       initial,
		subscribers: make(map[*Effect]struct{}),
	}
}

func (signal *SignalT[T]) Get() T {
	subscribe(signal.subscribers)
	return signal.value
}

func (signal *SignalT[T]) Set(v T) {
	signal.value = v
	signal.notify()
}

func (signal *SignalT[T]) Update(fn func(T) T) {
	signal.Set(fn(signal.value))
}

func (signal *SignalT[T]) notify() {
	notify(signal.subscribers)
}

type Effect struct {
	fn          func()
	isScheduled bool
	disposed    bool
	childs      []*Effect
	cleanUps    []func()
}

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

type ComputedT[T comparable] struct {
	value           T
	dependentEffect *Effect
	subscribers     map[*Effect]struct{}
}

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

func (c *ComputedT[T]) Get() T {
	subscribe(c.subscribers)
	return c.value
}

func (c *ComputedT[T]) notify() {
	notify(c.subscribers)
}

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

func UntrackGet[T any](gettable Gettable[T]) T {
	var value T
	Untrack(func() {
		value = gettable.Get()
	})
	return value
}

type Gettable[T any] interface {
	Get() T
}

type Settable[T any] interface {
	Set(T)
}
