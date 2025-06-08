package healthcheck

import (
	"context"
	"log/slog"
	"sync"
	"time"
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
		check.run(ctx)
	}

	s.checks = append(s.checks, checkContainer{
		ID:    checkID,
		Check: check,
	})
}

// RunAllChecks will run all check immediately.
func (s *Healthcheck) RunAllChecks(ctx context.Context) Report {
	s.checksMu.RLock()
	checksCopy := make([]checkContainer, len(s.checks))
	copy(checksCopy, s.checks)
	isShuttingDown := s.isShuttingDown
	s.checksMu.RUnlock()

	checks := make([]Check, len(checksCopy))
	{
		wg := new(sync.WaitGroup)
		wg.Add(len(checksCopy))

		// TODO(zhuravlev): do not run goroutines for checks like manual and bg check.
		for i := range checksCopy {
			go func(i int, check checkContainer) {
				defer wg.Done()

				checks[i] = s.runCheck(ctx, check)
			}(i, checksCopy[i])
		}

		wg.Wait()
	}

	if isShuttingDown {
		checks = append(checks, Check{
			Name: "__shutting_down__",
			State: CheckState{
				ActualAt: time.Now(),
				Status:   StatusDown,
				Error:    "The application in shutting down process",
			},
			Previous: nil,
		})
	}
	status := StatusUp
	for _, check := range checks {
		if check.State.Status == StatusDown {
			status = StatusDown
			break
		}
	}

	return Report{
		Status: status,
		Checks: checks,
	}
}

// Shutdown will disable all checks and set persistent marker that will immidiately return ready = false on all
// k8s requests. Shutdown should be called immideately after graceful shutdown process started.
func (s *Healthcheck) Shutdown() {
	s.checksMu.Lock()
	s.isShuttingDown = true
	s.checksMu.Unlock()
}
