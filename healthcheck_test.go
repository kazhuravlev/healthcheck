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

var timeNow = time.Date(1, 2, 3, 4, 5, 6, 7, time.UTC)

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}

	t.Errorf("Received unexpected error:\n%+v\n", err)
	t.FailNow()
}

func requireTrue(t *testing.T, val bool, msg string, args ...any) {
	t.Helper()
	if val {
		return
	}

	t.Error(fmt.Sprintf(msg, args...))
	t.FailNow()
}

func requireStateEqual(t *testing.T, exp, actual hc.CheckState) {
	t.Helper()

	//requireTrue(t, exp.ActualAt.Equal(actual.ActualAt), "unexpected check actual_at")
	requireTrue(t, exp.Status == actual.Status, "unexpected check status: exp %s, actual %s", exp.Status, actual.Status)
	requireTrue(t, exp.Error == actual.Error, "unexpected check error")
}

func requireReportEqual(t *testing.T, expected, actual hc.Report) {
	t.Helper()

	requireTrue(t, expected.Status == actual.Status, "unexpected status: exp %s, actual %s", expected.Status, actual.Status)
	requireTrue(t, len(expected.Checks) == len(actual.Checks), "unexpected checks count")

	for i := range expected.Checks {
		requireTrue(t, expected.Checks[i].Name == actual.Checks[i].Name, "unexpected check name")
		requireStateEqual(t, expected.Checks[i].State, actual.Checks[i].State)
		expLen := len(expected.Checks[i].Previous)
		actLen := len(actual.Checks[i].Previous)
		requireTrue(t, expLen == actLen, "unexpected previous count error. Exp (%d) Act (%d)", expLen, actLen)
		for ii := range expected.Checks[i].Previous {
			requireStateEqual(t, expected.Checks[i].Previous[ii], actual.Checks[i].Previous[ii])
		}
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

func TestManualCheck(t *testing.T) {
	t.Parallel()

	manualCheck := hc.NewManual("some_system")
	hcInst := hcWithChecks(t, manualCheck)

	t.Run("failed_by_default", func(t *testing.T) {
		res := hcInst.RunAllChecks(context.Background())
		requireReportEqual(t, hc.Report{
			Status: hc.StatusDown,
			Checks: []hc.Check{
				{Name: "some_system", State: hc.CheckState{ActualAt: timeNow, Status: hc.StatusDown, Error: "initial"}},
			},
		}, res)
	})

	t.Run("can_be_marked_as_ok", func(t *testing.T) {
		manualCheck.SetErr(nil)

		res := hcInst.RunAllChecks(context.Background())
		requireReportEqual(t, hc.Report{
			Status: hc.StatusUp,
			Checks: []hc.Check{
				{
					Name:  "some_system",
					State: hc.CheckState{ActualAt: timeNow, Status: hc.StatusUp, Error: ""},
					Previous: []hc.CheckState{
						{ActualAt: timeNow, Status: hc.StatusDown, Error: "initial"}, // from prev test
					},
				},
			},
		}, res)
	})

	t.Run("can_be_marked_as_failed", func(t *testing.T) {
		manualCheck.SetErr(fmt.Errorf("the sky was falling: %w", io.EOF))

		res := hcInst.RunAllChecks(context.Background())
		requireReportEqual(t, hc.Report{
			Status: hc.StatusDown,
			Checks: []hc.Check{
				{
					Name:  "some_system",
					State: hc.CheckState{ActualAt: timeNow, Status: hc.StatusDown, Error: "the sky was falling: EOF"},
					Previous: []hc.CheckState{
						{ActualAt: timeNow, Status: hc.StatusUp, Error: ""},          // from prev test
						{ActualAt: timeNow, Status: hc.StatusDown, Error: "initial"}, // from prev test
					},
				},
			},
		}, res)
	})
}

func TestBackgroundCheck(t *testing.T) {
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
			Checks: []hc.Check{
				{Name: "some_system", State: hc.CheckState{ActualAt: timeNow, Status: hc.StatusDown, Error: "not ready"}},
			},
		}, res)
	})

	// wait for bg check next run
	time.Sleep(delay)

	t.Run("check_current_error_nil", func(t *testing.T) {
		res := hcInst.RunAllChecks(context.Background())
		requireReportEqual(t, hc.Report{
			Status: hc.StatusUp,
			Checks: []hc.Check{
				{
					Name:  "some_system",
					State: hc.CheckState{ActualAt: timeNow, Status: hc.StatusUp, Error: ""},
					Previous: []hc.CheckState{
						{ActualAt: timeNow, Status: hc.StatusDown, Error: "not ready"}, // from prev test
					},
				},
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
			Checks: []hc.Check{
				{
					Name:  "some_system",
					State: hc.CheckState{ActualAt: timeNow, Status: hc.StatusDown, Error: "EOF"},
					Previous: []hc.CheckState{
						{ActualAt: timeNow, Status: hc.StatusUp, Error: ""},            // from prev test
						{ActualAt: timeNow, Status: hc.StatusDown, Error: "not ready"}, // from prev test
					},
				},
			},
		}, res)
	})

}

