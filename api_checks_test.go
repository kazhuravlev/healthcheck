package healthcheck

import (
	"context"
	"github.com/stretchr/testify/require"
	"io"
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
		require.Len(t, check.history(), 0)
	})

	t.Run("set_and_unset_error", func(t *testing.T) {
		t.Parallel()

		check := NewManual("sample")
		require.Error(t, check.check(context.TODO()).Error)

		check.SetErr(nil)
		require.NoError(t, check.check(context.TODO()).Error)

		check.SetErr(io.EOF)
		require.ErrorIs(t, check.check(context.TODO()).Error, io.EOF)

		t.Run("check_logs", func(t *testing.T) {
			records := check.history()
			require.Len(t, records, 2)

			require.Equal(t, StatusUp, records[0].Status)
			require.Empty(t, records[0].Error)
			require.Equal(t, StatusDown, records[1].Status)
			require.Equal(t, errInitial.Error(), records[1].Error)
		})
	})
}
