package hx

import (
	"fmt"
	"strings"
	"testing"
)

func TestBatchRunsAnEffectOnce(t *testing.T) {
	a, b := Signal(1), Signal(1)
	runs := 0
	EffectFunc(func() { runs++; a.Get(); b.Get() })
	Batch(func() {
		a.Set(2)
		b.Set(2)
		if runs != 1 {
			t.Fatalf("an effect ran inside the batch: runs=%d", runs)
		}
	})
	if runs != 2 {
		t.Fatalf("runs=%d, want 2 (one at creation, one for the whole batch)", runs)
	}
}

func TestBatchesNest(t *testing.T) {
	a := Signal(0)
	runs := 0
	EffectFunc(func() { runs++; a.Get() })
	Batch(func() {
		Batch(func() { a.Set(1) })
		if runs != 1 {
			t.Fatalf("the inner batch flushed: runs=%d", runs)
		}
		a.Set(2)
	})
	if runs != 2 {
		t.Fatalf("runs=%d, want 2", runs)
	}
}

// Inside a batch the effects wait, but a Computed is never stale when read.
func TestComputedIsFreshInsideABatch(t *testing.T) {
	a := Signal(1)
	double := Computed(func() int { return a.Get() * 2 })
	effectSaw := 0
	EffectFunc(func() { effectSaw = double.Get() })
	Batch(func() {
		a.Set(5)
		if got := UntrackGet[int](double); got != 10 {
			t.Fatalf("computed read inside the batch: %d, want 10", got)
		}
		if effectSaw != 2 {
			t.Fatalf("the effect ran before the batch ended: %d", effectSaw)
		}
	})
	if effectSaw != 10 {
		t.Fatalf("effectSaw=%d, want 10", effectSaw)
	}
}

func TestBatchFlushesEvenIfItPanics(t *testing.T) {
	a := Signal(0)
	runs := 0
	EffectFunc(func() { runs++; a.Get() })
	func() {
		defer func() { recover() }()
		Batch(func() { a.Set(1); panic("boom") })
	}()
	if runs != 2 {
		t.Fatalf("runs=%d, the change before the panic must reach the effect", runs)
	}
	a.Set(2) // and the batch is over: effects run again right away
	if runs != 3 {
		t.Fatalf("runs=%d after the batch", runs)
	}
}

// A change that reaches an effect through two computed values: the effect must
// run once and never see one of them updated and the other one not.
func TestNoGlitchWhenAChangeTakesTwoPaths(t *testing.T) {
	a := Signal(1)
	b := Computed(func() int { return a.Get() + 1 })
	c := Computed(func() int { return a.Get() * 10 })
	var seen []string
	EffectFunc(func() { seen = append(seen, fmt.Sprintf("b=%d c=%d", b.Get(), c.Get())) })
	a.Set(2)
	if got := strings.Join(seen, " | "); got != "b=2 c=10 | b=3 c=20" {
		t.Fatalf("the effect saw: %s", got)
	}
}

func TestNoGlitchWithALongChain(t *testing.T) {
	a := Signal(1)
	b := Computed(func() int { return a.Get() + 1 })
	c := Computed(func() int { return b.Get() + 1 })
	d := Computed(func() int { return c.Get() + a.Get() })
	var seen []int
	EffectFunc(func() { seen = append(seen, d.Get()+b.Get()) })
	a.Set(10)
	// at creation: d=4 (3+1), b=2 -> 6. after: b=11, c=12, d=22 -> 33
	if len(seen) != 2 || seen[0] != 6 || seen[1] != 33 {
		t.Fatalf("the effect saw %v, want [6 33]", seen)
	}
}

// A computed value that ends up the same does not wake the effects that read it.
func TestEffectDoesNotRunIfTheComputedValueIsTheSame(t *testing.T) {
	a := Signal(1)
	parity := Computed(func() int { return a.Get() % 2 })
	runs := 0
	EffectFunc(func() { runs++; parity.Get() })
	a.Set(3) // still odd
	a.Set(5)
	if runs != 1 {
		t.Fatalf("runs=%d, want 1", runs)
	}
	a.Set(6)
	if runs != 2 {
		t.Fatalf("runs=%d, want 2", runs)
	}
}

// A Set inside an effect: whoever depends on it runs after that effect ends,
// and the effect does not run itself again for a signal it also reads.
func TestSetInsideAnEffect(t *testing.T) {
	x, y := Signal(0), Signal(0)
	var order []string
	EffectFunc(func() {
		v := x.Get()
		order = append(order, fmt.Sprintf("A start x=%d", v))
		y.Set(v * 2)
		y.Get() // reads what it wrote
		order = append(order, "A end")
	})
	EffectFunc(func() { order = append(order, fmt.Sprintf("B y=%d", y.Get())) })
	order = nil
	x.Set(3)
	want := "A start x=3 | A end | B y=6"
	if got := strings.Join(order, " | "); got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
}

// The parent is created before its child, so when both read the same signal the
// parent runs first and the old child is disposed before it could run.
func TestParentRunsBeforeItsChild(t *testing.T) {
	s := Signal(0)
	childRuns := 0
	EffectFunc(func() {
		s.Get()
		EffectFunc(func() { childRuns++; s.Get() })
	})
	for i := 1; i <= 50; i++ { // a map has no order: repeat to be sure
		before := childRuns
		s.Set(i)
		if childRuns-before != 1 {
			t.Fatalf("Set %d: %d child runs, want 1 (the new child, never the old one)", i, childRuns-before)
		}
	}
}

func TestADisposedEffectInTheQueueDoesNotRun(t *testing.T) {
	a := Signal(0)
	runs := 0
	e := EffectFunc(func() { runs++; a.Get() })
	Batch(func() {
		a.Set(1)
		e.Dispose()
	})
	if runs != 1 {
		t.Fatalf("a disposed effect ran: runs=%d", runs)
	}
}

func TestSignalSetAlwaysNotifiesEvenIfEqual(t *testing.T) {
	a := Signal(1)
	runs := 0
	EffectFunc(func() { runs++; a.Get() })
	a.Set(1)
	if runs != 2 {
		t.Fatalf("runs=%d, Set must notify even if the value is equal", runs)
	}
}