func TestService(t *testing.T) { //nolint:funlen
	t.Run("empty_healthcheck_have_status_up", func(t *testing.T) {
		t.Parallel()

		res := hcWithChecks(t).RunAllChecks(context.Background())
		requireReportEqual(t, hc.Report{
			Status: hc.StatusUp,
			Checks: []hc.Check{},
		}, res)
	})

	t.Run("fail_when_context_cancelled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		res := hcWithChecks(t, simpleCheck("always_ok", nil)).RunAllChecks(ctx)
		requireReportEqual(t, hc.Report{
			Status: hc.StatusDown,
			Checks: []hc.Check{
				{Name: "always_ok", State: hc.CheckState{ActualAt: timeNow, Status: hc.StatusDown, Error: "context canceled"}},
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
			Checks: []hc.Check{
				{Name: "check1", State: hc.CheckState{ActualAt: timeNow, Status: hc.StatusUp, Error: ""}},
				{Name: "check2", State: hc.CheckState{ActualAt: timeNow, Status: hc.StatusUp, Error: ""}},
				{Name: "check_3", State: hc.CheckState{ActualAt: timeNow, Status: hc.StatusUp, Error: ""}},
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
			Checks: []hc.Check{
				{Name: "check1", State: hc.CheckState{ActualAt: timeNow, Status: hc.StatusUp, Error: ""}},
				{Name: "check1_x", State: hc.CheckState{ActualAt: timeNow, Status: hc.StatusUp, Error: ""}},
				{Name: "check1_x_x", State: hc.CheckState{ActualAt: timeNow, Status: hc.StatusUp, Error: ""}},
				{Name: "check1_x_x_x", State: hc.CheckState{ActualAt: timeNow, Status: hc.StatusUp, Error: ""}},
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
			Checks: []hc.Check{
				{Name: "always_ok", State: hc.CheckState{ActualAt: timeNow, Status: hc.StatusUp, Error: ""}},
				{Name: "always_ok_x", State: hc.CheckState{ActualAt: timeNow, Status: hc.StatusUp, Error: ""}},
				{Name: "context_timeout", State: hc.CheckState{ActualAt: timeNow, Status: hc.StatusDown, Error: "context deadline exceeded"}},
			},
		}, res)
	})
}

func TestPrevious(t *testing.T) { //nolint:funlen
	t.Parallel()

	manualCheck := hc.NewManual("x")
	hcInst := hcWithChecks(t, manualCheck)

	t.Run("run_all_checks_ignore_manual_checks", func(t *testing.T) {
		hcInst.RunAllChecks(context.Background())
		hcInst.RunAllChecks(context.Background())
		hcInst.RunAllChecks(context.Background())
		report := hcInst.RunAllChecks(context.Background())

		requireTrue(t, len(report.Checks[0].Previous) == 0, "RunAllChecks should not affect log of states for manual check")
	})

	t.Run("previous_will_filled_on_each_manual_change", func(t *testing.T) {
		manualCheck.SetErr(nil)
		manualCheck.SetErr(io.EOF)
		manualCheck.SetErr(io.ErrUnexpectedEOF)

		report := hcInst.RunAllChecks(context.Background())

		check := report.Checks[0]
		requireStateEqual(t, hc.CheckState{ActualAt: timeNow, Status: hc.StatusDown, Error: "unexpected EOF"}, check.State)

		prev := check.Previous
		requireTrue(t, len(prev) == 3, "no error, eof, unexpected eof")
		requireStateEqual(t, hc.CheckState{ActualAt: timeNow, Status: hc.StatusDown, Error: "EOF"}, prev[0])
		requireStateEqual(t, hc.CheckState{ActualAt: timeNow, Status: hc.StatusUp, Error: ""}, prev[1])
		requireStateEqual(t, hc.CheckState{ActualAt: timeNow, Status: hc.StatusDown, Error: "initial"}, prev[2])
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

func TestBackgroundCheckStop(t *testing.T) {
	var mu sync.Mutex
	callCount := 0
	check := hc.NewBackground("test_bg", nil, 0, 100*time.Millisecond, time.Second, func(ctx context.Context) error {
		mu.Lock()
		callCount++
		mu.Unlock()
		return nil
	})

	hcInst, err := hc.New()
	requireTrue(t, err == nil, "new should not produce error")
	ctx, cancel := context.WithCancel(context.Background())
	hcInst.Register(ctx, check)

	// Wait for a run
	time.Sleep(350 * time.Millisecond)

	cancel()

	mu.Lock()
	initialCount := callCount
	mu.Unlock()

	// Wait a bit more
	time.Sleep(200 * time.Millisecond)

	// Ensure no more calls were made after Stop
	mu.Lock()
	finalCount := callCount
	mu.Unlock()

	requireTrue(t, initialCount == finalCount, "background check should not run after Stop()")
	requireTrue(t, finalCount >= 3, "expected at least 3 calls before stop")
}

func TestShutdown(t *testing.T) {
	t.Parallel()

	t.Run("shutdown_with_no_checks", func(t *testing.T) {
		t.Parallel()

		hcInst, err := hc.New()
		requireNoError(t, err)

		// Before shutdown - should be up
		res := hcInst.RunAllChecks(context.Background())
		requireReportEqual(t, hc.Report{
			Status: hc.StatusUp,
			Checks: []hc.Check{},
		}, res)

		// Call shutdown
		hcInst.Shutdown()

		// After shutdown - should be down with shutdown check
		res = hcInst.RunAllChecks(context.Background())
		requireReportEqual(t, hc.Report{
			Status: hc.StatusDown,
			Checks: []hc.Check{
				{
					Name: "__shutting_down__",
					State: hc.CheckState{
						ActualAt: time.Now(),
						Status:   hc.StatusDown,
						Error:    "The application in shutting down process",
					},
					Previous: nil,
				},
			},
		}, res)
	})

	t.Run("shutdown_with_passing_checks", func(t *testing.T) {
		t.Parallel()

		hcInst := hcWithChecks(t,
			simpleCheck("check1", nil),
			simpleCheck("check2", nil),
		)

		// Before shutdown - should be up
		res := hcInst.RunAllChecks(context.Background())
		requireReportEqual(t, hc.Report{
			Status: hc.StatusUp,
			Checks: []hc.Check{
				{Name: "check1", State: hc.CheckState{ActualAt: timeNow, Status: hc.StatusUp, Error: ""}},
				{Name: "check2", State: hc.CheckState{ActualAt: timeNow, Status: hc.StatusUp, Error: ""}},
			},
		}, res)

		// Call shutdown
		hcInst.Shutdown()

		// After shutdown - should be down with shutdown check
		res = hcInst.RunAllChecks(context.Background())
		requireTrue(t, res.Status == hc.StatusDown, "status should be down after shutdown")
		requireTrue(t, len(res.Checks) == 3, "should have 3 checks after shutdown (2 original + shutdown)")

		// Find shutdown check
		shutdownCheck := helpFindCheck(t, res.Checks, "__shutting_down__")
		requireTrue(t, shutdownCheck.State.Status == hc.StatusDown, "shutdown check should be down")
		requireTrue(t, shutdownCheck.State.Error == "The application in shutting down process", "shutdown check should have correct error message")
	})

	t.Run("shutdown_with_failing_checks", func(t *testing.T) {
		t.Parallel()

		hcInst := hcWithChecks(t,
			simpleCheck("check1", nil),
			simpleCheck("check2", errors.New("service error")),
		)

		// Before shutdown - should already be down due to failing check
		res := hcInst.RunAllChecks(context.Background())
		requireTrue(t, res.Status == hc.StatusDown, "status should be down due to failing check")
		requireTrue(t, len(res.Checks) == 2, "should have 2 checks")

		// Call shutdown
		hcInst.Shutdown()

		// After shutdown - should still be down with shutdown check added
		res = hcInst.RunAllChecks(context.Background())
		requireTrue(t, res.Status == hc.StatusDown, "status should be down after shutdown")
		requireTrue(t, len(res.Checks) == 3, "should have 3 checks after shutdown (2 original + shutdown)")

		// Find shutdown check
		shutdownCheck := helpFindCheck(t, res.Checks, "__shutting_down__")
		requireTrue(t, shutdownCheck.State.Status == hc.StatusDown, "shutdown check should be down")
	})

	t.Run("multiple_shutdown_calls", func(t *testing.T) {
		t.Parallel()

		hcInst := hcWithChecks(t, simpleCheck("check1", nil))

		// Call shutdown multiple times
		hcInst.Shutdown()
		hcInst.Shutdown()
		hcInst.Shutdown()

		// Should only have one shutdown check
		res := hcInst.RunAllChecks(context.Background())
		requireTrue(t, res.Status == hc.StatusDown, "status should be down after shutdown")

		shutdownCheckCount := 0
		for i := range res.Checks {
			if res.Checks[i].Name == "__shutting_down__" {
				shutdownCheckCount++
			}
		}
		requireTrue(t, shutdownCheckCount == 1, "should have exactly one shutdown check")
	})

	t.Run("shutdown_with_manual_check", func(t *testing.T) {
		t.Parallel()

		manualCheck := hc.NewManual("manual_check")
		hcInst := hcWithChecks(t, manualCheck)

		// Set manual check to up
		manualCheck.SetErr(nil)

		// Before shutdown - should be up
		res := hcInst.RunAllChecks(context.Background())
		requireTrue(t, res.Status == hc.StatusUp, "status should be up before shutdown")

		// Call shutdown
		hcInst.Shutdown()

		// After shutdown - should be down with shutdown check
		res = hcInst.RunAllChecks(context.Background())
		requireTrue(t, res.Status == hc.StatusDown, "status should be down after shutdown")

		// Find shutdown check
		shutdownCheck := helpFindCheck(t, res.Checks, "__shutting_down__")
		requireTrue(t, shutdownCheck.State.Status == hc.StatusDown, "shutdown check should be down")
	})
}

func helpFindCheck(t *testing.T, checks []hc.Check, name string) hc.Check {
	t.Helper()

	for i := range checks {
		if checks[i].Name == name {
			return checks[i]
		}
	}

	t.Errorf("check not found: %s", name)

	return hc.Check{}
}
