package healthcheck

import (
	"log/slog"
	"os"
	"sync"
)

type Healthcheck struct {
	opts hcOptions

	checksMu *sync.RWMutex
	checks   []checkRec
}

func New(opts ...func(*hcOptions)) (*Healthcheck, error) {
	options := hcOptions{
		logger:         slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		setCheckStatus: func(string, Status) {},
		time:           realTime{},
	}
	for _, opt := range opts {
		opt(&options)
	}

	return &Healthcheck{
		opts:     options,
		checks:   nil,
		checksMu: new(sync.RWMutex),
	}, nil
}
