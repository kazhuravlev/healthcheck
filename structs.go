package healthcheck

import (
	"context"
	"time"
)

type Status string

const (
	StatusUp   Status = "up"
	StatusDown Status = "down"
)

type CheckState struct {
	ActualAt time.Time `json:"actual_at"`
	Status   Status    `json:"status"`
	Error    string    `json:"error"`
}

type Check struct {
	Name     string       `json:"name"`
	State    CheckState   `json:"state"`
	Previous []CheckState `json:"previous"`
}

type Report struct {
	Status Status  `json:"status"`
	Checks []Check `json:"checks"`
}

type CheckFn func(ctx context.Context) error

type ICheck interface {
	id() string
	check(ctx context.Context) result
	timeout() time.Duration
}

type result struct {
	Err error
}

type checkRec struct {
	ID      string
	CheckFn func(ctx context.Context) result
	Timeout time.Duration
}
