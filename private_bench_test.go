package healthcheck

import (
	"context"
	"testing"
	"time"
)

const benchHistorySize = 6

func BenchmarkRunCheckFn(b *testing.B) {
	b.Run("basic_empty_history", func(b *testing.B) {
		check := checkContainer{
			ID:    "bench",
			Check: NewBasic("bench", time.Second, func(context.Context) error { return nil }),
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = runCheckFn(context.Background(), check, func(string, Status) {})
		}
	})

	b.Run("manual_with_history", func(b *testing.B) {
		check := NewManual("bench")
		for i := 0; i < benchHistorySize; i++ {
			check.SetErr(nil)
		}

		container := checkContainer{
			ID:    "bench",
			Check: check,
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = runCheckFn(context.Background(), container, func(string, Status) {})
		}
	})
}
