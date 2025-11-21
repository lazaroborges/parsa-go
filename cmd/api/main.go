package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"parsa/internal/auth"
	"parsa/internal/config"
	"parsa/internal/database"
	"parsa/internal/handlers"
	"parsa/internal/middleware"
	"parsa/internal/scheduler"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Connect to database
	db, err := database.New(cfg.Database.ConnectionString())
	if err != nil {
		return err
	}
	defer db.Close()

	log.Println("Connected to database")

	// Initialize repositories
	userRepo := database.NewUserRepository(db)
	accountRepo := database.NewAccountRepository(db)
	transactionRepo := database.NewTransactionRepository(db)

	// Initialize auth components
	jwt := auth.NewJWT(cfg.JWT.Secret)
	googleOAuth := auth.NewGoogleOAuthProvider(
		cfg.OAuth.Google.ClientID,
		cfg.OAuth.Google.ClientSecret,
		cfg.OAuth.Google.RedirectURL,
	)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, googleOAuth, jwt)
	userHandler := handlers.NewUserHandler(userRepo)
	accountHandler := handlers.NewAccountHandler(accountRepo)
	transactionHandler := handlers.NewTransactionHandler(transactionRepo, accountRepo)

	// Initialize scheduler (if enabled)
	var sched *scheduler.Scheduler
	if cfg.Scheduler.Enabled {
		log.Println("Initializing scheduler...")
		schedulerConfig := scheduler.SchedulerConfig{
			ScheduleTimes: cfg.Scheduler.ScheduleTimes,
			WorkerCount:   cfg.Scheduler.WorkerCount,
			JobDelay:      cfg.Scheduler.JobDelay,
			QueueSize:     cfg.Scheduler.QueueSize,
			RunOnStartup:  cfg.Scheduler.RunOnStartup,
		}

		// Create Pierre Finance client (placeholder for now)
		var pierreClient scheduler.PierreFinanceClient = nil

		var err error
		sched, err = scheduler.NewScheduler(userRepo, pierreClient, schedulerConfig)
		if err != nil {
			return err
		}

		// Start scheduler in background
		sched.Start()
		log.Printf("Scheduler started with times: %v", cfg.Scheduler.ScheduleTimes)
	} else {
		log.Println("Scheduler is disabled")
	}

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
	mux.HandleFunc("/api/auth/oauth/url", authHandler.HandleAuthURL)
	mux.HandleFunc("/api/auth/oauth/callback", authHandler.HandleCallback)
	mux.HandleFunc("/health", handleHealth)

	// Protected routes - wrap with auth middleware
	authMiddleware := middleware.Auth(jwt)

	mux.Handle("/api/users/me", authMiddleware(http.HandlerFunc(userHandler.HandleGetMe)))
	mux.Handle("/api/accounts", authMiddleware(http.HandlerFunc(accountHandler.HandleListAccounts)))
	mux.Handle("/api/accounts/", authMiddleware(http.HandlerFunc(accountHandler.HandleGetAccount)))
	mux.Handle("/api/transactions", authMiddleware(http.HandlerFunc(transactionHandler.HandleListTransactions)))
	mux.Handle("/api/transactions/", authMiddleware(http.HandlerFunc(transactionHandler.HandleGetTransaction)))

	// Apply global middleware
	handler := middleware.Logging(middleware.CORS(mux))

	// Create server
	addr := cfg.Server.Host + ":" + cfg.Server.Port
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
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

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down HTTP server: %v", err)
	}

	// Shutdown scheduler if it was started
	if sched != nil {
		log.Println("Shutting down scheduler...")
		sched.Shutdown(30 * time.Second)
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
