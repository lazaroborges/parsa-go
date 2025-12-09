package transaction

import (
	"context"
	"time"
)

// DuplicateCriteria defines the search criteria for finding potential duplicates
type DuplicateCriteria struct {
	ExcludeID      string    // The transaction ID to exclude from results
	OppositeType   string    // The opposite type to search for (DEBIT -> CREDIT, CREDIT -> DEBIT). Empty string means any type.
	AbsoluteAmount float64   // The absolute amount to match
	DateLowerBound time.Time // Lower bound of transaction date range
	DateUpperBound time.Time // Upper bound of transaction date range
	UserID         int64     // User ID to scope the search
}

// Repository defines the interface for transaction data access
type Repository interface {
	Create(ctx context.Context, params CreateTransactionParams) (*Transaction, error)
	GetByID(ctx context.Context, id string) (*Transaction, error)
	ListByAccountID(ctx context.Context, accountID string, limit, offset int) ([]*Transaction, error)
	ListByUserID(ctx context.Context, userID int64, limit, offset int) ([]*Transaction, error)
	CountByUserID(ctx context.Context, userID int64) (int64, error)
	Update(ctx context.Context, id string, params UpdateTransactionParams) (*Transaction, error)
	Delete(ctx context.Context, id string) error
	Upsert(ctx context.Context, params UpsertTransactionParams) (*Transaction, error)
	// FindPotentialDuplicates finds transactions that match the duplicate criteria
	// Returns transactions with different ID, opposite type, same absolute amount,
	// and transaction date within the specified range for the same user
	FindPotentialDuplicates(ctx context.Context, criteria DuplicateCriteria) ([]*Transaction, error)
	// FindPotentialDuplicatesForBill finds transactions that could be duplicates related to bills
	// Returns transactions with different ID, same absolute amount (any type),
	// and transaction date within the specified range for the same user
	FindPotentialDuplicatesForBill(ctx context.Context, criteria DuplicateCriteria) ([]*Transaction, error)
	// SetTransactionTags replaces all tags for a transaction
	SetTransactionTags(ctx context.Context, transactionID string, tagIDs []string) error
	// GetTransactionTags returns all tag IDs for a transaction
	GetTransactionTags(ctx context.Context, transactionID string) ([]string, error)
}
