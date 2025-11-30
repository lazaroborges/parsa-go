package user

import "context"

// Repository defines the interface for user data access
type Repository interface {
	Create(ctx context.Context, params CreateUserParams) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByOAuth(ctx context.Context, provider, oauthID string) (*User, error)
	List(ctx context.Context) ([]*User, error)
	Update(ctx context.Context, userID int64, params UpdateUserParams) (*User, error)
	ListUsersWithProviderKey(ctx context.Context) ([]*User, error)
}
