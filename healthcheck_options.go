package healthcheck

type hcOptions struct {
	logger         ILogger
	setCheckStatus ISetCheckStatusFn
}

// WithCheckStatusFn will provide a function that will be called at each check changes.
func WithCheckStatusFn(fn ISetCheckStatusFn) func(*hcOptions) {
	return func(o *hcOptions) {
		o.setCheckStatus = fn
	}
}

type ISetCheckStatusFn func(checkID string, isReady Status)
