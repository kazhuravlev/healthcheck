package healthcheck_test

import (
	"context"
	"github.com/kazhuravlev/healthcheck"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

//go:generate mockgen -destination server_mock_test.go -package healthcheck_test . IHealthcheck

func TestReadyHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	hc := NewMockIHealthcheck(ctrl)

	f := func(status healthcheck.Status, expStatus int, expBody string) {
		t.Run("", func(t *testing.T) {
			ctx := context.Background()
			hc.
				EXPECT().
				RunAllChecks(gomock.Eq(ctx)).
				Return(healthcheck.Report{
					Status: status,
					Checks: []healthcheck.Check{},
				})

			req := httptest.NewRequest(http.MethodGet, "/ready", nil)
			w := httptest.NewRecorder()

			handler := healthcheck.ReadyHandler(hc)
			handler(w, req)

			res := w.Result()
			defer res.Body.Close()

			require.Equal(t, expStatus, res.StatusCode)
			require.Equal(t, "application/json", res.Header.Get("Content-Type"))

			bb, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			require.Equal(t, expBody, string(bb))
		})
	}

	f(healthcheck.StatusUp, http.StatusOK, `{"status":"up","checks":[]}`)

	f(healthcheck.StatusDown, http.StatusInternalServerError, `{"status":"down","checks":[]}`)

	f("i_do_not_know", http.StatusInternalServerError, `{"status":"unknown","checks":[]}`)
}
