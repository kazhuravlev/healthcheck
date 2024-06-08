# Healthcheck for go applications

This tools allow you to unlock the kubernetes
feature [Liveness and Readiness](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/).

## Quickstart

```shell
go get -u github.com/kazhuravlev/healthckeck
```

```go
package main

import (
	"context"
	"github.com/kazhuravlev/healthcheck/healthcheck"
	"github.com/kazhuravlev/healthcheck/server"
	redis "github.com/redis/go-redis/v9"
	"time"
)

func main() {
	ctx := context.TODO()

	// 1. Init healthcheck instance. It will store all our checks.
	hc, _ := healthcheck.New(healthcheck.NewOptions())

	// 2. Init some component that important for your system. In this example - redis client. 
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

	// 3. Register checks for our redis client
	hc.Register(ctx, healthcheck.NewBasic("redis", time.Second, func(ctx context.Context) error {
		return redisClient.Ping(ctx).Err()
	}))

	// 4. Init and run a webserver for integration with Kubernetes.
	sysServer, _ := server.New(server.NewOptions(
		server.WithPort(8080),
		server.WithHealthcheck(hc),
	))
	_ = sysServer.Run(ctx)

	// 5. Open http://localhost:8080/ready to check the status of your system
	select {}
}

```