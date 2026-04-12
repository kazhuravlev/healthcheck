package healthcheck

import (
	"context"
	"strings"
)

func (s *Healthcheck) runCheck(ctx context.Context, check checkContainer) Check {
	return runCheckFn(ctx, check, s.opts.setCheckStatus)
}

func runCheckFn(ctx context.Context, check checkContainer, setCheckStatus ISetCheckStatusFn) Check {
	ctx, cancel := context.WithTimeout(ctx, check.Check.timeout())
	defer cancel()

	rec := check.Check.check(ctx)
	if rec.Error == nil && ctx.Err() != nil {
		rec.Error = ctx.Err()
	}

	status := StatusUp
	errText := ""
	if rec.Error != nil {
		status = StatusDown
		errText = rec.Error.Error()
	}

	// TODO(zhuravlev): run on manual and bg checks.
	setCheckStatus(check.ID, status)

	logs := check.Check.log()
	var prev []CheckState
	if len(logs) > 0 {
		prev = make([]CheckState, len(logs))
		for i, rec := range logs {
			prevStatus := StatusUp
			prevErrText := ""
			if rec.Error != nil {
				prevStatus = StatusDown
				prevErrText = rec.Error.Error()
			}
			prev[i] = CheckState{
				ActualAt: rec.Time,
				Status:   prevStatus,
				Error:    prevErrText,
			}
		}
	}

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
