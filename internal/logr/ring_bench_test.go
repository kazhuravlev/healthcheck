package logr

import (
	"errors"
	"testing"
	"time"
)

func benchmarkRec(i int) Rec {
	var err error
	if i%2 == 1 {
		err = errors.New("benchmark error")
	}

	return Rec{
		Time:  time.Unix(int64(i), int64(i)),
		Error: err,
	}
}

func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = New()
	}
}

func BenchmarkRingPut(b *testing.B) {
	r := New()
	rec := benchmarkRec(1)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.Put(rec)
	}
}

func BenchmarkRingGetLast(b *testing.B) {
	b.Run("empty", func(b *testing.B) {
		r := New()

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = r.GetLast()
		}
	})

	b.Run("populated", func(b *testing.B) {
		r := New()
		r.Put(benchmarkRec(1))

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = r.GetLast()
		}
	})
}

func BenchmarkRingSlicePrev(b *testing.B) {
	b.Run("empty", func(b *testing.B) {
		r := New()

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = r.Slice()
		}
	})

	b.Run("partially_populated", func(b *testing.B) {
		r := New()
		for i := 0; i < 3; i++ {
			r.Put(benchmarkRec(i))
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = r.Slice()
		}
	})

	b.Run("full", func(b *testing.B) {
		r := New()
		for i := 0; i < maxStatesToStore; i++ {
			r.Put(benchmarkRec(i))
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = r.Slice()
		}
	})
}
