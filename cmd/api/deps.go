package main

import (
	"context"
	"fmt"
	"log"

	"parsa/internal/domain/account"
	"parsa/internal/domain/cousinrule"
	"parsa/internal/domain/notification"
	"parsa/internal/domain/openfinance"
	"parsa/internal/infrastructure/crypto"
	fcmclient "parsa/internal/infrastructure/firebase"
	ofclient "parsa/internal/infrastructure/openfinance"
	"parsa/internal/infrastructure/postgres"
	"parsa/internal/infrastructure/postgres/listener"
	httphandlers "parsa/internal/interfaces/http"
	"parsa/internal/shared/auth"
	"parsa/internal/shared/config"
	"parsa/internal/shared/messages"
)

// Dependencies holds all initialized application components.
type Dependencies struct {
	DB *postgres.DB

	// Handlers
	AuthHandler         *httphandlers.AuthHandler
	UserHandler         *httphandlers.UserHandler
	AccountHandler      *httphandlers.AccountHandler
	TransactionHandler  *httphandlers.TransactionHandler
	TagHandler          *httphandlers.TagHandler
	CousinRuleHandler   *httphandlers.CousinRuleHandler
	NotificationHandler *httphandlers.NotificationHandler

	// Auth
	JWT *auth.JWT

	// Sync services (for scheduler)
	AccountSyncService     *openfinance.AccountSyncService
	TransactionSyncService *openfinance.TransactionSyncService
	BillSyncService        *openfinance.BillSyncService

	// Repositories (for scheduler job provider)
	UserRepo *postgres.UserRepository

	// Background listeners
	CousinListener *listener.CousinListener
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

	// Load notification message texts (needed for sync services)
	msgs, err := messages.Load()
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to load notification messages: %w", err)
	}

	// Initialize Open Finance client
	ofClient := ofclient.NewClient()

	// Initialize notification components (needed for account sync provider-key-cleared notification)
	notificationRepo := postgres.NewNotificationRepository(db)
	var messenger notification.Messenger
	if cfg.Firebase.CredentialsFile != "" {
		deactivator := fcmclient.TokenDeactivator(notificationRepo.DeactivateToken)
		fbClient, err := fcmclient.NewClient(context.Background(), cfg.Firebase.CredentialsFile, deactivator)
		if err != nil {
			log.Printf("Warning: Failed to initialize Firebase: %v (notifications disabled)", err)
		} else {
			log.Println("Firebase Cloud Messaging initialized")
			messenger = fbClient
		}
	} else {
		log.Println("FIREBASE_CREDENTIALS_FILE not set, notifications disabled")
	}
	notificationService := notification.NewService(notificationRepo, messenger)

	// Initialize sync services (account sync needs notification service for provider_key_cleared)
	accountSyncService := openfinance.NewAccountSyncService(ofClient, userRepo, accountService, itemRepo, notificationService, msgs)
	transactionSyncService := openfinance.NewTransactionSyncService(ofClient, userRepo, accountService, accountRepo, transactionRepo, creditCardDataRepo, bankRepo, cfg.OpenFinance.TransactionSyncStartDate)
	billSyncService := openfinance.NewBillSyncService(ofClient, userRepo, accountService, accountRepo, billRepo, transactionRepo)

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

	notificationHandler := httphandlers.NewNotificationHandler(notificationService)

	userHandler := httphandlers.NewUserHandler(userRepo, accountRepo, ofClient, accountSyncService, transactionSyncService, billSyncService, notificationService, msgs)
	accountHandler := httphandlers.NewAccountHandler(accountService)
	tagRepo := postgres.NewTagRepository(db)
	tagHandler := httphandlers.NewTagHandler(tagRepo)

	// Initialize cousin rule components
	cousinRuleRepo := postgres.NewCousinRuleRepository(db)
	cousinRuleService := cousinrule.NewService(cousinRuleRepo, transactionRepo)
	cousinRuleHandler := httphandlers.NewCousinRuleHandler(cousinRuleService)

	// Initialize transaction handler with cousin rule repo for dont_ask_again lookups
	transactionHandler := httphandlers.NewTransactionHandler(transactionRepo, accountRepo, cousinRuleRepo)

	// Initialize and start cousin notification listener
	cousinListener := listener.NewCousinListener(cfg.Database.ConnectionString(), cousinRuleRepo, db.DB)
	cousinListener.Start(context.Background())

	return &Dependencies{
		DB:                     db,
		AuthHandler:            authHandler,
		UserHandler:            userHandler,
		AccountHandler:         accountHandler,
		TransactionHandler:     transactionHandler,
		TagHandler:             tagHandler,
		CousinRuleHandler:      cousinRuleHandler,
		NotificationHandler:    notificationHandler,
		JWT:                    jwt,
		AccountSyncService:     accountSyncService,
		TransactionSyncService: transactionSyncService,
		BillSyncService:        billSyncService,
		UserRepo:               userRepo,
		CousinListener:         cousinListener,
	}, nil
}

// Close releases all resources held by dependencies.
func (d *Dependencies) Close() {
	// Stop listeners first
	if d.CousinListener != nil {
		d.CousinListener.Stop()
	}

	if d.DB != nil {
		d.DB.Close()
	}
}
