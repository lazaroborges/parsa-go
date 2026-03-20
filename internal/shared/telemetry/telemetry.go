package telemetry

import (
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var meter metric.Meter

// Counters and histograms for HTTP request metrics.
var (
	requestCount    metric.Int64Counter
	requestDuration metric.Float64Histogram
)

// MetricsHandler returns an http.Handler that serves Prometheus metrics at /metrics.
var MetricsHandler http.Handler

// Init sets up Prometheus metrics export.
// Returns a shutdown function that flushes pending metrics.
func Init(metricsPort string) (func(), error) {
	registry := promclient.NewRegistry()

	exporter, err := prometheus.New(prometheus.WithRegisterer(registry))
	if err != nil {
		return nil, err
	}

	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	otel.SetMeterProvider(provider)

	meter = provider.Meter("parsa-api")
	if err := initMetrics(); err != nil {
		return nil, err
	}

	MetricsHandler = promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	// Start a dedicated metrics server so /metrics isn't behind auth
	go func() {
		metricsMux := http.NewServeMux()
		metricsMux.Handle("/metrics", MetricsHandler)
		log.Printf("Telemetry: Prometheus metrics server on :%s/metrics", metricsPort)
		if err := http.ListenAndServe(":"+metricsPort, metricsMux); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	shutdown := func() {
		provider.Shutdown(nil)
	}
	return shutdown, nil
}

func initMetrics() error {
	var err error
	requestCount, err = meter.Int64Counter("http_server_request_count",
		metric.WithDescription("Total HTTP requests"),
	)
	if err != nil {
		return err
	}
	requestDuration, err = meter.Float64Histogram("http_server_request_duration_seconds",
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
			attribute.String("http_method", r.Method),
			attribute.String("http_route", route),
			attribute.Int("http_status_code", sw.status),
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
	if w.wroteHeader {
		return
	}
	w.status = code
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(code)
}
