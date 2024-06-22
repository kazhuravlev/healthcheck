package healthcheck

import (
	"container/ring"
	"log/slog"
	"os"
	"sync"
)

type Healthcheck struct {
	opts hcOptions

	checksMu    *sync.RWMutex
	checks      []checkRec
	checkStates map[string]*ring.Ring
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
		opts:        options,
		checksMu:    new(sync.RWMutex),
		checks:      nil,
		checkStates: make(map[string]*ring.Ring),
	}, nil
}
