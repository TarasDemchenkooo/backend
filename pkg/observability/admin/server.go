package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	LivenessPath  = "/livez"
	ReadinessPath = "/readyz"
	MetricsPath   = "/metrics"
)

const (
	defaultAddr         = ":9090"
	defaultCheckTimeout = 2 * time.Second

	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
)

type Config struct {
	Addr         string
	Gatherer     prometheus.Gatherer
	Logger       *slog.Logger
	CheckTimeout time.Duration
}

type Server struct {
	srv          *http.Server
	log          *slog.Logger
	liveness     *probeSet
	readiness    *probeSet
	checkTimeout time.Duration
	state        atomic.Int32
}

const (
	stateStarting int32 = iota
	stateReady
	stateShuttingDown
)

func New(cfg Config) *Server {
	addr := cfg.Addr
	if addr == "" {
		addr = defaultAddr
	}

	checkTimeout := cfg.CheckTimeout
	if checkTimeout <= 0 {
		checkTimeout = defaultCheckTimeout
	}

	s := &Server{
		log:          cfg.Logger,
		liveness:     newProbeSet(),
		readiness:    newProbeSet(),
		checkTimeout: checkTimeout,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET "+LivenessPath, s.probeHandler(s.liveness))
	mux.HandleFunc("GET "+ReadinessPath, s.readinessHandler())

	if cfg.Gatherer != nil {
		mux.Handle("GET "+MetricsPath, s.metricsHandler(cfg.Gatherer))
	}

	s.srv = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	return s
}

func (s *Server) AddLivenessCheck(name string, check Check) {
	s.liveness.add(name, check)
}

func (s *Server) AddReadinessCheck(name string, check Check) {
	s.readiness.add(name, check)
}

func (s *Server) MarkReady() {
	s.state.CompareAndSwap(stateStarting, stateReady)
}

func (s *Server) MarkShuttingDown() {
	s.state.Store(stateShuttingDown)
}

func (s *Server) Addr() string { return s.srv.Addr }

func (s *Server) Handler() http.Handler { return s.srv.Handler }

func (s *Server) Start() error {
	if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve admin endpoints: %w", err)
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown admin server: %w", err)
	}

	return nil
}

func (s *Server) Close() error {
	if err := s.srv.Close(); err != nil {
		return fmt.Errorf("close admin server: %w", err)
	}

	return nil
}

func (s *Server) metricsHandler(gatherer prometheus.Gatherer) http.Handler {
	return promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{
		ErrorHandling:     promhttp.HTTPErrorOnError,
		ErrorLog:          promLogger{log: s.log},
		EnableOpenMetrics: true,
	})
}

type promLogger struct {
	log *slog.Logger
}

func (l promLogger) Println(v ...any) {
	l.log.Error("metrics handler", "error", fmt.Sprint(v...))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(v)
}
