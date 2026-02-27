// Package openfinance provides domain services for syncing financial data
package openfinance

import (
	"context"
	"errors"
	"fmt"
	"log"

	"parsa/internal/domain/account"
	"parsa/internal/domain/user"
	ofclient "parsa/internal/infrastructure/openfinance"
	"parsa/internal/models"
)

// ErrProviderUnauthorized is returned when the provider API rejects the key (401).
// Callers should stop the entire sync for this user.
var ErrProviderUnauthorized = errors.New("provider key unauthorized")

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
	client         ofclient.ClientInterface
	userRepo       user.Repository
	accountService *account.Service
	itemRepo       models.ItemRepository
}

// NewAccountSyncService creates a new account sync service
func NewAccountSyncService(
	client ofclient.ClientInterface,
	userRepo user.Repository,
	accountService *account.Service,
	itemRepo models.ItemRepository,
) *AccountSyncService {
	return &AccountSyncService{
		client:         client,
		userRepo:       userRepo,
		accountService: accountService,
		itemRepo:       itemRepo,
	}
}

// SyncUserAccountsWithData syncs accounts using pre-fetched account data.
func (s *AccountSyncService) SyncUserAccountsWithData(ctx context.Context, userID int64, accountResp *ofclient.AccountResponse) (*SyncResult, error) {
	if accountResp == nil {
		return &SyncResult{
			UserID:        userID,
			AccountsFound: 0,
			Errors:        []string{"account response is nil"},
		}, fmt.Errorf("account response is nil")
	}

	result := &SyncResult{
		UserID:        userID,
		AccountsFound: len(accountResp.Data),
		Errors:        []string{},
	}

	log.Printf("User %d: Syncing %d accounts", userID, result.AccountsFound)

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

// SyncUserAccounts syncs accounts for a specific user by fetching from API
func (s *AccountSyncService) SyncUserAccounts(ctx context.Context, userID int64) (*SyncResult, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return &SyncResult{UserID: userID, Errors: []string{}}, fmt.Errorf("failed to get user: %w", err)
	}

	if user.ProviderKey == nil || *user.ProviderKey == "" {
		return &SyncResult{UserID: userID, Errors: []string{}}, fmt.Errorf("user has no provider key")
	}

	accountResp, err := s.client.GetAccounts(ctx, *user.ProviderKey)
	if err != nil {
		return &SyncResult{UserID: userID, Errors: []string{}}, fmt.Errorf("failed to fetch accounts from API: %w", err)
	}

	return s.SyncUserAccountsWithData(ctx, userID, accountResp)
}

// syncAccount syncs a single account
func (s *AccountSyncService) syncAccount(ctx context.Context, userID int64, apiAccount ofclient.Account, result *SyncResult) error {
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

	// Find or create Item for this bank connection
	var itemID string
	if apiAccount.ItemID != "" {
		item, err := s.itemRepo.FindOrCreate(ctx, apiAccount.ItemID, userID)
		if err != nil {
			return fmt.Errorf("failed to find/create item: %w", err)
		}
		itemID = item.ID
		log.Printf("User %d: Using item %s for account %s", userID, itemID, apiAccount.AccountID)
	}

	// Check if account exists to determine if this is create or update
	exists, err := s.accountService.AccountExists(ctx, apiAccount.AccountID)
	if err != nil {
		return fmt.Errorf("failed to check account existence: %w", err)
	}

	// Prepare upsert parameters
	params := account.UpsertParams{
		ID:                apiAccount.AccountID,
		UserID:            userID,
		ItemID:            itemID,
		Name:              apiAccount.AccountName,
		AccountType:       apiAccount.AccountType,
		Currency:          apiAccount.AccountCurrencyCode,
		Balance:           balance,
		ProviderCreatedAt: createdAt,
		ProviderUpdatedAt: updatedAt,
	}

	// Set subtype if available
	if apiAccount.AccountSubtype != "" {
		params.Subtype = &apiAccount.AccountSubtype
	}

	// Upsert the account
	_, err = s.accountService.UpsertAccount(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to upsert account: %w", err)
	}

	if exists {
		result.Updated++
		log.Printf("User %d: Updated account %s (%s)", userID, apiAccount.AccountName, apiAccount.AccountID)
	} else {
		result.Created++
		log.Printf("User %d: Created account %s (%s)", userID, apiAccount.AccountName, apiAccount.AccountID)
	}

	return nil
}
