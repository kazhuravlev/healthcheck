package healthcheck_test

import (
	"context"
	"github.com/kazhuravlev/healthcheck"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
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

func TestServer(t *testing.T) {
	ctrl := gomock.NewController(t)
	hc := NewMockIHealthcheck(ctrl)

	port := rand.Intn(1000) + 8000
	srv, err := healthcheck.NewServer(hc, healthcheck.WithPort(port))
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, srv.Run(ctx))

	// FIXME: fix crunch
	time.Sleep(time.Second)

	t.Run("live_returns_200", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:"+strconv.Itoa(port)+"/live", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("ready_always_call_healthcheck", func(t *testing.T) {
		hc.
			EXPECT().
			RunAllChecks(gomock.Any()).
			Return(healthcheck.Report{
				Status: healthcheck.StatusDown,
				Checks: []healthcheck.Check{},
			})

		req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:"+strconv.Itoa(port)+"/ready", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}
