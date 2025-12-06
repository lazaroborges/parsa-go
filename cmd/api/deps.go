package main

import (
	"log"

	"parsa/internal/domain/account"
	"parsa/internal/domain/openfinance"
	"parsa/internal/infrastructure/crypto"
	ofclient "parsa/internal/infrastructure/openfinance"
	"parsa/internal/infrastructure/postgres"
	httphandlers "parsa/internal/interfaces/http"
	"parsa/internal/shared/auth"
	"parsa/internal/shared/config"
)

// Dependencies holds all initialized application components.
type Dependencies struct {
	DB *postgres.DB

	// Handlers
	AuthHandler        *httphandlers.AuthHandler
	UserHandler        *httphandlers.UserHandler
	AccountHandler     *httphandlers.AccountHandler
	TransactionHandler *httphandlers.TransactionHandler

	// Auth
	JWT *auth.JWT

	// Sync services (for scheduler)
	AccountSyncService     *openfinance.AccountSyncService
	TransactionSyncService *openfinance.TransactionSyncService
	BillSyncService        *openfinance.BillSyncService

	// Repositories (for scheduler job provider)
	UserRepo *postgres.UserRepository
}

// NewDependencies initializes all application dependencies.
func NewDependencies(cfg *config.Config) (*Dependencies, error) {
	// Connect to database
	db, err := postgres.New(cfg.Database.ConnectionString())
	if err != nil {
		return nil, err
	}
	log.Println("Connected to database")

	// Initialize encryptor
	encryptor, err := crypto.NewEncryptor(cfg.Encryption.Key)
	if err != nil {
		return nil, err
	}

	// Initialize repositories
	userRepo := postgres.NewUserRepository(db, encryptor)
	transactionRepo := postgres.NewTransactionRepository(db)
	itemRepo := postgres.NewItemRepository(db)
	creditCardDataRepo := postgres.NewCreditCardDataRepository(db)
	bankRepo := postgres.NewBankRepository(db)
	accountRepo := postgres.NewAccountRepository(db)

	// Initialize domain services
	accountService := account.NewService(accountRepo)

	// Initialize bill repository
	billRepo := postgres.NewBillRepository(db)

	// Initialize Open Finance client and sync services
	ofClient := ofclient.NewClient()
	accountSyncService := openfinance.NewAccountSyncService(ofClient, userRepo, accountService, itemRepo)
	transactionSyncService := openfinance.NewTransactionSyncService(ofClient, userRepo, accountService, accountRepo, transactionRepo, creditCardDataRepo, bankRepo)
	billSyncService := openfinance.NewBillSyncService(ofClient, userRepo, accountService, accountRepo, billRepo)

	// Initialize auth components
	jwt := auth.NewJWT(cfg.JWT.Secret)
	googleOAuth := auth.NewGoogleOAuthProvider(
		cfg.OAuth.Google.ClientID,
		cfg.OAuth.Google.ClientSecret,
		cfg.OAuth.Google.WebCallbackURL,
	)

	// Initialize handlers
	authHandler := httphandlers.NewAuthHandler(userRepo, googleOAuth, jwt, cfg.OAuth.Google.MobileCallbackURL, cfg.OAuth.Google.WebCallbackURL)

	// Initialize Apple OAuth if configured
	if cfg.OAuth.Apple.PrivateKeyPath != "" {
		appleOAuth, err := auth.NewAppleOAuthProvider(
			cfg.OAuth.Apple.TeamID,
			cfg.OAuth.Apple.KeyID,
			cfg.OAuth.Apple.ClientID,
			cfg.OAuth.Apple.PrivateKeyPath,
			cfg.OAuth.Apple.MobileCallbackURL,
		)
		if err != nil {
			log.Printf("Warning: Failed to initialize Apple OAuth: %v", err)
		} else {
			authHandler.SetAppleOAuthProvider(appleOAuth, cfg.OAuth.Apple.MobileCallbackURL)
		}
	}

	userHandler := httphandlers.NewUserHandler(userRepo, accountRepo, ofClient, accountSyncService, transactionSyncService, billSyncService)
	accountHandler := httphandlers.NewAccountHandler(accountService)
	transactionHandler := httphandlers.NewTransactionHandler(transactionRepo, accountRepo)

	return &Dependencies{
		DB:                     db,
		AuthHandler:            authHandler,
		UserHandler:            userHandler,
		AccountHandler:         accountHandler,
		TransactionHandler:     transactionHandler,
		JWT:                    jwt,
		AccountSyncService:     accountSyncService,
		TransactionSyncService: transactionSyncService,
		BillSyncService:        billSyncService,
		UserRepo:               userRepo,
	}, nil
}

// Close releases all resources held by dependencies.
func (d *Dependencies) Close() {
	if d.DB != nil {
		d.DB.Close()
	}
}
