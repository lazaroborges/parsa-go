package openfinance

import (
	"context"
	"fmt"
	"log"

	ofclient "parsa/internal/infrastructure/openfinance"
	"parsa/internal/infrastructure/postgres"
	"parsa/internal/domain/account"
	"parsa/internal/domain/transaction"
	"parsa/internal/models"
)

// TransactionSyncResult contains the results of a transaction sync operation
type TransactionSyncResult struct {
	UserID            int64
	TransactionsFound int
	Created           int
	Updated           int
	Skipped           int // Transactions that couldn't be matched to an account
	Errors            []string
}

// TransactionSyncService handles syncing transactions from the Open Finance API
type TransactionSyncService struct {
	client             *ofclient.Client
	userRepo           *postgres.UserRepository
	accountService     *account.Service
	accountRepo        *postgres.AccountRepository
	transactionRepo    *postgres.TransactionRepository
	creditCardDataRepo *postgres.CreditCardDataRepository
	bankRepo           *postgres.BankRepository
}

// NewTransactionSyncService creates a new transaction sync service
func NewTransactionSyncService(
	client *ofclient.Client,
	userRepo *postgres.UserRepository,
	accountService *account.Service,
	accountRepo *postgres.AccountRepository,
	transactionRepo *postgres.TransactionRepository,
	creditCardDataRepo *postgres.CreditCardDataRepository,
	bankRepo *postgres.BankRepository,
) *TransactionSyncService {
	return &TransactionSyncService{
		client:             client,
		userRepo:           userRepo,
		accountService:     accountService,
		accountRepo:        accountRepo,
		transactionRepo:    transactionRepo,
		creditCardDataRepo: creditCardDataRepo,
		bankRepo:           bankRepo,
	}
}

// SyncUserTransactions syncs all transactions for a specific user
func (s *TransactionSyncService) SyncUserTransactions(ctx context.Context, userID int64) (*TransactionSyncResult, error) {
	result := &TransactionSyncResult{
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

	// Fetch transactions from provider
	txResp, err := s.client.GetTransactions(ctx, *user.ProviderKey)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions from provider: %w", err)
	}

	result.TransactionsFound = len(txResp.Data)
	log.Printf("Fetched %d transactions for user %d", result.TransactionsFound, userID)

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

	// Process each transaction
	for _, apiTx := range txResp.Data {
		if err := s.processTransaction(ctx, userID, &apiTx, accountCache, result); err != nil {
			errMsg := fmt.Sprintf("failed to process transaction %s: %v", apiTx.ID, err)
			result.Errors = append(result.Errors, errMsg)
			log.Printf("Error: %s", errMsg)
		}
	}

	log.Printf("Transaction sync completed for user %d: found=%d, created=%d, updated=%d, skipped=%d, errors=%d",
		userID, result.TransactionsFound, result.Created, result.Updated, result.Skipped, len(result.Errors))

	return result, nil
}

