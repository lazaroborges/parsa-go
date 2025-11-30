package account

import "context"

// Repository defines the interface for account data access
// This interface is defined in the domain layer, but implemented in the infrastructure layer
type Repository interface {
	// Create creates a new account
	Create(ctx context.Context, params CreateParams) (*Account, error)

	// GetByID retrieves an account by its ID
	GetByID(ctx context.Context, id string) (*Account, error)

	// ListByUserID retrieves all accounts for a specific user
	ListByUserID(ctx context.Context, userID int64) ([]*Account, error)

	// Delete removes an account
	Delete(ctx context.Context, id string) error

	// Upsert creates or updates an account based on its ID
	Upsert(ctx context.Context, params UpsertParams) (*Account, error)

	// Exists checks if an account with the given ID exists
	Exists(ctx context.Context, id string) (bool, error)

	// FindByMatch finds an account by matching name, account_type, and subtype for a specific user
	FindByMatch(ctx context.Context, userID int64, name, accountType, subtype string) (*Account, error)

	// UpdateBankID updates the bank_id for an account
	UpdateBankID(ctx context.Context, accountID string, bankID int64) error
}
