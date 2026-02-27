package main

import (
	"log"
	"net/http"

	httphandlers "parsa/internal/interfaces/http"
	"parsa/internal/shared/config"
	"parsa/internal/shared/middleware"
)

// SetupRoutes configures all HTTP routes and returns the final handler with middleware.
func SetupRoutes(deps *Dependencies, cfg *config.Config) http.Handler {
	mux := http.NewServeMux()

	// Static pages (dev only)
	mux.HandleFunc("/", httphandlers.HandleLoginPage)
	mux.HandleFunc("/login", httphandlers.HandleLoginPage)
	mux.HandleFunc("/dashboard", httphandlers.HandleDashboard)
	mux.HandleFunc("/oauth-callback", httphandlers.HandleOAuthCallback)

	// Health check
	mux.HandleFunc("/health", httphandlers.HandleHealth)

	// Public auth routes
	mux.HandleFunc("/api/auth/register", deps.AuthHandler.HandleRegister)
	mux.HandleFunc("/api/auth/login", deps.AuthHandler.HandleLogin)
	mux.HandleFunc("/api/auth/logout", deps.AuthHandler.HandleLogout)

	// Web OAuth
	mux.HandleFunc("/api/auth/oauth/url", deps.AuthHandler.HandleAuthURL)
	mux.HandleFunc("/api/auth/oauth/callback", deps.AuthHandler.HandleCallback)

	// Mobile OAuth (Google)
	mux.HandleFunc("/api/auth/oauth/mobile/start", deps.AuthHandler.HandleMobileAuthStart)
	mux.HandleFunc("/api/auth/oauth/mobile/callback", deps.AuthHandler.HandleMobileAuthCallback)

	// Apple OAuth (Mobile)
	mux.HandleFunc("/api/auth/oauth/apple/mobile/start", deps.AuthHandler.HandleAppleMobileAuthStart)
	mux.HandleFunc("/api/auth/oauth/apple/mobile/callback", deps.AuthHandler.HandleAppleMobileAuthCallback)

	// Protected routes
	authMiddleware := middleware.Auth(deps.JWT)

	mux.Handle("/api/users/me", authMiddleware(http.HandlerFunc(deps.UserHandler.HandleMe)))
	mux.Handle("/api/accounts/", authMiddleware(http.HandlerFunc(deps.AccountHandler.HandleListAccounts)))
	mux.Handle("/api/accounts/{id}", authMiddleware(http.HandlerFunc(deps.AccountHandler.HandleAccountByID)))
	mux.Handle("/api/transactions/", authMiddleware(http.HandlerFunc(deps.TransactionHandler.HandleListTransactions)))
	mux.Handle("/api/transactions/update", authMiddleware(http.HandlerFunc(deps.TransactionHandler.HandleBatchTransactions)))
	mux.Handle("/api/transactions/{id}", authMiddleware(http.HandlerFunc(deps.TransactionHandler.HandleGetTransaction)))
	mux.Handle("/api/tags/", authMiddleware(http.HandlerFunc(deps.TagHandler.HandleTags)))
	mux.Handle("/api/tags/{id}", authMiddleware(http.HandlerFunc(deps.TagHandler.HandleTagByID)))
	mux.Handle("/api/cousin-rules/", authMiddleware(http.HandlerFunc(deps.CousinRuleHandler.HandleCousinRules)))
	mux.Handle("/api/cousin-rules/{id}", authMiddleware(http.HandlerFunc(deps.CousinRuleHandler.HandleCousinRuleByID)))
	mux.Handle("/api/notifications/register-device/", authMiddleware(http.HandlerFunc(deps.NotificationHandler.HandleRegisterDevice)))
	mux.Handle("/api/notifications/preferences/", authMiddleware(http.HandlerFunc(deps.NotificationHandler.HandlePreferences)))
	mux.Handle("/api/notifications/open/", authMiddleware(http.HandlerFunc(deps.NotificationHandler.HandleOpen)))
	mux.Handle("/api/notifications/{id}", authMiddleware(http.HandlerFunc(deps.NotificationHandler.HandleNotificationByID)))
	mux.Handle("/api/notifications/", authMiddleware(http.HandlerFunc(deps.NotificationHandler.HandleNotifications)))

	// Apply global middleware
	handler := middleware.Logging(middleware.CORS(cfg.Server.AllowedHosts)(mux))

	// Apply security middleware when TLS is enabled
	if cfg.TLS.Enabled {
		handler = middleware.HSTS(middleware.SecureCookies(handler))
		log.Println("TLS security middleware enabled (HSTS + SecureCookies)")
	}

	return handler
}
