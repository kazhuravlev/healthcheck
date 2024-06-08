package server

import (
	"context"
	"fmt"
	"github.com/kazhuravlev/healthcheck/healthcheck"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type Service struct {
	opts Options
}

func New(opts Options) (*Service, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("bad configuration: %w", err)
	}

	return &Service{
		opts: opts,
	}, nil
}

func (s *Service) Run(ctx context.Context) error {
	echoInst := echo.New()
	echoInst.HideBanner = true
	echoInst.HidePort = true

	echoInst.GET("/live", s.handleLive)
	echoInst.GET("/ready", s.handleReady)
	echoInst.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	go func() {
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second) //nolint:gomnd
		defer cancel()

		if err := echoInst.Shutdown(ctx); err != nil {
			s.opts.logger.ErrorContext(ctx, "shutdown webserver", slog.String("error", err.Error()))
		}
	}()

	go func() {
		if err := echoInst.Start(":" + strconv.Itoa(s.opts.port)); err != nil {
			s.opts.logger.ErrorContext(ctx, "start status server", slog.String("error", err.Error()))
		}
	}()

	return nil
}

func (s *Service) handleLive(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{})
}

func (s *Service) handleReady(c echo.Context) error {
	report := s.opts.healthcheck.RunAllChecks(c.Request().Context())
	switch report.Status {
	case healthcheck.StatusUp:
		return c.JSON(http.StatusOK, report)
	case healthcheck.StatusDown:
		s.opts.logger.ErrorContext(c.Request().Context(), "status error",
			slog.Int("status", c.Response().Status),
			slog.String("path", c.Path()),
			slog.Any("report", report))

		return c.JSON(http.StatusInternalServerError, report)
	}

	return c.JSON(http.StatusInternalServerError, map[string]any{
		"status": "unknown",
		"checks": []string{},
	})
}
