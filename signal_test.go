package hx

import "testing"

func TestSignalRerunsEffect(t *testing.T) {
	s := Signal(1)
	runs, last := 0, 0
	EffectFunc(func() {
		runs++
		last = s.Get()
	})
	s.Set(2)
	if runs != 2 || last != 2 {
		t.Fatalf("runs=%d last=%d", runs, last)
	}
}

func TestEffectReadingTwiceSubscribesOnce(t *testing.T) {
	s := Signal(1)
	EffectFunc(func() {
		s.Get()
		s.Get()
	})
	if len(s.subscribers) != 1 {
		t.Fatalf("subscribers=%d", len(s.subscribers))
	}
	e := currentEffectCleanups(s)
	if e != 1 {
		t.Fatalf("cleanups=%d", e)
	}
}

func currentEffectCleanups(s *SignalT[int]) int {
	for e := range s.subscribers {
		return len(e.cleanUps)
	}
	return 0
}

func TestEffectRunsOncePerSet(t *testing.T) {
	a, b := Signal(1), Signal(1)
	runs := 0
	EffectFunc(func() {
		runs++
		a.Get()
		b.Get()
	})
	a.Set(2)
	b.Set(2)
	if runs != 3 {
		t.Fatalf("runs=%d", runs)
	}
}

func TestChildEffectsAreDisposedOnRerun(t *testing.T) {
	outer := Signal(0)
	inner := Signal(0)
	innerRuns := 0
	EffectFunc(func() {
		outer.Get()
		EffectFunc(func() {
			innerRuns++
			inner.Get()
		})
	})
	outer.Set(1) // the first inner effect must die, a new one is created
	before := innerRuns
	inner.Set(1)
	if innerRuns-before != 1 {
		t.Fatalf("inner effect ran %d times, want 1", innerRuns-before)
	}
}

func TestComputed(t *testing.T) {
	s := Signal(1)
	c := Computed(func() int { return s.Get() * 2 })
	got := 0
	EffectFunc(func() { got = c.Get() })
	s.Set(5)
	if got != 10 {
		t.Fatalf("got=%d", got)
	}
}

func TestDisposedChildOfComputedDoesNotRun(t *testing.T) {
	outer := Signal(0)
	source := Signal(1)
	c := Computed(func() int { return source.Get() })
	runs := 0
	EffectFunc(func() {
		outer.Get()
		EffectFunc(func() {
			runs++
			c.Get()
		})
	})
	outer.Set(1)
	before := runs
	source.Set(2)
	if runs-before != 1 {
		t.Fatalf("effect ran %d times after computed changed, want 1", runs-before)
	}
}

func TestComputedSubscribersAreCleaned(t *testing.T) {
	s := Signal(1)
	c := Computed(func() int { return s.Get() })
	o := Signal(0)
	EffectFunc(func() {
		o.Get()
		c.Get()
	})
	o.Set(1)
	o.Set(2)
	if len(c.subscribers) != 1 {
		t.Fatalf("subscribers=%d", len(c.subscribers))
	}
}

func TestDispose(t *testing.T) {
	s := Signal(1)
	runs := 0
	e := EffectFunc(func() {
		runs++
		s.Get()
	})
	e.Dispose()
	s.Set(2)
	if runs != 1 {
		t.Fatalf("runs=%d", runs)
	}
}

func TestNestedUntrackRestoresState(t *testing.T) {
	Untrack(func() {
		Untrack(func() {})
		if !untrack {
			t.Fatal("untrack flag lost after nested Untrack")
		}
	})
	if untrack || currentEffect != nil {
		t.Fatal("state not restored")
	}
}

func TestEffectPanicRestoresCurrentEffect(t *testing.T) {
	func() {
		defer func() { recover() }()
		EffectFunc(func() { panic("boom") })
	}()
	if currentEffect != nil {
		t.Fatal("currentEffect leaked after panic")
	}
}

func TestUntrackedGetDoesNotSubscribe(t *testing.T) {
	s := Signal(1)
	runs := 0
	EffectFunc(func() {
		runs++
		UntrackGet[int](s)
	})
	s.Set(2)
	if runs != 1 {
		t.Fatalf("runs=%d", runs)
	}
}

func TestCreateRootDisposesEffects(t *testing.T) {
	s := Signal(1)
	runs := 0
	dispose := CreateRoot(func(dispose func()) func() {
		EffectFunc(func() {
			runs++
			s.Get()
		})
		return dispose
	})
	s.Set(2)
	if runs != 2 {
		t.Fatalf("runs=%d", runs)
	}
	dispose()
	dispose() // twice is fine
	s.Set(3)
	if runs != 2 {
		t.Fatalf("effect ran after dispose, runs=%d", runs)
	}
}

func TestCreateRootDoesNotCaptureNestedEffectsTwice(t *testing.T) {
	outer := Signal(0)
	inner := Signal(0)
	innerRuns := 0
	CreateRoot(func(dispose func()) int {
		EffectFunc(func() {
			outer.Get()
			EffectFunc(func() {
				innerRuns++
				inner.Get()
			})
		})
		return 0
	})
	outer.Set(1)
	before := innerRuns
	inner.Set(1)
	if innerRuns-before != 1 {
		t.Fatalf("inner ran %d times", innerRuns-before)
	}
}

func TestCreateRootRestoresPreviousRoot(t *testing.T) {
	CreateRoot(func(dispose func()) int {
		CreateRoot(func(dispose func()) int { return 0 })
		if currentRoot == nil {
			t.Fatal("outer root lost")
		}
		return 0
	})
	if currentRoot != nil {
		t.Fatal("root leaked")
	}
}

func TestOnCleanup(t *testing.T) {
	s := Signal(0)
	cleanups := 0
	e := EffectFunc(func() {
		s.Get()
		OnCleanup(func() { cleanups++ })
	})
	s.Set(1)
	if cleanups != 1 {
		t.Fatalf("cleanups=%d after rerun", cleanups)
	}
	e.Dispose()
	if cleanups != 2 {
		t.Fatalf("cleanups=%d after dispose", cleanups)
	}

	rootCleanups := 0
	dispose := CreateRoot(func(dispose func()) func() {
		OnCleanup(func() { rootCleanups++ })
		return dispose
	})
	dispose()
	if rootCleanups != 1 {
		t.Fatalf("rootCleanups=%d", rootCleanups)
	}
}
