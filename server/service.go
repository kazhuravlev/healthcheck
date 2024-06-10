package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kazhuravlev/healthcheck"
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
	mux := http.NewServeMux()

	mux.HandleFunc("/live", s.handleLive)
	mux.HandleFunc("/ready", s.handleReady)
	mux.Handle("/metrics", promhttp.Handler())

	httpServer := &http.Server{
		Addr:    ":" + strconv.Itoa(s.opts.port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second) //nolint:gomnd
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			s.opts.logger.ErrorContext(ctx, "shutdown webserver", slog.String("error", err.Error()))
		}
	}()

	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			s.opts.logger.ErrorContext(ctx, "run status server", slog.String("error", err.Error()))
		}
	}()

	return nil
}

func (s *Service) handleLive(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (s *Service) handleReady(w http.ResponseWriter, req *http.Request) {
	const unknownResp = `{"status":"unknown","checks":[]}`

	ctx := req.Context()
	w.Header().Set("Content-Type", "application/json")

	report := s.opts.healthcheck.RunAllChecks(ctx)
	reportJson, err := json.Marshal(report)
	if err != nil {
		s.opts.logger.ErrorContext(ctx, "marshal report", slog.String("error", err.Error()))

		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(unknownResp))
		return
	}

	switch report.Status {
	default:
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(unknownResp))
	case healthcheck.StatusUp:
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(reportJson)
	case healthcheck.StatusDown:
		s.opts.logger.ErrorContext(ctx, "status error", slog.Any("report", report))

		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(reportJson)
	}
}
