package transaction

import "context"

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
}
