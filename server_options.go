package healthcheck

import (
	"context"
	"log/slog"
)

type serverOptions struct {
	port        int
	healthcheck IHealthcheck
	logger      ILogger
}

type ILogger interface {
	WarnContext(ctx context.Context, msg string, attrs ...any)
	ErrorContext(ctx context.Context, msg string, attrs ...any)
}

type IHealthcheck interface {
	RunAllChecks(ctx context.Context) Report
}

func WithLogger(logger *slog.Logger) func(o *serverOptions) {
	return func(o *serverOptions) {
		o.logger = logger
	}
}

func WithPort(port int) func(o *serverOptions) {
	return func(o *serverOptions) {
		o.port = port
	}
}

func WithHealthcheck(hc *Healthcheck) func(o *serverOptions) {
	return func(o *serverOptions) {
		o.healthcheck = hc
	}
}
