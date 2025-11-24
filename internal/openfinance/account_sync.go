package openfinance

import (
	"context"
	"fmt"
	"log"

	"parsa/internal/database"
	"parsa/internal/models"
)

// SyncResult contains the results of a sync operation
type SyncResult struct {
	UserID        int64
	AccountsFound int
	Created       int
	Updated       int
	Errors        []string
}

// AccountSyncService handles syncing accounts from the Open Finance API
type AccountSyncService struct {
	client      *Client
	userRepo    *database.UserRepository
	accountRepo *database.AccountRepository
	bankRepo    *database.BankRepository
}

// NewAccountSyncService creates a new account sync service
func NewAccountSyncService(
	client *Client,
	userRepo *database.UserRepository,
	accountRepo *database.AccountRepository,
	bankRepo *database.BankRepository,
) *AccountSyncService {
	return &AccountSyncService{
		client:      client,
		userRepo:    userRepo,
		accountRepo: accountRepo,
		bankRepo:    bankRepo,
	}
}

// SyncUserAccounts syncs accounts for a specific user
func (s *AccountSyncService) SyncUserAccounts(ctx context.Context, userID int64) (*SyncResult, error) {
	result := &SyncResult{
		UserID: userID,
		Errors: []string{},
	}

	// Fetch user to get their provider key
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return result, fmt.Errorf("failed to get user: %w", err)
	}

	if user.ProviderKey == "" {
		return result, fmt.Errorf("user has no provider key")
	}

	// Fetch accounts from Open Finance API
	accountResp, err := s.client.GetAccounts(ctx, user.ProviderKey)
	if err != nil {
		return result, fmt.Errorf("failed to fetch accounts from API: %w", err)
	}

	result.AccountsFound = len(accountResp.Data)
	log.Printf("User %d: Found %d accounts from Open Finance API", userID, result.AccountsFound)

	// Process each account
	for _, apiAccount := range accountResp.Data {
		if err := s.syncAccount(ctx, userID, apiAccount, result); err != nil {
			errMsg := fmt.Sprintf("failed to sync account %s: %v", apiAccount.AccountID, err)
			result.Errors = append(result.Errors, errMsg)
			log.Printf("User %d: %s", userID, errMsg)
		}
	}

	log.Printf("User %d: Sync complete - Created: %d, Updated: %d, Errors: %d",
		userID, result.Created, result.Updated, len(result.Errors))

	return result, nil
}

// syncAccount syncs a single account
func (s *AccountSyncService) syncAccount(ctx context.Context, userID int64, apiAccount Account, result *SyncResult) error {
	// Parse balance from string
	balance, err := apiAccount.GetBalance()
	if err != nil {
		return fmt.Errorf("failed to parse balance: %w", err)
	}

	// Parse timestamps from API response
	createdAt, err := apiAccount.GetCreatedAt()
	if err != nil {
		return fmt.Errorf("failed to parse createdAt: %w", err)
	}
	updatedAt, err := apiAccount.GetUpdatedAt()
	if err != nil {
		return fmt.Errorf("failed to parse updatedAt: %w", err)
	}

	// Find or create bank using the 'name' field from API response
	var bankID *int64
	if apiAccount.AccountName != "" {
		// Use the account name as both bank name and connector
		// The connector will be the unique identifier for the bank
		bank, err := s.bankRepo.FindOrCreateByConnector(ctx, apiAccount.AccountName, apiAccount.AccountName)
		if err != nil {
			return fmt.Errorf("failed to find/create bank: %w", err)
		}
		bankID = &bank.ID
	}

	// Prepare upsert parameters
	params := models.UpsertAccountParams{
		UserID:            userID,
		Name:              apiAccount.AccountName,
		AccountType:       apiAccount.AccountType,
		Currency:          apiAccount.AccountCurrencyCode,
		Balance:           balance,
		BankID:            bankID,
		ProviderID:        apiAccount.AccountID,
		ProviderCreatedAt: createdAt,
		ProviderUpdatedAt: updatedAt,
	}

	// Set subtype if available
	if apiAccount.AccountSubtype != "" {
		params.Subtype = &apiAccount.AccountSubtype
	}

	// Check if account exists to determine if this is create or update
	existingAccount, err := s.accountRepo.GetByProviderID(ctx, userID, apiAccount.AccountID)
	isUpdate := err == nil && existingAccount != nil

	// Upsert the account
	_, err = s.accountRepo.UpsertByProviderID(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to upsert account: %w", err)
	}

	if isUpdate {
		result.Updated++
		log.Printf("User %d: Updated account %s (%s)", userID, apiAccount.AccountName, apiAccount.AccountID)
	} else {
		result.Created++
		log.Printf("User %d: Created account %s (%s)", userID, apiAccount.AccountName, apiAccount.AccountID)
	}

	return nil
}
