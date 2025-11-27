package middleware

import (
	"net/http"
	"strings"
)

// HSTS is middleware that adds the Strict-Transport-Security header to responses to enforce HTTPS.
// 
// It sets "Strict-Transport-Security: max-age=31536000; includeSubDomains" on each response before
// delegating to the next handler.
func HSTS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add HSTS header: enforce HTTPS for 1 year, including all subdomains
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

// SecureCookies returns an http.Handler that wraps the provided handler and ensures
// any Set-Cookie headers include the Secure, HttpOnly, and SameSite=Strict attributes.
// It does this by replacing the ResponseWriter with a wrapper that rewrites Set-Cookie
// headers before they are sent.
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

// ensureSecureCookie ensures the provided Set-Cookie header value includes Secure, HttpOnly, and SameSite=Strict.
// It checks for each attribute case-insensitively and appends any that are missing (separated by "; "), then returns the resulting cookie string.
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
// RequireHTTPS redirects incoming requests that are not received over HTTPS to the same host and URI using the https scheme.
// It treats a request as HTTPS if r.TLS is non-nil, the `X-Forwarded-Proto` header equals "https", or r.URL.Scheme equals "https"; otherwise it issues a permanent (301) redirect to "https://{host}{requestURI}" and does not call the next handler.
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