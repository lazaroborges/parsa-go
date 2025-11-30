package account

import (
	"context"
	"errors"
)

// Service contains the business logic for account operations
type Service struct {
	repo Repository
}

// NewService creates a new account service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateAccount creates a new account with business validation
func (s *Service) CreateAccount(ctx context.Context, params CreateParams) (*Account, error) {
	// Apply default currency if not provided
	if params.Currency == "" {
		params.Currency = "BRL"
	}

	// Validate parameters
	if err := params.Validate(); err != nil {
		return nil, err
	}

	// Create account
	return s.repo.Create(ctx, params)
}

// GetAccount retrieves an account by ID and verifies user ownership
func (s *Service) GetAccount(ctx context.Context, accountID string, userID int64) (*Account, error) {
	account, err := s.repo.GetByID(ctx, accountID)
	if err != nil {
		// Wrap repository errors
		if errors.Is(err, ErrAccountNotFound) || err.Error() == "account not found" {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}

	// Business rule: verify ownership
	if account.UserID != userID {
		return nil, ErrForbidden
	}

	return account, nil
}

// ListAccountsByUserID retrieves all accounts for a specific user
func (s *Service) ListAccountsByUserID(ctx context.Context, userID int64) ([]*Account, error) {
	if userID <= 0 {
		return nil, errors.New("valid user ID is required")
	}

	return s.repo.ListByUserID(ctx, userID)
}

// DeleteAccount deletes an account after verifying ownership
func (s *Service) DeleteAccount(ctx context.Context, accountID string, userID int64) error {
	// Verify ownership before deletion
	account, err := s.GetAccount(ctx, accountID, userID)
	if err != nil {
		return err
	}

	if account.UserID != userID {
		return ErrForbidden
	}

	return s.repo.Delete(ctx, accountID)
}

// UpsertAccount creates or updates an account with validation
func (s *Service) UpsertAccount(ctx context.Context, params UpsertParams) (*Account, error) {
	// Apply default currency if not provided
	if params.Currency == "" {
		params.Currency = "BRL"
	}

	// Validate parameters
	if err := params.Validate(); err != nil {
		return nil, err
	}

	return s.repo.Upsert(ctx, params)
}

// AccountExists checks if an account exists
func (s *Service) AccountExists(ctx context.Context, accountID string) (bool, error) {
	return s.repo.Exists(ctx, accountID)
}

// FindAccountByMatch finds an account by matching criteria
func (s *Service) FindAccountByMatch(ctx context.Context, userID int64, name, accountType, subtype string) (*Account, error) {
	// Validate account type
	if !IsValidAccountType(accountType) {
		return nil, ErrInvalidAccountType
	}

	// Validate account subtype if provided
	if subtype != "" && !IsValidAccountSubtype(subtype) {
		return nil, ErrInvalidAccountSubtype
	}

	return s.repo.FindByMatch(ctx, userID, name, accountType, subtype)
}

// UpdateAccountBankID updates the bank ID for an account
func (s *Service) UpdateAccountBankID(ctx context.Context, accountID string, bankID int64, userID int64) error {
	// Verify ownership
	account, err := s.GetAccount(ctx, accountID, userID)
	if err != nil {
		return err
	}

	if account.UserID != userID {
		return ErrForbidden
	}

	return s.repo.UpdateBankID(ctx, accountID, bankID)
}
