package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"parsa/internal/shared/auth"
)

func TestAuth(t *testing.T) {
	// Setup JWT
	secret := "test-secret"
	jwt := auth.NewJWT(secret)
	validToken, _ := jwt.Generate(1, "test@example.com")

	tests := []struct {
		name           string
		token          string
		setupRequest   func(r *http.Request)
		expectedStatus int
		expectedUser   bool
	}{
		{
			name:  "Valid Token in Cookie",
			token: validToken,
			setupRequest: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: "access_token", Value: validToken})
			},
			expectedStatus: http.StatusOK,
			expectedUser:   true,
		},
		{
			name:  "Valid Token in Header",
			token: validToken,
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer "+validToken)
			},
			expectedStatus: http.StatusOK,
			expectedUser:   true,
		},
		{
			name:  "No Token",
			token: "",
			setupRequest: func(r *http.Request) {
			},
			expectedStatus: http.StatusUnauthorized,
			expectedUser:   false,
		},
		{
			name:  "Invalid Token",
			token: "invalid",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer invalid")
			},
			expectedStatus: http.StatusUnauthorized,
			expectedUser:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handler that checks context
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				userID, ok := r.Context().Value(UserIDKey).(int64)
				if !ok && tt.expectedUser {
					t.Error("Expected user ID in context, got none")
				}
				if ok && !tt.expectedUser {
					t.Error("Unexpected user ID in context")
				}
				if ok && userID != 1 {
					t.Errorf("Expected user ID 1, got %d", userID)
				}
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with middleware
			handler := Auth(jwt)(nextHandler)

			req := httptest.NewRequest("GET", "/", nil)
			tt.setupRequest(req)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatus)
			}
		})
	}
}
