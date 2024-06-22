package healthcheck_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	hc "github.com/kazhuravlev/healthcheck"
)

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}

	t.Errorf("Received unexpected error:\n%+v\n", err)
	t.FailNow()
}

func requireTrue(t *testing.T, val bool, msg string) {
	t.Helper()
	if val {
		return
	}

	t.Error(msg)
	t.FailNow()
}

func requireReportEqual(t *testing.T, expected, actual hc.Report) {
	t.Helper()

	requireTrue(t, expected.Status == actual.Status, "unexpected status")
	requireTrue(t, len(expected.Checks) == len(actual.Checks), "unexpected checks count")

	for i := range expected.Checks {
		requireTrue(t, expected.Checks[i].Name == actual.Checks[i].Name, "unexpected check name")
		requireTrue(t, expected.Checks[i].Status == actual.Checks[i].Status, "unexpected check status")
		requireTrue(t, expected.Checks[i].Error == actual.Checks[i].Error, "unexpected check error")
	}
}

func simpleCheck(name string, err error) hc.ICheck { //nolint:ireturn,nolintlint
	return hc.NewBasic(name, time.Second, func(ctx context.Context) error { return err })
}

func hcWithChecks(t *testing.T, checks ...hc.ICheck) *hc.Healthcheck {
	t.Helper()

	hcInst, err := hc.New()
	requireNoError(t, err)

	for i := range checks {
		hcInst.Register(context.TODO(), checks[i])
	}

	return hcInst
}

