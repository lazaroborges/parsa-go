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
}

// Header intercepts header writes to modify Set-Cookie headers
func (w *secureCookieWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

// WriteHeader intercepts header writes to add secure flags to cookies
func (w *secureCookieWriter) WriteHeader(statusCode int) {
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
	// Check and add Secure flag if not present
	if !strings.Contains(cookie, "Secure") && !strings.Contains(cookie, "secure") {
		cookie += "; Secure"
	}

	// Check and add HttpOnly flag if not present
	if !strings.Contains(cookie, "HttpOnly") && !strings.Contains(cookie, "httponly") {
		cookie += "; HttpOnly"
	}

	// Check and add SameSite flag if not present
	if !strings.Contains(cookie, "SameSite") && !strings.Contains(cookie, "samesite") {
		cookie += "; SameSite=Strict"
	}

	return cookie
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
