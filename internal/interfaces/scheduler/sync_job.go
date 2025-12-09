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

// UserSyncJob is a composite job that runs account sync, transaction sync, then bill sync.
// This ensures accounts are synced before transactions and bills, avoiding race conditions.
type UserSyncJob struct {
	userID             int64
	accountSyncService *openfinance.AccountSyncService
	txSyncService      *openfinance.TransactionSyncService
	billSyncService    *openfinance.BillSyncService
}

// NewUserSyncJob creates a new composite sync job for a user
func NewUserSyncJob(userID int64, accountSyncService *openfinance.AccountSyncService, txSyncService *openfinance.TransactionSyncService, billSyncService *openfinance.BillSyncService) *UserSyncJob {
	return &UserSyncJob{
		userID:             userID,
		accountSyncService: accountSyncService,
		txSyncService:      txSyncService,
		billSyncService:    billSyncService,
	}
}

// Execute runs account sync first, then transaction sync, then bill sync on success
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

	// Run bill sync after transaction sync
	billJob := NewBillSyncJob(j.userID, j.billSyncService)
	if err := billJob.Execute(ctx); err != nil {
		return fmt.Errorf("bill sync failed: %w", err)
	}

	return nil
}

// UserID returns the user ID associated with this job
func (j *UserSyncJob) UserID() string {
	return strconv.FormatInt(j.userID, 10)
}

// Description returns a human-readable description of the job
func (j *UserSyncJob) Description() string {
	return fmt.Sprintf("Full sync (accounts + transactions + bills) for user %d", j.userID)
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

// BillSyncJob implements the Job interface for syncing user bills
type BillSyncJob struct {
	userID      int64
	syncService *openfinance.BillSyncService
}

// NewBillSyncJob creates a new bill sync job for a user
func NewBillSyncJob(userID int64, syncService *openfinance.BillSyncService) *BillSyncJob {
	return &BillSyncJob{
		userID:      userID,
		syncService: syncService,
	}
}

// Execute runs the bill sync job
func (j *BillSyncJob) Execute(ctx context.Context) error {
	log.Printf("Starting bill sync for user %d", j.userID)

	result, err := j.syncService.SyncUserBills(ctx, j.userID)
	if err != nil {
		log.Printf("Bill sync failed for user %d: %v", j.userID, err)
		return fmt.Errorf("sync failed: %w", err)
	}

	// Log results
	if len(result.Errors) > 0 {
		log.Printf("Bill sync for user %d completed with errors: Created=%d, Updated=%d, Skipped=%d, Errors=%d",
			j.userID, result.Created, result.Updated, result.Skipped, len(result.Errors))
		// Return error to mark for retry
		return fmt.Errorf("sync completed with %d errors", len(result.Errors))
	}

	log.Printf("Bill sync for user %d completed successfully: Created=%d, Updated=%d, Skipped=%d",
		j.userID, result.Created, result.Updated, result.Skipped)

	return nil
}

// UserID returns the user ID associated with this job
func (j *BillSyncJob) UserID() string {
	return strconv.FormatInt(j.userID, 10)
}

// Description returns a human-readable description of the job
func (j *BillSyncJob) Description() string {
	return fmt.Sprintf("Bill sync for user %d", j.userID)
}
