package healthcheck

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/kazhuravlev/healthcheck/internal/logr"
)

func BenchmarkApiChecks(b *testing.B) {
	ctx := context.Background()
	noErrFn := func(context.Context) error { return nil }

	b.Run("new_basic", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = NewBasic("bench", time.Second, noErrFn)
		}
	})

	b.Run("basic_check", func(b *testing.B) {
		check := NewBasic("bench", time.Second, noErrFn)

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = check.check(ctx)
		}
	})

	b.Run("basic_history", func(b *testing.B) {
		check := NewBasic("bench", time.Second, noErrFn)
		for i := 0; i < benchHistorySize; i++ {
			_ = check.check(ctx)
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = check.history()
		}
	})

	b.Run("new_manual", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = NewManual("bench")
		}
	})

	b.Run("manual_set_err", func(b *testing.B) {
		check := NewManual("bench")

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			check.SetErr(nil)
		}
	})

	b.Run("manual_check", func(b *testing.B) {
		check := NewManual("bench")

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = check.check(ctx)
		}
	})

	b.Run("manual_history", func(b *testing.B) {
		check := NewManual("bench")
		for i := 0; i < benchHistorySize; i++ {
			check.SetErr(io.EOF)
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = check.history()
		}
	})

	b.Run("new_background", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = NewBackground("bench", errInitial, time.Millisecond, time.Second, time.Second, noErrFn)
		}
	})

	b.Run("background_check", func(b *testing.B) {
		check := NewBackground("bench", errInitial, time.Millisecond, time.Second, time.Second, noErrFn)

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = check.check(ctx)
		}
	})

	b.Run("background_history", func(b *testing.B) {
		check := NewBackground("bench", errInitial, time.Millisecond, time.Second, time.Second, noErrFn)
		for i := 0; i < benchHistorySize; i++ {
			check.logg.Put(logr.Rec{
				Time:  time.Now(),
				Error: io.EOF,
			})
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = check.history()
		}
	})
}
