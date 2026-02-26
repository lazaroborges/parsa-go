package middleware

import (
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var (
	httpTracer             = otel.Tracer("parsa/http")
	httpMeter              = otel.Meter("parsa/http")
	httpRequestDuration, _ = httpMeter.Float64Histogram("http.server.request.duration",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)
	httpRequestTotal, _ = httpMeter.Int64Counter("http.server.request.total",
		metric.WithDescription("Total HTTP requests"),
	)
)

// Tracing creates OpenTelemetry spans and records HTTP metrics for each request.
func Tracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := httpTracer.Start(r.Context(), r.Method+" "+r.URL.Path,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.target", r.URL.Path),
			),
		)
		defer span.End()

		start := time.Now()
		wrapped := wrapResponseWriter(w)
		next.ServeHTTP(wrapped, r.WithContext(ctx))

		status := wrapped.status
		if status == 0 {
			status = 200
		}

		span.SetAttributes(attribute.Int("http.status_code", status))
		if status >= 500 {
			span.SetStatus(codes.Error, http.StatusText(status))
		}

		attrs := metric.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.route", r.URL.Path),
			attribute.Int("http.status_code", status),
		)
		httpRequestDuration.Record(ctx, time.Since(start).Seconds(), attrs)
		httpRequestTotal.Add(ctx, 1, attrs)
	})
}
