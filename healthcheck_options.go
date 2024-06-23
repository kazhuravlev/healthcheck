package healthcheck

type hcOptions struct {
	logger         ILogger
	setCheckStatus func(checkID string, isReady Status)
}

// WithCheckStatusFn will provide a function that will be called at each check changes.
func WithCheckStatusFn(fn func(checkID string, isReady Status)) func(*hcOptions) {
	return func(o *hcOptions) {
		o.setCheckStatus = fn
	}
}
