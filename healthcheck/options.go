package healthcheck

import (
	"context"
	"log/slog"
	"os"
)

//go:generate options-gen -from-struct=Options -defaults-from=var
type Options struct {
	logger         ILogger                              `validate:"required"`
	setCheckStatus func(checkID string, isReady Status) `validate:"required"`
}

var defaultOptions = Options{ //nolint:gochecknoglobals
	logger:         slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	setCheckStatus: func(string, Status) {},
}

type ILogger interface {
	WarnContext(ctx context.Context, msg string, attrs ...any)
}
