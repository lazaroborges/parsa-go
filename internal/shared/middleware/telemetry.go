package middleware

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Telemetry wraps an http.Handler with OpenTelemetry instrumentation.
// Records request duration, active requests, request/response sizes,
// and creates a trace span per request.
func Telemetry(next http.Handler) http.Handler {
	return otelhttp.NewMiddleware("parsa-api")(next)
}
