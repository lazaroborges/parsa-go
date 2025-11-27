package middleware

import (
	"context"
	"net/http"
	"strings"

	"parsa/internal/auth"
)

type ContextKey string

const (
	UserIDKey ContextKey = "user_id"
	EmailKey  ContextKey = "email"
)

func Auth(jwt *auth.JWT) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var token string

			// Try HttpOnly cookie first (browser requests)
			if cookie, err := r.Cookie("access_token"); err == nil {
				token = cookie.Value
			} else {
				// Fall back to Authorization header (API clients)
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" {
					http.Error(w, "Authentication required", http.StatusUnauthorized)
					return
				}
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) != 2 || parts[0] != "Bearer" {
					http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
					return
				}
				token = parts[1]
			}

			claims, err := jwt.Validate(token)
			if err != nil {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Add user ID to request context
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, EmailKey, claims.Email)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
