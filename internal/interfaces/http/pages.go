package http

import (
	"net/http"

	"parsa/internal/web"
)

// HandleHealth returns a simple health check response.
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

// HandleLoginPage serves the login page.
// Dev only - static HTML file serving.
func HandleLoginPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, web.FS, "login.html")
}

// HandleDashboard serves the dashboard page.
// Dev only - static HTML file serving.
func HandleDashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, web.FS, "dashboard.html")
}

// HandleOAuthCallback serves the OAuth callback page.
// Dev only - static HTML file serving.
func HandleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, web.FS, "oauth-callback.html")
}
