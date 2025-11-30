package scheduler

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"parsa/internal/domain/openfinance"
)

// AccountSyncJob implements the Job interface for syncing user accounts
type AccountSyncJob struct {
	userID      int64
	syncService *openfinance.AccountSyncService
}

// NewAccountSyncJob creates a new account sync job for a user
func NewAccountSyncJob(userID int64, syncService *openfinance.AccountSyncService) *AccountSyncJob {
	return &AccountSyncJob{
		userID:      userID,
		syncService: syncService,
	}
}

// Execute runs the account sync job
func (j *AccountSyncJob) Execute(ctx context.Context) error {
	log.Printf("Starting account sync for user %d", j.userID)

	result, err := j.syncService.SyncUserAccounts(ctx, j.userID)
	if err != nil {
		log.Printf("Account sync failed for user %d: %v", j.userID, err)
		return fmt.Errorf("sync failed: %w", err)
	}

	// Log results
	if len(result.Errors) > 0 {
		log.Printf("Account sync for user %d completed with errors: Created=%d, Updated=%d, Errors=%d",
			j.userID, result.Created, result.Updated, len(result.Errors))
		// Return error to mark for retry
		return fmt.Errorf("sync completed with %d errors", len(result.Errors))
	}

	log.Printf("Account sync for user %d completed successfully: Created=%d, Updated=%d",
		j.userID, result.Created, result.Updated)

	return nil
}

// UserID returns the user ID associated with this job
func (j *AccountSyncJob) UserID() string {
	return strconv.FormatInt(j.userID, 10)
}

// Description returns a human-readable description of the job
func (j *AccountSyncJob) Description() string {
	return fmt.Sprintf("Account sync for user %d", j.userID)
}

// UserSyncJob is a composite job that runs account sync followed by transaction sync.
// This ensures accounts are synced before transactions, avoiding race conditions.
type UserSyncJob struct {
	userID             int64
	accountSyncService *openfinance.AccountSyncService
	txSyncService      *openfinance.TransactionSyncService
}

// NewUserSyncJob creates a new composite sync job for a user
func NewUserSyncJob(userID int64, accountSyncService *openfinance.AccountSyncService, txSyncService *openfinance.TransactionSyncService) *UserSyncJob {
	return &UserSyncJob{
		userID:             userID,
		accountSyncService: accountSyncService,
		txSyncService:      txSyncService,
	}
}

// Execute runs account sync first, then transaction sync on success
func (j *UserSyncJob) Execute(ctx context.Context) error {
	// Run account sync first
	accountJob := NewAccountSyncJob(j.userID, j.accountSyncService)
	if err := accountJob.Execute(ctx); err != nil {
		return fmt.Errorf("account sync failed, skipping transaction sync: %w", err)
	}

	// Only run transaction sync if account sync succeeded
	txJob := NewTransactionSyncJob(j.userID, j.txSyncService)
	if err := txJob.Execute(ctx); err != nil {
		return fmt.Errorf("transaction sync failed: %w", err)
	}

	return nil
}

// UserID returns the user ID associated with this job
func (j *UserSyncJob) UserID() string {
	return strconv.FormatInt(j.userID, 10)
}

// Description returns a human-readable description of the job
func (j *UserSyncJob) Description() string {
	return fmt.Sprintf("Full sync (accounts + transactions) for user %d", j.userID)
}

// TransactionSyncJob implements the Job interface for syncing user transactions
type TransactionSyncJob struct {
	userID      int64
	syncService *openfinance.TransactionSyncService
}

// NewTransactionSyncJob creates a new transaction sync job for a user
func NewTransactionSyncJob(userID int64, syncService *openfinance.TransactionSyncService) *TransactionSyncJob {
	return &TransactionSyncJob{
		userID:      userID,
		syncService: syncService,
	}
}

// Execute runs the transaction sync job
func (j *TransactionSyncJob) Execute(ctx context.Context) error {
	log.Printf("Starting transaction sync for user %d", j.userID)

	result, err := j.syncService.SyncUserTransactions(ctx, j.userID)
	if err != nil {
		log.Printf("Transaction sync failed for user %d: %v", j.userID, err)
		return fmt.Errorf("sync failed: %w", err)
	}

	// Log results
	if len(result.Errors) > 0 {
		log.Printf("Transaction sync for user %d completed with errors: Created=%d, Updated=%d, Skipped=%d, Errors=%d",
			j.userID, result.Created, result.Updated, result.Skipped, len(result.Errors))
		// Return error to mark for retry
		return fmt.Errorf("sync completed with %d errors", len(result.Errors))
	}

	log.Printf("Transaction sync for user %d completed successfully: Created=%d, Updated=%d, Skipped=%d",
		j.userID, result.Created, result.Updated, result.Skipped)

	return nil
}

// UserID returns the user ID associated with this job
func (j *TransactionSyncJob) UserID() string {
	return strconv.FormatInt(j.userID, 10)
}

// Description returns a human-readable description of the job
func (j *TransactionSyncJob) Description() string {
	return fmt.Sprintf("Transaction sync for user %d", j.userID)
}
