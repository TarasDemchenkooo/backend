package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

type Config struct {
	Namespace       string
	ConstLabels     prometheus.Labels
	DurationBuckets []float64
}

type Metrics struct {
	registry *prometheus.Registry
	http     *HTTPMetrics
}

func New(cfg Config) (*Metrics, error) {
	registry := prometheus.NewRegistry()

	if err := registry.Register(collectors.NewGoCollector()); err != nil {
		return nil, fmt.Errorf("register go collector: %w", err)
	}

	processCollector := collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})
	if err := registry.Register(processCollector); err != nil {
		return nil, fmt.Errorf("register process collector: %w", err)
	}

	httpMetrics, err := newHTTPMetrics(registry, cfg)
	if err != nil {
		return nil, err
	}

	return &Metrics{registry: registry, http: httpMetrics}, nil
}

func (m *Metrics) Registry() *prometheus.Registry { return m.registry }

func (m *Metrics) HTTP() *HTTPMetrics { return m.http }
