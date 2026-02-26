package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// Config holds OpenTelemetry configuration.
type Config struct {
	ServiceName  string
	OTLPEndpoint string // host:port for OTLP HTTP receiver (e.g. "localhost:4318")
}

// Providers holds initialized OTel providers for graceful shutdown.
type Providers struct {
	tracer *sdktrace.TracerProvider
	meter  *sdkmetric.MeterProvider
}

// Init initializes OpenTelemetry with OTLP HTTP exporters for traces and metrics.
// When shut down, all pending data is flushed to the collector.
func Init(ctx context.Context, cfg Config) (*Providers, error) {
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(cfg.ServiceName),
	)

	// Trace exporter (OTLP over HTTP)
	traceExp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cfg.OTLPEndpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	// Metric exporter (OTLP over HTTP)
	metricExp, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(cfg.OTLPEndpoint),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("metric exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	// Propagate trace context across service boundaries
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Providers{tracer: tp, meter: mp}, nil
}

// Shutdown flushes pending telemetry data and releases resources.
func (p *Providers) Shutdown(ctx context.Context) error {
	if err := p.tracer.Shutdown(ctx); err != nil {
		return fmt.Errorf("tracer shutdown: %w", err)
	}
	if err := p.meter.Shutdown(ctx); err != nil {
		return fmt.Errorf("meter shutdown: %w", err)
	}
	return nil
}
