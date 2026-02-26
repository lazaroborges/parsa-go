package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsOriginAllowed(t *testing.T) {
	tests := []struct {
		name         string
		origin       string
		allowedHosts []string
		want         bool
	}{
		{
			name:         "exact match with port",
			origin:       "http://example.com:8080",
			allowedHosts: []string{"example.com:8080"},
			want:         true,
		},
		{
			name:         "hostname match ignoring port",
			origin:       "http://example.com:3000",
			allowedHosts: []string{"example.com"},
			want:         true,
		},
		{
			name:         "no match",
			origin:       "http://evil.com",
			allowedHosts: []string{"example.com"},
			want:         false,
		},
		{
			name:         "case insensitive",
			origin:       "http://Example.COM",
			allowedHosts: []string{"example.com"},
			want:         true,
		},
		{
			name:         "invalid origin URL",
			origin:       "://invalid",
			allowedHosts: []string{"example.com"},
			want:         false,
		},
		{
			name:         "subdomain mismatch",
			origin:       "http://sub.example.com",
			allowedHosts: []string{"example.com"},
			want:         false,
		},
		{
			name:         "localhost",
			origin:       "http://localhost:3000",
			allowedHosts: []string{"localhost"},
			want:         true,
		},
		{
			name:         "allowed host with whitespace",
			origin:       "http://example.com",
			allowedHosts: []string{"  example.com  "},
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isOriginAllowed(tt.origin, tt.allowedHosts)
			if got != tt.want {
				t.Errorf("isOriginAllowed(%q, %v) = %v, want %v", tt.origin, tt.allowedHosts, got, tt.want)
			}
		})
	}
}

func TestCORS_NoAllowedHosts(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := CORS([]string{})(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://any-origin.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("expected Allow-Origin *, got %q", got)
	}
}

func TestCORS_AllowedOrigin(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := CORS([]string{"example.com"})(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://example.com" {
		t.Errorf("expected Allow-Origin http://example.com, got %q", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("expected Allow-Credentials true, got %q", got)
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := CORS([]string{"example.com"})(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://evil.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestCORS_PreflightRequest(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called for preflight")
	})

	handler := CORS([]string{})(next)

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204 for preflight, got %d", rr.Code)
	}
}

func TestCORS_OAuthCallbackSkipsCORS(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := CORS([]string{"example.com"})(next)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/callback", nil)
	req.Header.Set("Origin", "http://evil.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for OAuth callback, got %d", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("expected Allow-Origin * for OAuth callback, got %q", got)
	}
}

func TestCORS_NoOriginHeader(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := CORS([]string{"example.com"})(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 when no Origin header, got %d", rr.Code)
	}
}
