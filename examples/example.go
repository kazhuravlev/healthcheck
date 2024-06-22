package examples

import (
	"context"
	"errors"
	"github.com/kazhuravlev/healthcheck"
	"math/rand/v2"
	"time"
)

func example() {
	ctx := context.TODO()

	// 1. Init healthcheck instance. It will store all our checks.
	hc, _ := healthcheck.New()

	// 2. Register basic check. It will be called each time when you call `/ready` endpoint.
	{
		hc.Register(ctx, healthcheck.NewBasic("postgres", time.Second, func(ctx context.Context) error {
			if rand.Float64() > 0.5 {
				return errors.New("service is not available")
			}

			return nil
		}))
	}

	// 3. Register manual check.
	{
		cacheWarmUp := healthcheck.NewManual("cache-warmup")
		cacheWarmUp.SetErr(errors.New("cache did not warmed up yet"))

		time.AfterFunc(5*time.Second, func() {
			// Mark this check as OK
			cacheWarmUp.SetErr(nil)
		})

		hc.Register(ctx, cacheWarmUp)
	}

	// 4. Register background check. It will be checked in background even nobody call `/ready` endpoint.
	{
		hc.Register(ctx, healthcheck.NewBackground(
			"clickhouse",
			errors.New("clickhouse not ready"),
			1*time.Second,
			30*time.Second,
			10*time.Second,
			func(ctx context.Context) error {
				return nil
			},
		))
	}

	// 3. Init and run a webserver for integration with Kubernetes.
	sysServer, _ := healthcheck.NewServer(hc, healthcheck.WithPort(8080))
	_ = sysServer.Run(ctx)

	// 4. Open http://localhost:8080/ready to check the status of your system
	select {}
}
