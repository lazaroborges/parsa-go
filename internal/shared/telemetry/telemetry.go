package telemetry

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

var meter metric.Meter

// Counters and histograms for HTTP request metrics.
var (
	requestCount    metric.Int64Counter
	requestDuration metric.Float64Histogram
)

// Init sets up OTLP metrics export to Netdata.
// Returns a shutdown function that flushes pending metrics.
func Init(ctx context.Context, otlpEndpoint string) (func(context.Context) error, error) {
	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpointURL(otlpEndpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(15*time.Second))),
	)
	otel.SetMeterProvider(provider)

	meter = provider.Meter("parsa-api")
	if err := initMetrics(); err != nil {
		return provider.Shutdown, err
	}

	log.Printf("Telemetry: OTLP metrics export to %s enabled", otlpEndpoint)
	return provider.Shutdown, nil
}

func initMetrics() error {
	var err error
	requestCount, err = meter.Int64Counter("http.server.request_count",
		metric.WithDescription("Total HTTP requests"),
	)
	if err != nil {
		return err
	}
	requestDuration, err = meter.Float64Histogram("http.server.request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
	)
	return err
}

// Middleware records request count and duration per method+route+status.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)

		route := r.Pattern
		if route == "" {
			route = "unmatched"
		}
		attrs := []attribute.KeyValue{
			attribute.String("http.method", r.Method),
			attribute.String("http.route", route),
			attribute.Int("http.status_code", sw.status),
		}
		requestCount.Add(r.Context(), 1, metric.WithAttributes(attrs...))
		requestDuration.Record(r.Context(), time.Since(start).Seconds(), metric.WithAttributes(attrs...))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.status = code
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(code)
}
