package healthcheck

import (
	"context"
	"encoding/json"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Server struct {
	opts serverOptions
}

func NewServer(hc IHealthcheck, opts ...func(*serverOptions)) (*Server, error) {
	options := serverOptions{
		port:        8000,
		healthcheck: hc,
		logger:      slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}

	for _, opt := range opts {
		opt(&options)
	}

	return &Server{opts: options}, nil
}

func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/live", s.handleLive)
	mux.HandleFunc("/ready", ReadyHandler(s.opts.healthcheck))
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

func (s *Server) handleLive(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// ReadyHandler build a http.HandlerFunc from healthcheck.
func ReadyHandler(healthcheck IHealthcheck) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		const unknownResp = `{"status":"unknown","checks":[]}`

		ctx := req.Context()
		w.Header().Set("Content-Type", "application/json")

		report := healthcheck.RunAllChecks(ctx)
		reportJson, err := json.Marshal(report)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(unknownResp))
			return
		}

		switch report.Status {
		default:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(unknownResp))
		case StatusUp:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(reportJson)
		case StatusDown:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write(reportJson)
		}
	}
}
