package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"

	"parsa/internal/domain/openfinance"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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
	ctx, span := jobTracer.Start(ctx, "sync.accounts",
		trace.WithAttributes(attribute.Int64("user.id", j.userID)),
	)
	defer span.End()

	log.Printf("Starting account sync for user %d", j.userID)

	result, err := j.syncService.SyncUserAccounts(ctx, j.userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if errors.Is(err, openfinance.ErrProviderUnauthorized) {
			log.Printf("User %d: Provider key invalid (401) — account sync aborted, key cleared", j.userID)
			return fmt.Errorf("sync aborted: %w", err)
		}
		log.Printf("Account sync failed for user %d: %v", j.userID, err)
		return fmt.Errorf("sync failed: %w", err)
	}

	span.SetAttributes(
		attribute.Int("sync.created", result.Created),
		attribute.Int("sync.updated", result.Updated),
		attribute.Int("sync.errors", len(result.Errors)),
	)

	// Log results
	if len(result.Errors) > 0 {
		span.SetStatus(codes.Error, fmt.Sprintf("sync completed with %d errors", len(result.Errors)))
		log.Printf("Account sync for user %d completed with errors: Created=%d, Updated=%d, Errors=%d",
			j.userID, result.Created, result.Updated, len(result.Errors))
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

// Execute runs account sync first, then transaction sync, then bill sync on success.
// Transaction sync uses full history if new accounts were created, otherwise last 7 days.
func (j *UserSyncJob) Execute(ctx context.Context) error {
	ctx, span := jobTracer.Start(ctx, "sync.user",
		trace.WithAttributes(attribute.Int64("user.id", j.userID)),
	)
	defer span.End()

	log.Printf("Starting full sync for user %d", j.userID)

	// Phase 1: Account sync — acts as provider key validation gate
	accountResult, err := func() (*openfinance.SyncResult, error) {
		ctx, s := jobTracer.Start(ctx, "sync.user.accounts")
		defer s.End()

		result, err := j.accountSyncService.SyncUserAccounts(ctx, j.userID)
		if err != nil {
			s.RecordError(err)
			s.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		s.SetAttributes(
			attribute.Int("sync.created", result.Created),
			attribute.Int("sync.updated", result.Updated),
			attribute.Int("sync.errors", len(result.Errors)),
		)
		return result, nil
	}()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "account sync failed")
		if errors.Is(err, openfinance.ErrProviderUnauthorized) {
			log.Printf("User %d: Provider key invalid (401) — full sync aborted, key cleared", j.userID)
			return fmt.Errorf("sync aborted: %w", err)
		}
		log.Printf("Account sync failed for user %d: %v", j.userID, err)
		return fmt.Errorf("account sync failed, skipping transaction sync: %w", err)
	}

	log.Printf("Account sync for user %d: Created=%d, Updated=%d, Errors=%d",
		j.userID, accountResult.Created, accountResult.Updated, len(accountResult.Errors))

	// Phase 2: Transaction sync
	hasNewAccounts := accountResult.Created > 0
	txResult, err := func() (*openfinance.TransactionSyncResult, error) {
		ctx, s := jobTracer.Start(ctx, "sync.user.transactions")
		defer s.End()

		result, err := j.txSyncService.SyncUserTransactions(ctx, j.userID, hasNewAccounts)
		if err != nil {
			s.RecordError(err)
			s.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		s.SetAttributes(
			attribute.Int("sync.created", result.Created),
			attribute.Int("sync.updated", result.Updated),
			attribute.Int("sync.skipped", result.Skipped),
			attribute.Int("sync.errors", len(result.Errors)),
		)
		return result, nil
	}()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "transaction sync failed")
		log.Printf("Transaction sync failed for user %d: %v", j.userID, err)
		return fmt.Errorf("transaction sync failed: %w", err)
	}

	log.Printf("Transaction sync for user %d: Created=%d, Updated=%d, Skipped=%d, Errors=%d",
		j.userID, txResult.Created, txResult.Updated, txResult.Skipped, len(txResult.Errors))

	// Phase 3: Bill sync
	billJob := NewBillSyncJob(j.userID, j.billSyncService)
	if err := billJob.Execute(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "bill sync failed")
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

// Execute runs the transaction sync job.
// When run standalone, uses incremental sync (last 7 days).
func (j *TransactionSyncJob) Execute(ctx context.Context) error {
	ctx, span := jobTracer.Start(ctx, "sync.transactions",
		trace.WithAttributes(attribute.Int64("user.id", j.userID)),
	)
	defer span.End()

	log.Printf("Starting transaction sync for user %d", j.userID)

	// Standalone execution uses incremental sync
	result, err := j.syncService.SyncUserTransactions(ctx, j.userID, false)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.Printf("Transaction sync failed for user %d: %v", j.userID, err)
		return fmt.Errorf("sync failed: %w", err)
	}

	span.SetAttributes(
		attribute.Int("sync.created", result.Created),
		attribute.Int("sync.updated", result.Updated),
		attribute.Int("sync.skipped", result.Skipped),
		attribute.Int("sync.errors", len(result.Errors)),
	)

	// Log results
	if len(result.Errors) > 0 {
		span.SetStatus(codes.Error, fmt.Sprintf("sync completed with %d errors", len(result.Errors)))
		log.Printf("Transaction sync for user %d completed with errors: Created=%d, Updated=%d, Skipped=%d, Errors=%d",
			j.userID, result.Created, result.Updated, result.Skipped, len(result.Errors))
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
	ctx, span := jobTracer.Start(ctx, "sync.bills",
		trace.WithAttributes(attribute.Int64("user.id", j.userID)),
	)
	defer span.End()

	log.Printf("Starting bill sync for user %d", j.userID)

	result, err := j.syncService.SyncUserBills(ctx, j.userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.Printf("Bill sync failed for user %d: %v", j.userID, err)
		return fmt.Errorf("sync failed: %w", err)
	}

	span.SetAttributes(
		attribute.Int("sync.created", result.Created),
		attribute.Int("sync.updated", result.Updated),
		attribute.Int("sync.skipped", result.Skipped),
		attribute.Int("sync.errors", len(result.Errors)),
	)

	// Log results
	if len(result.Errors) > 0 {
		span.SetStatus(codes.Error, fmt.Sprintf("sync completed with %d errors", len(result.Errors)))
		log.Printf("Bill sync for user %d completed with errors: Created=%d, Updated=%d, Skipped=%d, Errors=%d",
			j.userID, result.Created, result.Updated, result.Skipped, len(result.Errors))
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
