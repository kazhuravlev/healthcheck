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

- **Multiple Check Types**: Basic (sync), Manual, and Background (async) checks for different use cases
- **Kubernetes Native**: Built-in `/live` and `/ready` endpoints following k8s conventions
- **JSON Status Reports**: Detailed health status with history for debugging
- **Metrics Integration**: Callbacks for Prometheus or other monitoring systems
- **Thread-Safe**: Concurrent-safe operations with proper synchronization
- **Graceful Shutdown**: Proper cleanup of background checks
- **Check History**: Last 5 states stored for each check for debugging

## Installation

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

	// 1. Create healthcheck instance
	hc, _ := healthcheck.New()

	// 2. Register a simple check
	hc.Register(ctx, healthcheck.NewBasic("redis", time.Second, func(ctx context.Context) error {
		if rand.Float64() > 0.5 {
			return errors.New("service is not available")
		}
		return nil
	}))

	// 3. Start HTTP server
	server, _ := healthcheck.NewServer(hc, healthcheck.WithPort(8080))
	_ = server.Run(ctx)

	// 4. Check health at http://localhost:8080/ready
	select {}
}
## Types of Health Checks

### 1. Basic Checks (Synchronous)

Basic checks run on-demand when the `/ready` endpoint is called. Use these for:

- Fast operations (< 1 second)
- Checks that need fresh data
- Low-cost operations

```go
// Database connectivity check
dbCheck := healthcheck.NewBasic("postgres", time.Second, func (ctx context.Context) error {
  return db.PingContext(ctx)
})
```

### 2. Background Checks (Asynchronous)

Background checks run periodically in a separate goroutine (in background mode). Use these for:

- Expensive operations (API calls, complex queries)
- Checks with rate limits (when checks running rarely than k8s requests to `/ready`)
- Operations that can use slightly stale data

```go
// External API health check - runs every 30 seconds
apiCheck := healthcheck.NewBackground(
  "payment-api",
  nil, // initial error state
  5*time.Second, // initial delay
  30*time.Second, // check interval
  5*time.Second,  // timeout per check
  func (ctx context.Context) error {
    resp, err := client.Get("https://api.payment.com/health")
    if err != nil {
      return err
    }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
      return errors.New("unhealthy")
    }
    return nil
  },
)
```

### 3. Manual Checks

Manual checks are controlled by your application logic. Use these for:

- Initialization states (cache warming, data loading)
- Circuit breaker patterns
- Feature flags

```go
// Cache warming check
cacheCheck := healthcheck.NewManual("cache-warmed")
hc.Register(ctx, cacheCheck)

// Set unhealthy during startup
cacheCheck.SetErr(errors.New("cache warming in progress"))

// After cache is warmed
cacheCheck.SetErr(nil)
```

## Best Practices

### 1. Choose the Right Check Type

| Scenario            | Check Type | Why                          |
|---------------------|------------|------------------------------|
| Database ping       | Basic      | Fast, needs fresh data       |
| File system check   | Basic      | Fast, local operation        |
| External API health | Background | Expensive, rate-limited      |
| Message queue depth | Background | Metrics query, can be stale  |
| Cache warmup status | Manual     | Application-controlled state |

### 2. Set Appropriate Timeouts

```go
// ❌ Bad: Too long timeout blocks readiness. Timeout should less than timeout in k8s
healthcheck.NewBasic("db", 30*time.Second, checkFunc)

// ✅ Good: Short timeout 
healthcheck.NewBasic("db", 1*time.Second, checkFunc)
```

### 3. Use Status Codes Correctly

- **Liveness** (`/live`): Should almost always return 200 OK
    - Only fail if the application is in an unrecoverable state
    - Kubernetes will restart the pod on failure

- **Readiness** (`/ready`): Should fail when:
    - Critical dependencies are unavailable
    - Application is still initializing
    - Application is shutting down


```
