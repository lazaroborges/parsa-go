package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"parsa/internal/domain/account"
	"parsa/internal/domain/openfinance"
	"parsa/internal/infrastructure/crypto"
	ofclient "parsa/internal/infrastructure/openfinance"
	"parsa/internal/infrastructure/postgres"
	httphandlers "parsa/internal/interfaces/http"
	"parsa/internal/interfaces/scheduler"
	"parsa/internal/shared/auth"
	"parsa/internal/shared/config"
	"parsa/internal/shared/middleware"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}

func run() error {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Connect to database
	db, err := postgres.New(cfg.Database.ConnectionString())
	if err != nil {
		return err
	}
	defer db.Close()

	log.Println("Connected to database")

	// Initialize encryptor
	encryptor, err := crypto.NewEncryptor(cfg.Encryption.Key)
	if err != nil {
		return err
	}

	// Initialize repositories
	userRepo := postgres.NewUserRepository(db, encryptor)
	transactionRepo := postgres.NewTransactionRepository(db)
	itemRepo := postgres.NewItemRepository(db)
	creditCardDataRepo := postgres.NewCreditCardDataRepository(db)
	bankRepo := postgres.NewBankRepository(db)

	// Initialize infrastructure layer (new architecture)
	accountRepo := postgres.NewAccountRepository(db)

	// Initialize domain services (business logic layer)
	accountService := account.NewService(accountRepo)

	// Initialize auth components
	jwt := auth.NewJWT(cfg.JWT.Secret)
	googleOAuth := auth.NewGoogleOAuthProvider(
		cfg.OAuth.Google.ClientID,
		cfg.OAuth.Google.ClientSecret,
		cfg.OAuth.Google.WebCallbackURL,
	)

	// Initialize handlers
	authHandler := httphandlers.NewAuthHandler(userRepo, googleOAuth, jwt, cfg.OAuth.Google.MobileCallbackURL, cfg.OAuth.Google.WebCallbackURL)
	userHandler := httphandlers.NewUserHandler(userRepo, accountRepo)
	// Use new service-based handler (refactored architecture)
	accountHandler := httphandlers.NewAccountHandler(accountService)
	transactionHandler := httphandlers.NewTransactionHandler(transactionRepo, accountRepo)

	// Create router
	mux := http.NewServeMux()

	// Serve pages
	mux.HandleFunc("/", handleLoginPage)
	mux.HandleFunc("/login", handleLoginPage)
	mux.HandleFunc("/dashboard", handleDashboard)
	mux.HandleFunc("/oauth-callback", handleOAuthCallback)

	// Public API routes
	mux.HandleFunc("/api/auth/register", authHandler.HandleRegister)
	mux.HandleFunc("/api/auth/login", authHandler.HandleLogin)
	mux.HandleFunc("/api/auth/logout", authHandler.HandleLogout)

	// Web OAuth
	mux.HandleFunc("/api/auth/oauth/url", authHandler.HandleAuthURL)
	mux.HandleFunc("/api/auth/oauth/callback", authHandler.HandleCallback)

	// Mobile OAuth
	mux.HandleFunc("/api/auth/oauth/mobile/start", authHandler.HandleMobileAuthStart)
	mux.HandleFunc("/api/auth/oauth/mobile/callback", authHandler.HandleMobileAuthCallback)

	mux.HandleFunc("/health", handleHealth)

	// Protected routes - wrap with auth middleware
	authMiddleware := middleware.Auth(jwt)

	mux.Handle("/api/users/me", authMiddleware(http.HandlerFunc(userHandler.HandleMe)))
	// Account routes - single handler for all /api/accounts/ paths
	mux.Handle("/api/accounts/", authMiddleware(http.HandlerFunc(accountHandler.HandleAccounts)))
	mux.Handle("/api/transactions/", authMiddleware(http.HandlerFunc(transactionHandler.HandleListTransactions)))
	mux.Handle("/api/transactions/:id", authMiddleware(http.HandlerFunc(transactionHandler.HandleGetTransaction)))

	// Apply global middleware
	handler := middleware.Logging(middleware.CORS(cfg.Server.AllowedHosts)(mux))

	// Apply security middleware when TLS is enabled
	if cfg.TLS.Enabled {
		handler = middleware.HSTS(middleware.SecureCookies(handler))
		log.Println("TLS security middleware enabled (HSTS + SecureCookies)")
	}

	// Initialize Open Finance sync (if enabled)
	var sched *scheduler.Scheduler
	if cfg.Scheduler.Enabled {
		// Initialize Open Finance client and sync services
		ofClient := ofclient.NewClient()
		accountSyncService := openfinance.NewAccountSyncService(ofClient, userRepo, accountService, itemRepo)
		transactionSyncService := openfinance.NewTransactionSyncService(ofClient, userRepo, accountService, accountRepo, transactionRepo, creditCardDataRepo, bankRepo)

		// Create job provider function that creates composite sync jobs per user
		jobProvider := func(ctx context.Context) ([]scheduler.Job, error) {
			users, err := userRepo.ListUsersWithProviderKey(ctx)
			if err != nil {
				return nil, err
			}

			// Create one composite job per user that runs account sync then transaction sync
			jobs := make([]scheduler.Job, 0, len(users))
			for _, user := range users {
				job := scheduler.NewUserSyncJob(user.ID, accountSyncService, transactionSyncService)
				jobs = append(jobs, job)
			}

			log.Printf("Job provider: Created %d sync jobs (%d users)", len(jobs), len(users))
			return jobs, nil
		}

		// Initialize scheduler
		var err error
		sched, err = scheduler.NewScheduler(scheduler.SchedulerConfig{
			ScheduleTimes: cfg.Scheduler.ScheduleTimes,
			WorkerCount:   cfg.Scheduler.WorkerCount,
			JobDelay:      cfg.Scheduler.JobDelay,
			QueueSize:     cfg.Scheduler.QueueSize,
			RunOnStartup:  cfg.Scheduler.RunOnStartup,
			JobProvider:   jobProvider,
		})
		if err != nil {
			return err
		}

		// Start scheduler
		sched.Start()
		log.Println("Sync scheduler started (accounts + transactions)")
	}

	// Create main server
	addr := cfg.Server.Host + ":" + cfg.Server.Port
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP redirect server if TLS redirect is enabled
	var redirectSrv *http.Server
	if cfg.TLS.Enabled && cfg.TLS.RedirectHTTP {
		redirectHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the host to redirect to (check X-Forwarded-Host for proxy support)
			host := r.Header.Get("X-Forwarded-Host")
			if host == "" {
				host = r.Host
			}

			// Validate host against allowed hosts to prevent redirect poisoning
			if !isHostAllowed(host, cfg.Server.AllowedHosts) {
				http.Error(w, "Invalid host", http.StatusBadRequest)
				return
			}

			// Strip port from host if present for cleaner redirect
			canonicalHost := host
			if idx := strings.Index(host, ":"); idx != -1 {
				canonicalHost = host[:idx]
			}

			// Redirect to HTTPS with validated host and request URI
			httpsURL := "https://" + canonicalHost + r.RequestURI
			http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
		})

		redirectSrv = &http.Server{
			Addr:         ":80",
			Handler:      redirectHandler,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		go func() {
			log.Println("HTTP redirect server starting on :80")
			if err := redirectSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("HTTP redirect server error: %v", err)
			}
		}()
	}

	// Start main server in a goroutine
	go func() {
		if cfg.TLS.Enabled {
			log.Printf("HTTPS server starting on %s", addr)
			if err := srv.ListenAndServeTLS(cfg.TLS.CertPath, cfg.TLS.KeyPath); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTPS server error: %v", err)
			}
		} else {
			log.Printf("HTTP server starting on %s", addr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTP server error: %v", err)
			}
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")

	// Graceful shutdown with 30 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown scheduler if running
	if sched != nil {
		sched.Shutdown(30 * time.Second)
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
	return nil
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func handleLoginPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/login.html")
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/dashboard.html")
}

func handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/oauth-callback.html")
}

// isHostAllowed validates a host against the allowed hosts list
func isHostAllowed(host string, allowedHosts []string) bool {
	// If no allowed hosts configured, allow all (backwards compatible)
	if len(allowedHosts) == 0 {
		return true
	}

	host = strings.ToLower(strings.TrimSpace(host))

	// Strip port from host for comparison
	hostWithoutPort := host
	if idx := strings.Index(host, ":"); idx != -1 {
		hostWithoutPort = host[:idx]
	}

	// Check against each allowed host
	for _, allowedHost := range allowedHosts {
		allowedHost = strings.ToLower(strings.TrimSpace(allowedHost))

		// Strip port from allowed host for comparison
		allowedHostWithoutPort := allowedHost
		if idx := strings.Index(allowedHost, ":"); idx != -1 {
			allowedHostWithoutPort = allowedHost[:idx]
		}

		// Match with or without port
		if host == allowedHost || hostWithoutPort == allowedHostWithoutPort {
			return true
		}
	}

	return false
}
