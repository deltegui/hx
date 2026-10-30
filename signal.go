package hx

import (
	"fmt"
	"log"
	"runtime"
	"sync"
)

// tracking tells if signals follow who reads them. The engine of the browser
// needs it. A server engine turns it off (see Engine.Tracks): a signal is then a
// plain value, a Computed is calculated on every read and an effect runs once.
// With nothing to follow there is no global state, so many UIs can be built at
// the same time. It is set once, before anything runs.
var tracking = true

var (
	currentEffect *Effect
	currentRoot   *Root
	untrack       bool
	mu            sync.Mutex
)

// The effects that have to run because something they read changed wait in a
// queue, and flush runs them. A change marks what depends on it first and only
// then are the effects run, so an effect never sees half of a change.
var (
	queueMu    sync.Mutex
	queue      []*Effect
	batchDepth int
	flushing   bool
)

func accessEffect(action func(*Effect)) {
	mu.Lock()
	defer mu.Unlock()
	action(currentEffect)
}

// source is what an effect can read: a signal or a computed value. An effect
// remembers the version of each source it read, and runs again only if one of
// them has a different version.
type source interface {
	subscribersOf() map[*Effect]struct{}
	version() uint64
	refresh() // makes the value current: a computed value recalculates if it is stale
}

// dep is a source an effect read, and the version it had then.
type dep struct {
	src source
	ver uint64
}

// subscribe registers the current effect as a subscriber of the source.
// The subscription is removed when the effect cleans up.
func subscribe(src source) {
	if !tracking {
		return
	}
	accessEffect(func(currentEffect *Effect) {
		if currentEffect == nil {
			if !untrack {
				where := ""
				if _, file, line, ok := runtime.Caller(4); ok {
					where = fmt.Sprintf(" (%s:%d)", file, line)
				}
				message := "Cannot call Get if there is no effect" + where + ": the value is read once and the UI will not follow it. " +
					"Use BindText or BindValue, an effect (node.Effect, hx.Dynamic), or UntrackGet if reading it once is what you want."
				if strictReads {
					panic("hx: " + message)
				}
				log.Print(message)
			}
			return
		}
		subscribers := src.subscribersOf()
		if _, ok := subscribers[currentEffect]; ok {
			return
		}
		subscribers[currentEffect] = struct{}{}
		currentEffect.deps = append(currentEffect.deps, dep{src, src.version()})
		currentEffect.cleanUps = append(currentEffect.cleanUps, func() {
			delete(subscribers, currentEffect)
		})
	})
}

// notify marks everything that depends on a source that changed and, unless a
// batch is open or effects are already running, runs the effects that are due.
func notify(subscribers map[*Effect]struct{}) {
	if len(subscribers) == 0 {
		return
	}
	// Marking does not touch the subscriptions, so the map can be walked as it is.
	for effect := range subscribers {
		effect.markStale()
	}
	flushIfIdle()
}

func flushIfIdle() {
	queueMu.Lock()
	idle := batchDepth == 0
	queueMu.Unlock()
	if idle {
		flush()
	}
}

// flush runs the effects in the queue, one after another, until it is empty. An
// effect that is in the queue runs only if what it read really changed. Effects
// that are made due while it runs wait in the same queue, so only one flush runs
// at a time and a Set inside an effect does not nest.
func flush() {
	queueMu.Lock()
	if flushing {
		queueMu.Unlock()
		return
	}
	flushing = true
	queueMu.Unlock()
	defer func() {
		queueMu.Lock()
		flushing = false
		queueMu.Unlock()
	}()

	for {
		queueMu.Lock()
		if len(queue) == 0 {
			queueMu.Unlock()
			return
		}
		effect := queue[0]
		queue = queue[1:]
		// An effect created inside another one dies when the other one runs again.
		// If one of its owners is waiting for its turn, it goes after it: running
		// it now would be wasted, since it is about to be disposed.
		if effect.ownerIsWaiting() {
			queue = append(queue, effect)
			queueMu.Unlock()
			continue
		}
		effect.queued = false
		queueMu.Unlock()

		if !effect.disposed && effect.depsChanged() {
			effect.run()
		}
	}
}

