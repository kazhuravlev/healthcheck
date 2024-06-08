package healthcheck

import (
	"context"
	"errors"
	"sync"
	"time"
)

type basicCheck struct {
	name string
	ttl  time.Duration
	fn   CheckFn
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
	}
}

func (c *basicCheck) id() string                       { return c.name }
func (c *basicCheck) timeout() time.Duration           { return c.ttl }
func (c *basicCheck) check(ctx context.Context) result { return result{Err: c.fn(ctx)} }

type manualCheck struct {
	name string

	mu  *sync.RWMutex
	err error
}

// NewManual create new check, that can be managed by client. Marked as failed by default.
//
//	hc, _ := healthcheck.New(...)
//	check := healthcheck.NewManual("some_subsystem")
//	check.SetError(nil)
//	hc.Register(check)
//	check.SetError(errors.New("service unavailable"))
func NewManual(name string) *manualCheck {
	return &manualCheck{
		name: name,
		mu:   new(sync.RWMutex),
		err:  errors.New("initial status"), //nolint:goerr113 // This error should not be handled
	}
}

func (c *manualCheck) SetErr(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.err = err
}

func (c *manualCheck) id() string             { return c.name }
func (c *manualCheck) timeout() time.Duration { return time.Hour }
func (c *manualCheck) check(_ context.Context) result {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return result{Err: c.err}
}
