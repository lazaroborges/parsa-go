package tag

import (
	"context"
)

type Repository interface {
	Create(ctx context.Context, userID int64, params CreateTagParams) (*Tag, error)
	GetByID(ctx context.Context, id string) (*Tag, error)
	ListByUserID(ctx context.Context, userID int64) ([]*Tag, error)
	Update(ctx context.Context, id string, params UpdateTagParams) (*Tag, error)
	Delete(ctx context.Context, id string) error
}
