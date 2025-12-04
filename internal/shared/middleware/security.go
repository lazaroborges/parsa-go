package middleware

import (
	"net/http"
	"strings"
)

// HSTS adds Strict-Transport-Security header to enforce HTTPS
func HSTS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add HSTS header: enforce HTTPS for 1 year, including all subdomains
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

// SecureCookies wraps ResponseWriter to enforce secure cookie flags
func SecureCookies(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wrap response writer to intercept Set-Cookie headers
		sw := &secureCookieWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)
	})
}

// secureCookieWriter wraps http.ResponseWriter to enforce secure cookie attributes
type secureCookieWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

// Header intercepts header writes to modify Set-Cookie headers
func (w *secureCookieWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

// Write ensures WriteHeader is called through the wrapper before writing response body
func (w *secureCookieWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// WriteHeader intercepts header writes to add secure flags to cookies
func (w *secureCookieWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true

	// Process all Set-Cookie headers
	cookies := w.ResponseWriter.Header()["Set-Cookie"]
	if len(cookies) > 0 {
		// Remove existing Set-Cookie headers
		w.ResponseWriter.Header().Del("Set-Cookie")

		// Add them back with secure attributes
		for _, cookie := range cookies {
			securedCookie := ensureSecureCookie(cookie)
			w.ResponseWriter.Header().Add("Set-Cookie", securedCookie)
		}
	}

	w.ResponseWriter.WriteHeader(statusCode)
}

// ensureSecureCookie adds Secure, HttpOnly, and SameSite attributes to a cookie
func ensureSecureCookie(cookie string) string {
	parts := strings.Split(cookie, ";")

	hasSecure := false
	hasHttpOnly := false
	hasSameSite := false

	for i, p := range parts {
		p = strings.TrimSpace(p)
		lower := strings.ToLower(p)

		switch {
		case lower == "secure":
			hasSecure = true
		case lower == "httponly":
			hasHttpOnly = true
		case strings.HasPrefix(lower, "samesite"):
			hasSameSite = true
		}

		parts[i] = p
	}

	if !hasSecure {
		parts = append(parts, "Secure")
	}
	if !hasHttpOnly {
		parts = append(parts, "HttpOnly")
	}
	if !hasSameSite {
		parts = append(parts, "SameSite=Strict")
	}

	return strings.Join(parts, "; ")
}

// RequireHTTPS redirects HTTP requests to HTTPS
// This should only be used when the Go app is handling TLS directly
// (not when behind a reverse proxy like nginx)
func RequireHTTPS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request came via HTTPS
		// Look at the scheme or X-Forwarded-Proto header
		isHTTPS := r.TLS != nil ||
			r.Header.Get("X-Forwarded-Proto") == "https" ||
			r.URL.Scheme == "https"

		if !isHTTPS {
			// Construct HTTPS URL
			httpsURL := "https://" + r.Host + r.RequestURI

			// Redirect permanently to HTTPS
			http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// IsHostAllowed validates a host against the allowed hosts list.
// Used for preventing redirect poisoning attacks when redirecting HTTP to HTTPS.
// Returns true if no allowed hosts are configured (backwards compatible).
func IsHostAllowed(host string, allowedHosts []string) bool {
	if len(allowedHosts) == 0 {
		return true
	}

	host = strings.ToLower(strings.TrimSpace(host))
	hostWithoutPort, _, err := net.SplitHostPort(host)
	if err != nil {
		hostWithoutPort = host // No port present
	}

	for _, allowedHost := range allowedHosts {
		allowedHost = strings.ToLower(strings.TrimSpace(allowedHost))
		allowedHostWithoutPort := allowedHost
		if idx := strings.Index(allowedHost, ":"); idx != -1 {
			allowedHostWithoutPort = allowedHost[:idx]
		}

		if host == allowedHost || hostWithoutPort == allowedHostWithoutPort {
			return true
		}
	}

	return false
}
