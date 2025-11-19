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

	// Create router
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/auth/google/url", authHandler.HandleAuthURL)
	mux.HandleFunc("/auth/google/callback", authHandler.HandleCallback)
	mux.HandleFunc("/health", handleHealth)

	// Protected routes - wrap with auth middleware
	authMiddleware := middleware.Auth(jwt)

	mux.Handle("/users/me", authMiddleware(http.HandlerFunc(userHandler.HandleGetMe)))
	mux.Handle("/accounts", authMiddleware(http.HandlerFunc(accountHandler.HandleListAccounts)))
	mux.Handle("/accounts/", authMiddleware(http.HandlerFunc(accountHandler.HandleGetAccount)))
	mux.Handle("/transactions", authMiddleware(http.HandlerFunc(transactionHandler.HandleListTransactions)))
	mux.Handle("/transactions/", authMiddleware(http.HandlerFunc(transactionHandler.HandleGetTransaction)))

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

	if err := srv.Shutdown(ctx); err != nil {
		return err
	}

	log.Println("Server stopped")
	return nil
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
