package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"parsa/internal/interfaces/scheduler"
	"parsa/internal/shared/config"
	"parsa/internal/shared/middleware"
)

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Handler      http.Handler
	Addr         string
	TLSEnabled   bool
	CertPath     string
	KeyPath      string
	RedirectHTTP bool
	AllowedHosts []string
}

// StartServers creates and starts the main server and optional redirect server.
// Returns the main server and redirect server (nil if not enabled).
func StartServers(scfg ServerConfig) (*http.Server, *http.Server) {
	srv := &http.Server{
		Addr:         scfg.Addr,
		Handler:      scfg.Handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	var redirectSrv *http.Server

	// Start HTTP redirect server if TLS redirect is enabled
	if scfg.TLSEnabled && scfg.RedirectHTTP {
		redirectSrv = createRedirectServer(scfg.AllowedHosts)
		go func() {
			log.Println("HTTP redirect server starting on :80")
			if err := redirectSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("HTTP redirect server error: %v", err)
			}
		}()
	}

	// Start main server
	go func() {
		if scfg.TLSEnabled {
			log.Printf("HTTPS server starting on %s", scfg.Addr)
			if err := srv.ListenAndServeTLS(scfg.CertPath, scfg.KeyPath); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTPS server error: %v", err)
			}
		} else {
			log.Printf("HTTP server starting on %s", scfg.Addr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTP server error: %v", err)
			}
		}
	}()

	return srv, redirectSrv
}

// GracefulShutdown performs graceful shutdown of all servers and scheduler.
func GracefulShutdown(srv, redirectSrv *http.Server, sched *scheduler.Scheduler, timeout time.Duration) {
	log.Println("Server shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Shutdown scheduler if running
	if sched != nil {
		sched.Shutdown(timeout)
	}

	// Shutdown HTTP redirect server if running
	if redirectSrv != nil {
		if err := redirectSrv.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down HTTP redirect server: %v", err)
		}
	}

	// Shutdown main server
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down main server: %v", err)
	}

	log.Println("Server stopped")
}

// createRedirectServer creates an HTTP server that redirects all requests to HTTPS.
func createRedirectServer(allowedHosts []string) *http.Server {
	redirectHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Header.Get("X-Forwarded-Host")
		if host == "" {
			host = r.Host
		}

		if !middleware.IsHostAllowed(host, allowedHosts) {
			http.Error(w, "Invalid host", http.StatusBadRequest)
			return
		}

		canonicalHost := host
		if idx := strings.Index(host, ":"); idx != -1 {
			canonicalHost = host[:idx]
		}

		httpsURL := "https://" + canonicalHost + r.RequestURI
		http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
	})

	return &http.Server{
		Addr:         ":80",
		Handler:      redirectHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// NewServerConfigFromConfig creates ServerConfig from application config.
func NewServerConfigFromConfig(handler http.Handler, cfg *config.Config) ServerConfig {
	return ServerConfig{
		Handler:      handler,
		Addr:         cfg.Server.Host + ":" + cfg.Server.Port,
		TLSEnabled:   cfg.TLS.Enabled,
		CertPath:     cfg.TLS.CertPath,
		KeyPath:      cfg.TLS.KeyPath,
		RedirectHTTP: cfg.TLS.RedirectHTTP,
		AllowedHosts: cfg.Server.AllowedHosts,
	}
}
