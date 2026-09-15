package admin

import (
	"context"
	"net/http"
	"sync"
)

type Check func(ctx context.Context) error

const (
	statusOK           = "ok"
	statusFail         = "fail"
	statusStarting     = "starting"
	statusShuttingDown = "shutting_down"
)

type probeResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

type probeSet struct {
	mu     sync.RWMutex
	checks map[string]Check
}

func newProbeSet() *probeSet {
	return &probeSet{checks: make(map[string]Check)}
}

func (p *probeSet) add(name string, check Check) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.checks[name] = check
}

func (p *probeSet) run(ctx context.Context) (probeResponse, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	resp := probeResponse{Status: statusOK}
	if len(p.checks) == 0 {
		return resp, true
	}

	resp.Checks = make(map[string]string, len(p.checks))
	healthy := true

	for name, check := range p.checks {
		if err := check(ctx); err != nil {
			resp.Checks[name] = err.Error()
			resp.Status = statusFail
			healthy = false

			continue
		}

		resp.Checks[name] = statusOK
	}

	return resp, healthy
}

func (s *Server) readinessHandler() http.HandlerFunc {
	probe := s.probeHandler(s.readiness)

	return func(w http.ResponseWriter, r *http.Request) {
		switch s.state.Load() {
		case stateStarting:
			writeJSON(w, http.StatusServiceUnavailable, probeResponse{Status: statusStarting})
		case stateShuttingDown:
			writeJSON(w, http.StatusServiceUnavailable, probeResponse{Status: statusShuttingDown})
		default:
			probe(w, r)
		}
	}
}

func (s *Server) probeHandler(probes *probeSet) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), s.checkTimeout)
		defer cancel()

		resp, healthy := probes.run(ctx)

		status := http.StatusOK
		if !healthy {
			status = http.StatusServiceUnavailable

			s.log.WarnContext(ctx, "probe failed", "path", r.URL.Path, "checks", resp.Checks)
		}

		writeJSON(w, status, resp)
	}
}
