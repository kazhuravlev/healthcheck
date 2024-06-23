package healthcheck

import (
	"context"
	"github.com/kazhuravlev/healthcheck/internal/logr"
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
	check(ctx context.Context) logr.Rec
	timeout() time.Duration
	log() []logr.Rec
}

type checkContainer struct {
	ID    string
	Check ICheck
}
