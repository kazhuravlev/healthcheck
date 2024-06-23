package healthcheck

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestManualCheck2(t *testing.T) {
	t.Parallel()

	t.Run("check_constructor", func(t *testing.T) {
		t.Parallel()

		check := NewManual("sample")

		require.Equal(t, "sample", check.id())
		require.Equal(t, time.Hour, check.timeout())
		require.ErrorIs(t, check.check(context.TODO()).Error, errInitial)
		require.Len(t, check.log(), 0)
	})
}
