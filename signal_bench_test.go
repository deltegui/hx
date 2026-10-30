// Benchmarks of the propagation of changes. Run them with:
//
//	go test -run xxx -bench . -benchmem .

package hx

import "testing"

func BenchmarkRowsEachOwnSignal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sigs := make([]*SignalT[int], 5000)
		sink := 0
		for j := range sigs {
			s := Signal(j)
			sigs[j] = s
			EffectFunc(func() { sink += s.Get() })
		}
		for _, s := range sigs {
			s.Update(func(n int) int { return n + 1 })
		}
	}
}

func BenchmarkOneSignalManyEffects(b *testing.B) {
	s := Signal(0)
	sink := 0
	for j := 0; j < 2000; j++ {
		EffectFunc(func() { sink += s.Get() })
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Set(i)
	}
}

func BenchmarkComputedChain(b *testing.B) {
	s := Signal(0)
	var last Gettable[int] = s
	for j := 0; j < 20; j++ {
		prev := last
		last = Computed(func() int { return prev.Get() + 1 })
	}
	sink := 0
	EffectFunc(func() { sink += last.Get() })
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Set(i)
	}
}
