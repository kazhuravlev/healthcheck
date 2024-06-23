package healthcheck

import (
	"context"
	"github.com/kazhuravlev/healthcheck/internal/logr"
	"github.com/kazhuravlev/just"
	"strings"
	"time"
)

func (s *Healthcheck) runCheck(ctx context.Context, check checkContainer) Check {
	ctx, cancel := context.WithTimeout(ctx, check.Check.timeout())
	defer cancel()

	resCh := make(chan logr.Rec, 1)
	go func() {
		defer close(resCh)
		resCh <- check.Check.check(ctx)
	}()

	rec := logr.Rec{
		Time:  time.Now(),
		Error: nil,
	}

	select {
	case <-ctx.Done():
		rec = logr.Rec{
			Time:  time.Now(),
			Error: ctx.Err(),
		}
	case rec = <-resCh:
	}

	status := StatusUp
	errText := ""
	if rec.Error != nil {
		status = StatusDown
		errText = rec.Error.Error()
	}

	// TODO(zhuravlev): run on manual and bg checks.
	s.opts.setCheckStatus(check.ID, status)

	prev := just.SliceMap(check.Check.log(), func(rec logr.Rec) CheckState {
		status := StatusUp
		errText := ""

		if rec.Error != nil {
			errText = rec.Error.Error()
			status = StatusDown
		}

		return CheckState{
			ActualAt: rec.Time,
			Status:   status,
			Error:    errText,
		}
	})

	return Check{
		Name: check.ID,
		State: CheckState{
			ActualAt: rec.Time,
			Status:   status,
			Error:    errText,
		},
		Previous: prev,
	}
}

func name2id(name string) (string, bool) {
	id := strings.ReplaceAll(strings.ToLower(name), "-", "_")

	return id, id == name
}
