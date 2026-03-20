package telemetry

import (
	"context"
	"fmt"
	"log"
	"net"
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

// metricsHandler holds the Prometheus HTTP handler for the /metrics endpoint.
var metricsHandler http.Handler

// Init sets up Prometheus metrics export.
// Returns a shutdown function that flushes pending metrics.
func Init(metricsAddr string) (func(), error) {
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

	metricsHandler = promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	// Start a dedicated metrics server so /metrics isn't behind auth.
	// Bind synchronously so startup fails fast on port conflicts.
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", metricsHandler)
	ln, err := net.Listen("tcp", metricsAddr)
	if err != nil {
		return nil, fmt.Errorf("metrics listener on %s: %w", metricsAddr, err)
	}
	metricsSrv := &http.Server{Handler: metricsMux}
	go func() {
		log.Printf("Telemetry: Prometheus metrics server on %s/metrics", metricsAddr)
		if err := metricsSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := metricsSrv.Shutdown(ctx); err != nil {
			log.Printf("Metrics server shutdown error: %v", err)
		}
		if err := provider.Shutdown(ctx); err != nil {
			log.Printf("Metrics provider shutdown error: %v", err)
		}
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
