package bill

import (
	"context"
)

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
}
