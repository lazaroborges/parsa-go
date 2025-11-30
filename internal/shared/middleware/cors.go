package middleware

import (
	"net/http"
	"net/url"
	"strings"
)

func CORS(allowedHosts []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// If no allowed hosts configured, allow all origins (backwards compatible)
			if len(allowedHosts) == 0 {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" {
				// Validate origin against allowed hosts
				if isOriginAllowed(origin, allowedHosts) {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				} else {
					// Origin not allowed - don't set CORS headers
					// Browser will block the response
					http.Error(w, "Origin not allowed", http.StatusForbidden)
					return
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "3600")

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isOriginAllowed checks if an origin matches any of the allowed hosts
func isOriginAllowed(origin string, allowedHosts []string) bool {
	// Parse the origin URL
	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}

	originHost := strings.ToLower(originURL.Host)

	// Check against each allowed host
	for _, allowedHost := range allowedHosts {
		allowedHost = strings.ToLower(strings.TrimSpace(allowedHost))

		// Exact match
		if originHost == allowedHost {
			return true
		}

		// If allowed host doesn't have a port, match just the hostname part
		if !strings.Contains(allowedHost, ":") {
			originHostname := originHost
			if idx := strings.Index(originHost, ":"); idx != -1 {
				originHostname = originHost[:idx]
			}
			if originHostname == allowedHost {
				return true
			}
		}
	}

	return false
}
