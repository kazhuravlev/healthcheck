package healthcheck

import (
	"context"
	"log/slog"
	"sync"

	"github.com/kazhuravlev/just"
)

// Register will register a check.
//
// All checks should have a name. Will be better that name will contain only lowercase symbols and lodash.
// This is allowing to have the same name for Check and for metrics.
func (s *Healthcheck) Register(ctx context.Context, check ICheck) {
	s.checksMu.Lock()
	defer s.checksMu.Unlock()

	checkID, ok := name2id(check.id())
	if !ok {
		s.opts.logger.WarnContext(ctx, "choose a better name for check. see docs of Register method",
			slog.String("name", check.id()),
			slog.String("better_name", checkID))
	}

CheckID:
	for i := range s.checks {
		if s.checks[i].ID == checkID {
			newID := checkID + "_x"
			s.opts.logger.WarnContext(ctx, "check name is duplicated. add prefix",
				slog.String("name", check.id()),
				slog.String("new_name", newID))
			checkID = newID

			goto CheckID
		}
	}

	switch check := check.(type) {
	case *bgCheck:
		check.run()
	}

	s.checks = append(s.checks, checkRec{
		ID:      checkID,
		CheckFn: check.check,
		Timeout: check.timeout(),
	})
}

// RunAllChecks will run all check immediately.
func (s *Healthcheck) RunAllChecks(ctx context.Context) Report {
	s.checksMu.RLock()
	defer s.checksMu.RUnlock()

	checks := make([]Check, len(s.checks))
	{
		wg := new(sync.WaitGroup)
		wg.Add(len(s.checks))

		for i := range s.checks {
			go func(i int, check checkRec) {
				defer wg.Done()

				checks[i] = runCheck(ctx, s.opts, check)
			}(i, s.checks[i])
		}

		wg.Wait()
	}

	failedChecks := just.SliceFilter(checks, func(s Check) bool {
		return s.State.Status == StatusDown
	})

	status := just.If(len(failedChecks) == 0, StatusUp, StatusDown)

	return Report{
		Status: status,
		Checks: checks,
	}
}
