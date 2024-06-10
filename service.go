package healthcheck

import (
	"fmt"
	"sync"
)

type Service struct {
	opts Options

	checksMu *sync.RWMutex
	checks   []checkRec
}

func New(opts Options) (*Service, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("bad configuration: %w", err)
	}

	return &Service{
		opts:     opts,
		checks:   nil,
		checksMu: new(sync.RWMutex),
	}, nil
}