// Batch runs fn and holds back the effects until it ends: any number of Set
// calls inside it run each affected effect once, at the end, and the effects see
// the final values. A Computed read inside fn is always current. Batches nest:
// the effects run when the outermost one ends, even if fn panics.
//
// Event handlers and the end of an Action already run in a batch.
func Batch(fn func()) {
	if !tracking {
		fn()
		return
	}
	queueMu.Lock()
	batchDepth++
	queueMu.Unlock()
	defer func() {
		queueMu.Lock()
		batchDepth--
		end := batchDepth == 0
		queueMu.Unlock()
		if end {
			flush()
		}
	}()
	fn()
}

// SignalT is a reactive value. Create it with Signal.
type SignalT[T any] struct {
	value       T
	ver         uint64
	subscribers map[*Effect]struct{}
}

// Signal creates a reactive value. Reading it with Get inside an effect makes the effect run again every time the value changes.
func Signal[T any](initial T) *SignalT[T] {
	if !tracking {
		return &SignalT[T]{value: initial}
	}
	return &SignalT[T]{
		value:       initial,
		subscribers: make(map[*Effect]struct{}),
	}
}

// Get returns the value. Inside an effect (EffectFunc, node.Effect, Bind*, Computed) it subscribes the effect to the signal.
// Outside of an effect it just returns the value, and logs a warning unless it runs inside Untrack or an event handler.
func (signal *SignalT[T]) Get() T {
	subscribe(signal)
	return signal.value
}

func (signal *SignalT[T]) subscribersOf() map[*Effect]struct{} { return signal.subscribers }
func (signal *SignalT[T]) version() uint64                     { return signal.ver }
func (signal *SignalT[T]) refresh()                            {}

// Subscribers is how many effects follow the signal. It is meant for tests: an
// effect that was disposed must not be counted.
func (signal *SignalT[T]) Subscribers() int { return len(signal.subscribers) }

// Set changes the value and runs the effects that read the signal. Outside of an
// effect and of a Batch they have run when Set returns; inside one, they run when
// it ends. It always notifies, even if the new value equals the old one.
func (signal *SignalT[T]) Set(v T) {
	signal.value = v
	signal.ver++
	notify(signal.subscribers)
}

// Update sets the value to fn(current value). It does not subscribe the current effect.
func (signal *SignalT[T]) Update(fn func(T) T) {
	signal.Set(fn(signal.value))
}

// Effect is a function that runs again when any signal it read changes.
type Effect struct {
	fn       func()
	owner    *Effect // the effect that was running when this one was made
	disposed bool
	running  bool
	queued   bool // it is in the queue of effects that are due
	stale    bool // a computed value that may be out of date
	computed bool
	// downstream is, for a computed value, who reads it. A stale mark goes on to them.
	downstream map[*Effect]struct{}
	deps       []dep
	childs     []*Effect
	cleanUps   []func()
}

// newEffect makes an effect and gives it its owner: the effect that is running,
// or else the root.
func newEffect(fn func(), computed bool) *Effect {
	e := &Effect{
		fn:       fn,
		computed: computed,
		childs:   make([]*Effect, 0),
		cleanUps: make([]func(), 0),
	}
	mu.Lock()
	if currentEffect != nil {
		e.owner = currentEffect
		currentEffect.childs = append(currentEffect.childs, e)
	} else if currentRoot != nil {
		currentRoot.effects = append(currentRoot.effects, e)
	}
	mu.Unlock()
	return e
}

// EffectFunc runs fn now and again each time a signal read by fn changes.
// An effect created inside another effect is its child and is disposed when the parent runs again.
// An effect created inside CreateRoot is owned by the root. Any other effect lives forever unless you call Dispose.
// For effects tied to a node use INode.Effect instead.
func EffectFunc(fn func()) *Effect {
	if !tracking {
		fn()
		return &Effect{disposed: true}
	}
	e := newEffect(fn, false)
	e.run()
	return e
}

