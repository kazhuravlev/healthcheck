# Healthcheck for go applications

[![Go Reference](https://pkg.go.dev/badge/github.com/kazhuravlev/healthcheck.svg)](https://pkg.go.dev/github.com/kazhuravlev/healthcheck)
[![License](https://img.shields.io/github/license/kazhuravlev/healthcheck?color=blue)](https://github.com/kazhuravlev/healthcheck/blob/master/LICENSE)
[![Build Status](https://github.com/kazhuravlev/healthcheck/actions/workflows/tests.yml/badge.svg)](https://github.com/kazhuravlev/healthcheck/actions/workflows/tests.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/kazhuravlev/healthcheck)](https://goreportcard.com/report/github.com/kazhuravlev/healthcheck)
[![CodeCov](https://codecov.io/gh/kazhuravlev/healthcheck/branch/master/graph/badge.svg?token=tNKcOjlxLo)](https://codecov.io/gh/kazhuravlev/healthcheck)

This tools allow you to unlock the kubernetes
feature [Liveness and Readiness](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/).

## Features

- Logger to log failed probes
- Automatic and manual checks
- Respond with all healthchecks status in JSON format
- Callback for integrate with metrics or other systems
- Integrated web server

## Quickstart

```shell
go get -u github.com/kazhuravlev/healthckeck
```

```go
package main

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/kazhuravlev/healthcheck"
)

func main() {
	ctx := context.TODO()

	// 1. Init healthcheck instance. It will store all our checks.
	hc, _ := healthcheck.New()

	// 2. Register checks for our redis client
	hc.Register(ctx, healthcheck.NewBasic("redis", time.Second, func(ctx context.Context) error {
		if rand.Float64() > 0.5 {
			return errors.New("service is not available")
		}

		return nil
	}))

	// 3. Init and run a webserver for integration with Kubernetes.
	sysServer, _ := healthcheck.NewServer(hc, healthcheck.WithPort(8080))
	_ = sysServer.Run(ctx)

	// 4. Open http://localhost:8080/ready to check the status of your system
	select {}
}

```
