package openfinance

import (
	"context"
	"fmt"
	"log"
	"strconv"
)

// AccountSyncJob implements the scheduler.Job interface for syncing user accounts
type AccountSyncJob struct {
	userID      int64
	syncService *AccountSyncService
}

// NewAccountSyncJob creates a new account sync job for a user
func NewAccountSyncJob(userID int64, syncService *AccountSyncService) *AccountSyncJob {
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
