package account

import (
	"context"
	"errors"
	"fmt"
	"log"

	"parsa/internal/domain/transaction"
	"parsa/internal/models"
)

// Service contains the business logic for account operations
type Service struct {
	repo            Repository
	itemRepo        models.ItemRepository
	transactionRepo transaction.Repository
}

// NewService creates a new account service
func NewService(repo Repository, itemRepo models.ItemRepository, transactionRepo transaction.Repository) *Service {
	return &Service{
		repo:            repo,
		itemRepo:        itemRepo,
		transactionRepo: transactionRepo,
	}
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

// ListAccountsWithBankByUserID retrieves all accounts for a specific user with bank data
func (s *Service) ListAccountsWithBankByUserID(ctx context.Context, userID int64) ([]*AccountWithBank, error) {
	if userID <= 0 {
		return nil, errors.New("valid user ID is required")
	}

	return s.repo.ListByUserIDWithBank(ctx, userID)
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

// UpdateAccount updates an account after verifying ownership
func (s *Service) UpdateAccount(ctx context.Context, accountID string, params UpdateParams, userID int64) (*Account, error) {
	// Verify ownership before update
	account, err := s.GetAccount(ctx, accountID, userID)
	if err != nil {
		return nil, err
	}

	if account.UserID != userID {
		return nil, ErrForbidden
	}

	return s.repo.Update(ctx, accountID, params)
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

// GetAccountByID retrieves an account by ID without ownership check (for internal/sync use)
func (s *Service) GetAccountByID(ctx context.Context, accountID string) (*Account, error) {
	return s.repo.GetByID(ctx, accountID)
}

// AccountExists checks if an account exists
func (s *Service) AccountExists(ctx context.Context, accountID string) (bool, error) {
	return s.repo.Exists(ctx, accountID)
}

// FindAccountByMatch finds an account by matching criteria
// Note: This method does not validate account type/subtype to allow matching
// against accounts that may have been synced from external APIs with types
// not in our predefined list. Validation only occurs during create/upsert.
func (s *Service) FindAccountByMatch(ctx context.Context, userID int64, name, accountType, subtype string) (*Account, error) {
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

// RemoveAccount soft-removes an account (sets removed_at) after verifying ownership
func (s *Service) RemoveAccount(ctx context.Context, accountID string, userID int64) error {
	acc, err := s.GetAccount(ctx, accountID, userID)
	if err != nil {
		return err
	}

	if acc.RemovedAt != nil {
		return ErrAccountAlreadyRemoved
	}

	return s.repo.SoftRemove(ctx, accountID)
}

// RestoreAccount restores a previously removed account after verifying ownership.
// Nulls removed_at; the repo returns ErrAccountNotRemoved if the account was not removed.
func (s *Service) RestoreAccount(ctx context.Context, accountID string, userID int64) error {
	if _, err := s.GetAccount(ctx, accountID, userID); err != nil {
		return err
	}
	return s.repo.Restore(ctx, accountID)
}

// DeleteBank deletes all accounts and their transactions for the bank connection (item)
// associated with the given account, then soft-deletes the item itself.
func (s *Service) DeleteBank(ctx context.Context, accountID string, userID int64) error {
	acc, err := s.GetAccount(ctx, accountID, userID)
	if err != nil {
		return err
	}

	if acc.ItemID == "" {
		return ErrAccountNoItem
	}

	accounts, err := s.repo.ListByItemID(ctx, acc.ItemID)
	if err != nil {
		return fmt.Errorf("failed to list accounts for item: %w", err)
	}

	for _, a := range accounts {
		if err := s.transactionRepo.DeleteByAccountID(ctx, a.ID); err != nil {
			log.Printf("Error deleting transactions for account %s: %v", a.ID, err)
			return fmt.Errorf("failed to delete transactions for account %s: %w", a.ID, err)
		}
	}

	if err := s.repo.DeleteByItemID(ctx, acc.ItemID); err != nil {
		return fmt.Errorf("failed to delete accounts for item: %w", err)
	}

	if err := s.itemRepo.SoftDelete(ctx, acc.ItemID); err != nil {
		return fmt.Errorf("failed to soft-delete item: %w", err)
	}

	return nil
}
