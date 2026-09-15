package tracing

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

type Config struct {
	Enabled        bool
	ServiceName    string
	ServiceVersion string
	Environment    string
	Endpoint       string
	Insecure       bool
	SampleRatio    float64
}

type Provider struct {
	shutdown func(context.Context) error
}

func New(ctx context.Context, cfg Config) (*Provider, error) {
	res, err := newResource(ctx, cfg)
	if err != nil {
		return nil, err
	}

	sampler := sdktrace.Sampler(sdktrace.NeverSample())
	opts := []sdktrace.TracerProviderOption{sdktrace.WithResource(res)}

	if cfg.Enabled {
		exporter, err := otlptracegrpc.New(ctx, exporterOptions(cfg)...)
		if err != nil {
			return nil, fmt.Errorf("create otlp trace exporter: %w", err)
		}

		sampler = ratioSampler(cfg.SampleRatio)
		opts = append(opts, sdktrace.WithBatcher(exporter))
	}

	tracerProvider := sdktrace.NewTracerProvider(append(opts, sdktrace.WithSampler(sampler))...)

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Provider{shutdown: tracerProvider.Shutdown}, nil
}

func (p *Provider) Shutdown(ctx context.Context) error {
	if err := p.shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown tracer provider: %w", err)
	}

	return nil
}

func newResource(ctx context.Context, cfg Config) (*resource.Resource, error) {
	res, err := resource.New(ctx,
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithHost(),
		resource.WithProcessRuntimeDescription(),
		resource.WithTelemetrySDK(),
		resource.WithFromEnv(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironmentNameKey.String(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("build trace resource: %w", err)
	}

	return res, nil
}

func exporterOptions(cfg Config) []otlptracegrpc.Option {
	opts := make([]otlptracegrpc.Option, 0, 2)

	switch {
	case strings.Contains(cfg.Endpoint, "://"):
		opts = append(opts, otlptracegrpc.WithEndpointURL(cfg.Endpoint))
	case cfg.Endpoint != "":
		opts = append(opts, otlptracegrpc.WithEndpoint(cfg.Endpoint))
	}

	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	return opts
}

func ratioSampler(ratio float64) sdktrace.Sampler {
	switch {
	case ratio <= 0:
		return sdktrace.ParentBased(sdktrace.NeverSample())
	case ratio >= 1:
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	default:
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
	}
}
