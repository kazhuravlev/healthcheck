package server

import (
	"context"
	"log/slog"
	"os"

	"github.com/kazhuravlev/healthcheck/healthcheck"
)

//go:generate options-gen -from-struct=Options -defaults-from=var
type Options struct {
	port        int                  `validate:"required"`
	healthcheck *healthcheck.Service `validate:"required"`
	logger      ILogger              `validate:"required"`
}

var defaultOptions = Options{ //nolint:exhaustruct,gochecknoglobals
	port:   5001, //nolint:gomnd
	logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
}

type ILogger interface {
	ErrorContext(ctx context.Context, msg string, attrs ...any)
}