func (e *Effect) run() {
	mu.Lock()
	prev := currentEffect
	currentEffect = e
	e.running = true
	mu.Unlock()

	defer func() {
		mu.Lock()
		currentEffect = prev
		e.running = false
		mu.Unlock()
	}()

	e.clean()
	e.fn()
}

// ownerIsWaiting tells if an effect that owns this one is in the queue.
func (e *Effect) ownerIsWaiting() bool {
	for owner := e.owner; owner != nil; owner = owner.owner {
		if owner.queued && !owner.disposed {
			return true
		}
	}
	return false
}

// markStale says that something this effect reads has changed. A computed value
// passes it on to the ones that read it; an effect waits in the queue. One that
// is running ignores it: an effect does not run itself again for what it wrote.
func (e *Effect) markStale() {
	mu.Lock()
	skip := e.disposed || e.running
	mu.Unlock()
	if skip {
		return
	}
	if e.computed {
		if e.stale {
			return
		}
		e.stale = true
		for target := range e.downstream {
			target.markStale()
		}
		return
	}
	queueMu.Lock()
	if !e.queued {
		e.queued = true
		queue = append(queue, e)
	}
	queueMu.Unlock()
}

// depsChanged tells if something the effect read has a new version. It brings
// each computed value up to date first, in the order they were read, so a
// computed value that comes out the same does not wake the effect.
func (e *Effect) depsChanged() bool {
	for i := 0; i < len(e.deps); i++ {
		d := e.deps[i]
		d.src.refresh()
		if d.src.version() != d.ver {
			return true
		}
	}
	return false
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
	e.deps = e.deps[:0]
}

// Disposed tells if the effect has been stopped.
func (e *Effect) Disposed() bool { return e.disposed }

// Dispose stops the effect for good: it is unsubscribed from every
// signal and it will never run again. Child effects are disposed too.
func (e *Effect) Dispose() {
	e.disposed = true
	e.clean()
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
	if !tracking {
		return fn(func() {})
	}
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
	if !tracking {
		return
	}
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
	fn          func() T // set when nothing is tracked: the value is calculated on every read
	value       T
	ver         uint64
	node        *Effect // the effect that calculates it, owned like any effect
	subscribers map[*Effect]struct{}
}

// Computed creates a value derived from other signals or computeds. It is
// calculated when it is created and again, when it is read, after something it
// depends on changed. It only wakes the effects that read it when the result is
// different, and one that reads it always gets the current value, even in the
// middle of a Batch.
func Computed[T comparable](fn func() T) *ComputedT[T] {
	if !tracking {
		return &ComputedT[T]{fn: fn}
	}
	c := &ComputedT[T]{
		subscribers: make(map[*Effect]struct{}),
	}
	c.node = newEffect(func() {
		if next := fn(); next != c.value {
			c.value = next
			c.ver++
		}
	}, true)
	c.node.downstream = c.subscribers
	c.node.run()
	return c
}

// Get returns the current value and subscribes the current effect, like Signal.Get.
func (c *ComputedT[T]) Get() T {
	if c.fn != nil {
		return c.fn()
	}
	c.refresh()
	subscribe(c)
	return c.value
}

func (c *ComputedT[T]) subscribersOf() map[*Effect]struct{} { return c.subscribers }
func (c *ComputedT[T]) version() uint64                     { return c.ver }

// refresh calculates the value again if something it depends on changed.
func (c *ComputedT[T]) refresh() {
	n := c.node
	if n == nil || n.disposed || !n.stale {
		return
	}
	n.stale = false
	if n.depsChanged() {
		n.run()
	}
}

// Subscribers is how many effects follow the computed value, like SignalT.Subscribers.
func (c *ComputedT[T]) Subscribers() int { return len(c.subscribers) }

// Untrack runs fn without subscribing the current effect to the signals fn reads.
func Untrack(fn func()) {
	if !tracking {
		fn()
		return
	}
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
