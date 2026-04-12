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

	return Check{
		Name: check.ID,
		State: CheckState{
			ActualAt: rec.Time,
			Status:   status,
			Error:    errText,
		},
		Previous: check.Check.history(),
	}
}

func name2id(name string) (string, bool) {
	id := strings.ReplaceAll(strings.ToLower(name), "-", "_")

	return id, id == name
}
