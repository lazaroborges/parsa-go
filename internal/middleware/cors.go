package middleware

import "net/http"

// CORS wraps an http.Handler to apply Cross-Origin Resource Sharing headers and preflight handling.
// 
// It sets the following response headers on every request:
// - Access-Control-Allow-Origin: *
// - Access-Control-Allow-Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS
// - Access-Control-Allow-Headers: Content-Type, Authorization
// - Access-Control-Max-Age: 3600
//
// For HTTP OPTIONS (preflight) requests it responds with 204 No Content and does not call the next handler.
// For all other methods it delegates request handling to the provided next handler.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
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