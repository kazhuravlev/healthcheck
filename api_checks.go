package healthcheck

import (
	"context"
	"errors"
	"github.com/kazhuravlev/healthcheck/internal/logr"
	"time"
)

var (
	_ ICheck = (*basicCheck)(nil)
	_ ICheck = (*manualCheck)(nil)
	_ ICheck = (*bgCheck)(nil)
)

// errInitial used as initial error for some checks.
var errInitial = errors.New("initial")

type basicCheck struct {
	name string
	ttl  time.Duration
	fn   CheckFn
	logg *logr.Ring
}

// NewBasic creates a basic check. This check will only be performed when RunAllChecks is called.
//
//	hc, _ := healthcheck.New(...)
//	hc.Register(healthcheck.NewBasic("postgres", time.Second, func(context.Context) error { ... }))
func NewBasic(name string, timeout time.Duration, fn CheckFn) *basicCheck {
	return &basicCheck{
		name: name,
		ttl:  timeout,
		fn:   fn,
		logg: logr.New(),
	}
}

func (c *basicCheck) id() string             { return c.name }
func (c *basicCheck) timeout() time.Duration { return c.ttl }
func (c *basicCheck) check(ctx context.Context) logr.Rec {
	res := logr.Rec{
		Time:  time.Now(),
		Error: c.fn(ctx),
	}
	c.logg.Put(res)

	return res
}
func (c *basicCheck) log() []logr.Rec {
	return c.logg.SlicePrev()
}

type manualCheck struct {
	name string
	logg *logr.Ring
}

// NewManual create new check, that can be managed by client. Marked as failed by default.
//
//	hc, _ := healthcheck.New(...)
//	check := healthcheck.NewManual("some_subsystem")
//	check.SetError(nil)
//	hc.Register(check)
//	check.SetError(errors.New("service unavailable"))
func NewManual(name string) *manualCheck {
	check := &manualCheck{
		name: name,
		logg: logr.New(),
	}

	check.SetErr(errInitial)

	return check
}

func (c *manualCheck) SetErr(err error) {
	c.logg.Put(logr.Rec{
		Time:  time.Now(),
		Error: err,
	})
}

func (c *manualCheck) id() string             { return c.name }
func (c *manualCheck) timeout() time.Duration { return time.Hour }
func (c *manualCheck) check(_ context.Context) logr.Rec {
	rec, ok := c.logg.GetLast()
	if !ok {
		panic("manual check must have initial state")
	}

	return rec
}
func (c *manualCheck) log() []logr.Rec {
	return c.logg.SlicePrev()
}

type bgCheck struct {
	name   string
	period time.Duration
	delay  time.Duration
	ttl    time.Duration
	fn     CheckFn
	logg   *logr.Ring
}

// NewBackground will create a check that runs in background. Usually used for slow or expensive checks.
// Note: period should be greater than timeout.
//
//	hc, _ := healthcheck.New(...)
//	hc.Register(healthcheck.NewBackground("some_subsystem"))
func NewBackground(name string, initialErr error, delay, period, timeout time.Duration, fn CheckFn) *bgCheck {
	check := &bgCheck{
		name:   name,
		period: period,
		delay:  delay,
		ttl:    timeout,
		fn:     fn,
		logg:   logr.New(),
	}

	check.logg.Put(logr.Rec{
		Time:  time.Now(),
		Error: initialErr,
	})

	return check
}

func (c *bgCheck) run(ctx context.Context) {
	go func() {
		time.Sleep(c.delay)

		t := time.NewTicker(c.period)
		defer t.Stop()

		for {
			func() {
				ctx, cancel := context.WithTimeout(ctx, c.ttl)
				defer cancel()

				err := c.fn(ctx)

				c.logg.Put(logr.Rec{
					Time:  time.Now(),
					Error: err,
				})
			}()

			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()
}

func (c *bgCheck) id() string             { return c.name }
func (c *bgCheck) timeout() time.Duration { return time.Hour }
func (c *bgCheck) check(_ context.Context) logr.Rec {
	val, ok := c.logg.GetLast()
	if !ok {
		return logr.Rec{
			Time:  time.Now(),
			Error: nil,
		}
	}

	return val
}
func (c *bgCheck) log() []logr.Rec {
	return c.logg.SlicePrev()
}
