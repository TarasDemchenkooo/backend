package config

type Log struct {
	Level  string
	Format string
}

func loadLog() Log {
	return Log{
		Level:  envOr("LOG_LEVEL", "info"),
		Format: envOr("LOG_FORMAT", "json"),
	}
}

type Tracing struct {
	Enabled     bool
	Endpoint    string
	Insecure    bool
	SampleRatio float64
}

func loadTracing() (Tracing, error) {
	enabled, err := envBool("TRACING_ENABLED", true)
	if err != nil {
		return Tracing{}, err
	}

	insecure, err := envBool("OTEL_EXPORTER_OTLP_INSECURE", true)
	if err != nil {
		return Tracing{}, err
	}

	sampleRatio, err := envFloat("OTEL_TRACES_SAMPLER_ARG", 1)
	if err != nil {
		return Tracing{}, err
	}

	return Tracing{
		Enabled:     enabled,
		Endpoint:    envOr("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		Insecure:    insecure,
		SampleRatio: sampleRatio,
	}, nil
}

type Metrics struct {
	Namespace string
}

func loadMetrics() Metrics {
	return Metrics{Namespace: envOr("METRICS_NAMESPACE", "")}
}
