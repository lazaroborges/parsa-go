package bill

import (
	"context"
	"time"
)

// BillMatchCriteria defines the search criteria for finding a matching bill
type BillMatchCriteria struct {
	AccountID      string    // The account ID to match
	Amount         float64   // The amount to match (exact match)
	DateLowerBound time.Time // Lower bound of due date range
	DateUpperBound time.Time // Upper bound of due date range
}

// Repository defines the interface for bill data access
type Repository interface {
	Create(ctx context.Context, params CreateParams) (*Bill, error)
	GetByID(ctx context.Context, id string) (*Bill, error)
	ListByAccountID(ctx context.Context, accountID string, limit, offset int) ([]*Bill, error)
	ListByUserID(ctx context.Context, userID int64, limit, offset int) ([]*Bill, error)
	CountByUserID(ctx context.Context, userID int64) (int64, error)
	Update(ctx context.Context, id string, params UpdateParams) (*Bill, error)
	Delete(ctx context.Context, id string) error
	Upsert(ctx context.Context, params UpsertParams) (*Bill, error)
	// FindMatchingBill finds a bill that matches the given criteria
	// Used for detecting credit card bill payment transactions
	FindMatchingBill(ctx context.Context, criteria BillMatchCriteria) (*Bill, error)
}