// processTransaction processes a single transaction from the API
func (s *TransactionSyncService) processTransaction(
	ctx context.Context,
	userID int64,
	apiTx *ofclient.Transaction,
	accountCache map[string]*account.Account,
	result *TransactionSyncResult,
) error {
	// Match transaction to account using: account_name -> name, account_type -> account_type, account_subtype -> subtype
	matchKey := fmt.Sprintf("%s|%s|%s", apiTx.AccountName, apiTx.AccountType, apiTx.AccountSubtype)
	account, found := accountCache[matchKey]

	if !found {
		// Try to find in database (in case cache is stale)
		var err error
		account, err = s.accountService.FindAccountByMatch(ctx, userID, apiTx.AccountName, apiTx.AccountType, apiTx.AccountSubtype)
		if err != nil {
			return fmt.Errorf("failed to find account: %w", err)
		}
		if account == nil {
			log.Printf("Skipping transaction %s: no matching account found for name=%s, type=%s, subtype=%s",
				apiTx.ID, apiTx.AccountName, apiTx.AccountType, apiTx.AccountSubtype)
			result.Skipped++
			return nil
		}
		// Update cache
		accountCache[matchKey] = account
	}

	// If account has no bank assigned and we have item_bank_name, assign it
	// Note: We can't verify ownership here as this is a sync operation from the provider
	// The account already belongs to the user by virtue of being in their provider data
	if account.BankID == 0 && apiTx.ItemBankName != "" {
		bank, err := s.bankRepo.FindOrCreateByName(ctx, apiTx.ItemBankName)
		if err != nil {
			log.Printf("Warning: failed to find/create bank '%s': %v", apiTx.ItemBankName, err)
		} else {
			if err := s.accountRepo.UpdateBankID(ctx, account.ID, bank.ID); err != nil {
				log.Printf("Warning: failed to update account %s with bank_id %d: %v", account.ID, bank.ID, err)
			} else {
				account.BankID = bank.ID // Update cache
				log.Printf("Assigned bank '%s' (id=%d) to account %s", bank.Name, bank.ID, account.ID)
			}
		}
	}

	// Parse amount
	amount, err := apiTx.GetAmount()
	if err != nil {
		return fmt.Errorf("failed to parse amount: %w", err)
	}

	// Parse transaction date
	txDate, err := apiTx.GetDate()
	if err != nil {
		return fmt.Errorf("failed to parse date: %w", err)
	}
	if txDate == nil {
		return fmt.Errorf("transaction date is required")
	}

	// Check if transaction already exists
	existingTx, err := s.transactionRepo.GetByID(ctx, apiTx.ID)
	if err != nil {
		return fmt.Errorf("failed to check existing transaction: %w", err)
	}

	// Prepare upsert params
	upsertParams := transaction.UpsertTransactionParams{
		ID:              apiTx.ID,
		AccountID:       account.ID,
		Amount:          amount,
		Description:     apiTx.Description,
		Category:        apiTx.Category,
		TransactionDate: *txDate,
		Type:            apiTx.Type,
		Status:          apiTx.Status,
	}

	// Upsert transaction
	_, err = s.transactionRepo.Upsert(ctx, upsertParams)
	if err != nil {
		return fmt.Errorf("failed to upsert transaction: %w", err)
	}

	if existingTx == nil {
		result.Created++
	} else {
		result.Updated++
	}

	// Process credit card data if present
	if apiTx.CreditCardData != nil {
		if err := s.processCreditCardData(ctx, apiTx.ID, apiTx.CreditCardData); err != nil {
			return fmt.Errorf("failed to process credit card data: %w", err)
		}
	}

	return nil
}

// processCreditCardData processes credit card installment data for a transaction
func (s *TransactionSyncService) processCreditCardData(
	ctx context.Context,
	transactionID string,
	ccData *ofclient.TransactionCreditData,
) error {
	purchaseDate, err := ccData.GetPurchaseDate()
	if err != nil {
		return fmt.Errorf("failed to parse purchase date: %w", err)
	}
	if purchaseDate == nil {
		return nil // No purchase date, skip
	}

	installmentNum, err := ccData.GetInstallmentNumber()
	if err != nil {
		return fmt.Errorf("failed to parse installment number: %w", err)
	}

	totalInstallments, err := ccData.GetTotalInstallments()
	if err != nil {
		return fmt.Errorf("failed to parse total installments: %w", err)
	}

	// Skip if no installment data
	if installmentNum == 0 || totalInstallments == 0 {
		return nil
	}

	params := models.CreateCreditCardDataParams{
		PurchaseDate:      *purchaseDate,
		InstallmentNumber: installmentNum,
		TotalInstallments: totalInstallments,
	}

	_, err = s.creditCardDataRepo.Upsert(ctx, transactionID, params)
	if err != nil {
		return fmt.Errorf("failed to upsert credit card data: %w", err)
	}

	return nil
}

// SyncAllUsersTransactions syncs transactions for all users with provider keys
func (s *TransactionSyncService) SyncAllUsersTransactions(ctx context.Context) ([]*TransactionSyncResult, error) {
	users, err := s.userRepo.ListUsersWithProviderKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list users with provider keys: %w", err)
	}

	var results []*TransactionSyncResult
	for _, user := range users {
		result, err := s.SyncUserTransactions(ctx, user.ID)
		if err != nil {
			log.Printf("Failed to sync transactions for user %d: %v", user.ID, err)
			results = append(results, &TransactionSyncResult{
				UserID: user.ID,
				Errors: []string{err.Error()},
			})
			continue
		}
		results = append(results, result)
	}

	return results, nil
}
