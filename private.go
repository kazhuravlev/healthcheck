package healthcheck

import (
	"context"
	"strings"
)

func (s *Healthcheck) runCheck(ctx context.Context, opts hcOptions, check checkRec) Check {
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

	opts.setCheckStatus(check.ID, checkStatus)

	return Check{
		Name: check.ID,
		State: CheckState{
			ActualAt: opts.time.Now(),
			Status:   checkStatus,
			Error:    checkErr,
		},
		Previous: nil,
	}
}

func name2id(name string) (string, bool) {
	id := strings.ReplaceAll(strings.ToLower(name), "-", "_")

	return id, id == name
}
