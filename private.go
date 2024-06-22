package healthcheck

import (
	"context"
	"strings"
)

func (s *Healthcheck) runCheck(ctx context.Context, check checkRec) Check {
	ctx, cancel := context.WithTimeout(ctx, check.Timeout)
	defer cancel()

	resCh := make(chan result, 1)
	go func() {
		defer close(resCh)
		resCh <- check.CheckFn(ctx)
	}()

	checkStatus := StatusUp
	var checkErr string

	select {
	case <-ctx.Done():
		checkErr = ctx.Err().Error()
		checkStatus = StatusDown
	case res := <-resCh:
		if res.Err != nil {
			checkErr = res.Err.Error()
			checkStatus = StatusDown
		}
	}

	s.opts.setCheckStatus(check.ID, checkStatus)

	curState := CheckState{
		ActualAt: s.opts.time.Now(),
		Status:   checkStatus,
		Error:    checkErr,
	}

	prev := make([]CheckState, 0, maxStatesToStore)
	{
		s.checksMu.RLock()
		p := s.checkStates[check.ID]
		p.Do(func(checkState any) {
			if checkState == nil {
				return
			}

			prev = append(prev, checkState.(CheckState))
		})

		p = p.Prev()
		p.Value = curState

		s.checkStates[check.ID] = p
		s.checksMu.RUnlock()
	}

	return Check{
		Name:     check.ID,
		State:    curState,
		Previous: prev,
	}
}

func name2id(name string) (string, bool) {
	id := strings.ReplaceAll(strings.ToLower(name), "-", "_")

	return id, id == name
}
