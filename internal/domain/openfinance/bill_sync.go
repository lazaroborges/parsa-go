package openfinance

import (
	"context"
	"fmt"
	"log"

	"parsa/internal/domain/account"
	"parsa/internal/domain/bill"
	"parsa/internal/domain/user"
	ofclient "parsa/internal/infrastructure/openfinance"
)

// BillSyncResult contains the results of a bill sync operation
type BillSyncResult struct {
	UserID     int64
	BillsFound int
	Created    int
	Updated    int
	Skipped    int // Bills that couldn't be matched to an account
	Errors     []string
}

// BillSyncService handles syncing past due credit card bills from the Open Finance API
type BillSyncService struct {
	client         ofclient.ClientInterface
	userRepo       user.Repository
	accountService *account.Service
	accountRepo    account.Repository
	billRepo       bill.Repository
}

// NewBillSyncService creates a new bill sync service
func NewBillSyncService(
	client ofclient.ClientInterface,
	userRepo user.Repository,
	accountService *account.Service,
	accountRepo account.Repository,
	billRepo bill.Repository,
) *BillSyncService {
	return &BillSyncService{
		client:         client,
		userRepo:       userRepo,
		accountService: accountService,
		accountRepo:    accountRepo,
		billRepo:       billRepo,
	}
}

// SyncUserBills syncs all past due credit card bills for a specific user
func (s *BillSyncService) SyncUserBills(ctx context.Context, userID int64) (*BillSyncResult, error) {
	result := &BillSyncResult{
		UserID: userID,
		Errors: []string{},
	}

	// Get user's API key
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user.ProviderKey == nil || *user.ProviderKey == "" {
		return nil, fmt.Errorf("user has no provider API key configured")
	}

	// Fetch past due bills from provider
	billResp, err := s.client.GetBills(ctx, *user.ProviderKey)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch bills from provider: %w", err)
	}

	result.BillsFound = len(billResp.Data)
	log.Printf("Fetched %d past due bills for user %d", result.BillsFound, userID)

	// Build a cache of accounts for matching
	// Key: "name|account_type|subtype" -> account
	accountCache := make(map[string]*account.Account)
	accounts, err := s.accountService.ListAccountsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user accounts: %w", err)
	}
	for i := range accounts {
		key := fmt.Sprintf("%s|%s|%s", accounts[i].Name, accounts[i].AccountType, accounts[i].Subtype)
		accountCache[key] = accounts[i]
	}

	// Also build account ID map for direct lookups
	accountIDMap := make(map[string]*account.Account)
	for i := range accounts {
		accountIDMap[accounts[i].ID] = accounts[i]
	}

	// Process each bill
	for _, apiBill := range billResp.Data {
		if err := s.processBill(ctx, userID, &apiBill, accountCache, accountIDMap, result); err != nil {
			errMsg := fmt.Sprintf("failed to process bill %s: %v", apiBill.ID, err)
			result.Errors = append(result.Errors, errMsg)
			log.Printf("Error: %s", errMsg)
		}
	}

	log.Printf("Bill sync completed for user %d: found=%d, created=%d, updated=%d, skipped=%d, errors=%d",
		userID, result.BillsFound, result.Created, result.Updated, result.Skipped, len(result.Errors))

	return result, nil
}

// processBill processes a single past due credit card bill from the API
func (s *BillSyncService) processBill(
	ctx context.Context,
	userID int64,
	apiBill *ofclient.Bill,
	accountCache map[string]*account.Account,
	accountIDMap map[string]*account.Account,
	result *BillSyncResult,
) error {
	var matchedAccount *account.Account

	// First try direct account ID match
	if apiBill.AccountID != "" {
		if acc, ok := accountIDMap[apiBill.AccountID]; ok {
			matchedAccount = acc
		}
	}

	// If no direct match, try matching by account name/type/subtype
	if matchedAccount == nil && apiBill.AccountName != "" {
		matchKey := fmt.Sprintf("%s|%s|%s", apiBill.AccountName, apiBill.AccountType, apiBill.AccountSubtype)
		if acc, ok := accountCache[matchKey]; ok {
			matchedAccount = acc
		}
	}

	// If still no match, try database lookup
	if matchedAccount == nil && apiBill.AccountName != "" {
		acc, err := s.accountService.FindAccountByMatch(ctx, userID, apiBill.AccountName, apiBill.AccountType, apiBill.AccountSubtype)
		if err != nil {
			return fmt.Errorf("failed to find account: %w", err)
		}
		matchedAccount = acc
	}

	if matchedAccount == nil {
		log.Printf("Skipping bill %s: no matching account found for id=%s, name=%s, type=%s, subtype=%s",
			apiBill.ID, apiBill.AccountID, apiBill.AccountName, apiBill.AccountType, apiBill.AccountSubtype)
		result.Skipped++
		return nil
	}

	// Parse total amount
	totalAmount, err := apiBill.GetTotalAmount()
	if err != nil {
		return fmt.Errorf("failed to parse total amount: %w", err)
	}

	// Parse due date
	dueDate, err := apiBill.GetDueDate()
	if err != nil {
		return fmt.Errorf("failed to parse due date: %w", err)
	}
	if dueDate == nil {
		return fmt.Errorf("due date is required")
	}

	// Parse timestamps
	createdAt, _ := apiBill.GetCreatedAt()
	updatedAt, _ := apiBill.GetUpdatedAt()

	// Check if bill already exists
	existingBill, err := s.billRepo.GetByID(ctx, apiBill.ID)
	if err != nil {
		return fmt.Errorf("failed to check existing bill: %w", err)
	}

	// Prepare upsert params
	upsertParams := bill.UpsertParams{
		ID:                apiBill.ID,
		AccountID:         matchedAccount.ID,
		DueDate:           *dueDate,
		TotalAmount:       totalAmount,
		ProviderCreatedAt: createdAt,
		ProviderUpdatedAt: updatedAt,
	}

	// Upsert bill
	_, err = s.billRepo.Upsert(ctx, upsertParams)
	if err != nil {
		return fmt.Errorf("failed to upsert bill: %w", err)
	}

	if existingBill == nil {
		result.Created++
	} else {
		result.Updated++
	}

	return nil
}

// SyncAllUsersBills syncs past due bills for all users with provider keys
func (s *BillSyncService) SyncAllUsersBills(ctx context.Context) ([]*BillSyncResult, error) {
	users, err := s.userRepo.ListUsersWithProviderKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list users with provider keys: %w", err)
	}

	var results []*BillSyncResult
	for _, user := range users {
		result, err := s.SyncUserBills(ctx, user.ID)
		if err != nil {
			log.Printf("Failed to sync bills for user %d: %v", user.ID, err)
			results = append(results, &BillSyncResult{
				UserID: user.ID,
				Errors: []string{err.Error()},
			})
			continue
		}
		results = append(results, result)
	}

	return results, nil
}
