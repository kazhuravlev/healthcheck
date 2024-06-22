package healthcheck

import "time"

type hcOptions struct {
	logger         ILogger
	setCheckStatus func(checkID string, isReady Status)
	time           iTime
}

type iTime interface {
	Now() time.Time
}

type realTime struct{}

func (realTime) Now() time.Time { return time.Now() }

// WithCheckStatusFn will provide a function that will be called at each check changes.
func WithCheckStatusFn(fn func(checkID string, isReady Status)) func(*hcOptions) {
	return func(o *hcOptions) {
		o.setCheckStatus = fn
	}
}

// WithTime is for mocking a time function.
func WithTime(impl iTime) func(*hcOptions) {
	return func(o *hcOptions) {
		o.time = impl
	}
}
