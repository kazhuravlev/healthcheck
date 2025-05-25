# Healthcheck for Go Applications

[![Go Reference](https://pkg.go.dev/badge/github.com/kazhuravlev/healthcheck.svg)](https://pkg.go.dev/github.com/kazhuravlev/healthcheck)
[![License](https://img.shields.io/github/license/kazhuravlev/healthcheck?color=blue)](https://github.com/kazhuravlev/healthcheck/blob/master/LICENSE)
[![Build Status](https://github.com/kazhuravlev/healthcheck/actions/workflows/tests.yml/badge.svg)](https://github.com/kazhuravlev/healthcheck/actions/workflows/tests.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/kazhuravlev/healthcheck)](https://goreportcard.com/report/github.com/kazhuravlev/healthcheck)
[![CodeCov](https://codecov.io/gh/kazhuravlev/healthcheck/branch/master/graph/badge.svg?token=tNKcOjlxLo)](https://codecov.io/gh/kazhuravlev/healthcheck)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go#utilities)

A production-ready health check library for Go applications that enables proper monitoring and graceful degradation in
modern cloud environments,
especially [Kubernetes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/).

## Why Health Checks Matter

Health checks are critical for building resilient, self-healing applications in distributed systems. They provide:

1. **Automatic Recovery**: In Kubernetes, failed health checks trigger automatic pod restarts, ensuring your application
   recovers from transient failures without manual intervention.
2. **Load Balancer Integration**: Health checks prevent traffic from being routed to unhealthy instances, maintaining
   service quality even during partial outages.
3. **Graceful Degradation**: By monitoring dependencies (databases, caches, external APIs), your application can degrade
   gracefully when non-critical services fail.
4. **Operational Visibility**: Health endpoints provide instant insight into system state, making debugging and incident
   response faster.
5. **Zero-Downtime Deployments**: Readiness checks ensure new deployments only receive traffic when fully initialized.

## Features

- Logger to log failed probes
- Automatic, manual and background checks
- Respond with all healthchecks status in JSON format
- Callback for integrate with metrics or other systems
- Integrated web server

## Quickstart

```shell
go get -u github.com/kazhuravlev/healthcheck
```

Check an [examples](./examples/example.go).

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

	// 2. Register checks that will random respond with an error.
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
