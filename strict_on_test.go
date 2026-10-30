//go:build hxstrict

package hx

import "testing"

func TestStrictReads(t *testing.T) {
	s := Signal(1)
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("a Get outside of an effect must panic in strict mode")
			}
		}()
		s.Get()
	}()
	// what is fine stays fine
	EffectFunc(func() { s.Get() })
	UntrackGet[int](s)
}