func TestService(t *testing.T) { //nolint:funlen
	t.Run("empty_healthcheck_have_status_up", func(t *testing.T) {
		t.Parallel()

		res := hcWithChecks(t).RunAllChecks(context.Background())
		requireReportEqual(t, hc.Report{
			Status: hc.StatusUp,
			Checks: []hc.CheckStatus{},
		}, res)
	})

	t.Run("fail_when_context_cancelled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		res := hcWithChecks(t, simpleCheck("always_ok", nil)).RunAllChecks(ctx)
		requireReportEqual(t, hc.Report{
			Status: hc.StatusDown,
			Checks: []hc.CheckStatus{
				{Name: "always_ok", Status: hc.StatusDown, Error: "context canceled"},
			},
		}, res)
	})

	t.Run("normalize_check_names", func(t *testing.T) {
		t.Parallel()

		res := hcWithChecks(t,
			simpleCheck("Check1", nil),
			simpleCheck("CHECK2", nil),
			simpleCheck("Check-3", nil),
		).RunAllChecks(context.Background())
		requireReportEqual(t, hc.Report{
			Status: hc.StatusUp,
			Checks: []hc.CheckStatus{
				{Name: "check1", Status: hc.StatusUp, Error: ""},
				{Name: "check2", Status: hc.StatusUp, Error: ""},
				{Name: "check_3", Status: hc.StatusUp, Error: ""},
			},
		}, res)
	})

	t.Run("non_unique_names_will_be_unique", func(t *testing.T) {
		t.Parallel()

		res := hcWithChecks(t,
			simpleCheck("Check1", nil),
			simpleCheck("CHECK1", nil),
			simpleCheck("Check1", nil),
			simpleCheck("check1", nil),
		).RunAllChecks(context.Background())
		requireReportEqual(t, hc.Report{
			Status: hc.StatusUp,
			Checks: []hc.CheckStatus{
				{Name: "check1", Status: hc.StatusUp, Error: ""},
				{Name: "check1_x", Status: hc.StatusUp, Error: ""},
				{Name: "check1_x_x", Status: hc.StatusUp, Error: ""},
				{Name: "check1_x_x_x", Status: hc.StatusUp, Error: ""},
			},
		}, res)
	})

	t.Run("fail_when_at_least_one_check_failed", func(t *testing.T) {
		t.Parallel()

		hcInst := hcWithChecks(t,
			simpleCheck("always_ok", nil),
			simpleCheck("always_ok", nil),
			hc.NewBasic(
				"context_timeout",
				time.Millisecond,
				func(ctx context.Context) error {
					time.Sleep(time.Second)
					return nil //nolint:nlreturn
				},
			),
		)

		res := hcInst.RunAllChecks(context.Background())
		requireReportEqual(t, hc.Report{
			Status: hc.StatusDown,
			Checks: []hc.CheckStatus{
				{Name: "always_ok", Status: hc.StatusUp, Error: ""},
				{Name: "always_ok_x", Status: hc.StatusUp, Error: ""},
				{Name: "context_timeout", Status: hc.StatusDown, Error: "context deadline exceeded"},
			},
		}, res)
	})

	t.Run("manual_check", func(t *testing.T) {
		t.Parallel()

		manualCheck := hc.NewManual("some_system")
		hcInst := hcWithChecks(t, manualCheck)

		t.Run("failed_by_default", func(t *testing.T) {
			res := hcInst.RunAllChecks(context.Background())
			requireReportEqual(t, hc.Report{
				Status: hc.StatusDown,
				Checks: []hc.CheckStatus{
					{Name: "some_system", Status: hc.StatusDown, Error: "initial status"},
				},
			}, res)
		})

		t.Run("can_be_marked_as_ok", func(t *testing.T) {
			manualCheck.SetErr(nil)

			res := hcInst.RunAllChecks(context.Background())
			requireReportEqual(t, hc.Report{
				Status: hc.StatusUp,
				Checks: []hc.CheckStatus{
					{Name: "some_system", Status: hc.StatusUp, Error: ""},
				},
			}, res)
		})

		t.Run("can_be_marked_as_failed", func(t *testing.T) {
			manualCheck.SetErr(fmt.Errorf("the sky was falling: %w", io.EOF))

			res := hcInst.RunAllChecks(context.Background())
			requireReportEqual(t, hc.Report{
				Status: hc.StatusDown,
				Checks: []hc.CheckStatus{
					{Name: "some_system", Status: hc.StatusDown, Error: "the sky was falling: EOF"},
				},
			}, res)
		})
	})

	t.Run("background_check", func(t *testing.T) {
		t.Parallel()

		errNotReady := errors.New("not ready")

		curErrorMu := new(sync.Mutex)
		var curError error

		delay := 200 * time.Millisecond
		bgCheck := hc.NewBackground(
			"some_system",
			errNotReady,
			delay,
			delay,
			10*time.Second,
			func(ctx context.Context) error {
				curErrorMu.Lock()
				defer curErrorMu.Unlock()

				return curError
			},
		)
		hcInst := hcWithChecks(t, bgCheck)

		t.Run("initial_error_is_used", func(t *testing.T) {
			res := hcInst.RunAllChecks(context.Background())
			requireReportEqual(t, hc.Report{
				Status: hc.StatusDown,
				Checks: []hc.CheckStatus{
					{Name: "some_system", Status: hc.StatusDown, Error: "not ready"},
				},
			}, res)
		})

		// wait for bg check next run
		time.Sleep(delay)

		t.Run("check_current_error_nil", func(t *testing.T) {
			res := hcInst.RunAllChecks(context.Background())
			requireReportEqual(t, hc.Report{
				Status: hc.StatusUp,
				Checks: []hc.CheckStatus{
					{Name: "some_system", Status: hc.StatusUp, Error: ""},
				},
			}, res)
		})

		// set error
		curErrorMu.Lock()
		curError = io.EOF
		curErrorMu.Unlock()
		// wait for bg check next run
		time.Sleep(delay)

		t.Run("change_status_after_each_run", func(t *testing.T) {
			res := hcInst.RunAllChecks(context.Background())
			requireReportEqual(t, hc.Report{
				Status: hc.StatusDown,
				Checks: []hc.CheckStatus{
					{Name: "some_system", Status: hc.StatusDown, Error: "EOF"},
				},
			}, res)
		})
	})
}

func TestServiceMetrics(t *testing.T) { //nolint:paralleltest
	res := make(map[string]hc.Status)
	mu := new(sync.Mutex)
	setStatus := func(id string, status hc.Status) {
		mu.Lock()
		res[id] = status
		mu.Unlock()
	}

	hcInst, err := hc.New(hc.WithCheckStatusFn(setStatus))
	requireNoError(t, err)

	hcInst.Register(context.TODO(), hc.NewBasic("check_without_error", time.Second, func(ctx context.Context) error { return nil }))
	hcInst.Register(context.TODO(), hc.NewBasic("check_with_error", time.Second, func(ctx context.Context) error { return io.EOF }))

	_ = hcInst.RunAllChecks(context.Background())

	requireTrue(t, len(res) == 2, "response must contains two elems")
	requireTrue(t, res["check_without_error"] == hc.StatusUp, "response without error must have status UP")
	requireTrue(t, res["check_with_error"] == hc.StatusDown, "response without error must have status UP")
}
