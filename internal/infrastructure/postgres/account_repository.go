package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"parsa/internal/domain/account"
)

// AccountRepository implements the account.Repository interface for PostgreSQL
type AccountRepository struct {
	db *sql.DB
}

// NewAccountRepository creates a new PostgreSQL account repository
func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

// Create creates a new account
func (r *AccountRepository) Create(ctx context.Context, params account.CreateParams) (*account.Account, error) {
	query := `
		INSERT INTO accounts (id, user_id, item_id, name, account_type, currency, balance, bank_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, user_id, item_id, name, account_type, subtype, currency, balance, bank_id, created_at, updated_at
	`

	var acc account.Account
	var itemID, subtype sql.NullString
	var bankID sql.NullInt64

	err := r.db.QueryRowContext(
		ctx, query,
		params.ID, params.UserID, nullString(params.ItemID), params.Name, params.AccountType, params.Currency, params.Balance, nullInt64(params.BankID),
	).Scan(
		&acc.ID, &acc.UserID, &itemID, &acc.Name,
		&acc.AccountType, &subtype, &acc.Currency, &acc.Balance, &bankID,
		&acc.CreatedAt, &acc.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	if itemID.Valid {
		acc.ItemID = itemID.String
	}
	if subtype.Valid {
		acc.Subtype = subtype.String
	}
	if bankID.Valid {
		acc.BankID = bankID.Int64
	}

	return &acc, nil
}

// GetByID retrieves an account by its ID
func (r *AccountRepository) GetByID(ctx context.Context, id string) (*account.Account, error) {
	query := `
		SELECT id, user_id, item_id, name, account_type, subtype, currency, balance, bank_id,
		       provider_updated_at, provider_created_at, created_at, updated_at
		FROM accounts
		WHERE id = $1
	`

	var acc account.Account
	var itemID, subtype sql.NullString
	var bankID sql.NullInt64
	var providerUpdatedAt, providerCreatedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&acc.ID, &acc.UserID, &itemID, &acc.Name,
		&acc.AccountType, &subtype, &acc.Currency, &acc.Balance,
		&bankID, &providerUpdatedAt, &providerCreatedAt,
		&acc.CreatedAt, &acc.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, account.ErrAccountNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if itemID.Valid {
		acc.ItemID = itemID.String
	}
	if subtype.Valid {
		acc.Subtype = subtype.String
	}
	if bankID.Valid {
		acc.BankID = bankID.Int64
	}
	if providerUpdatedAt.Valid {
		acc.ProviderUpdatedAt = providerUpdatedAt.Time
	}
	if providerCreatedAt.Valid {
		acc.ProviderCreatedAt = providerCreatedAt.Time
	}

	return &acc, nil
}

// ListByUserID retrieves all accounts for a specific user
func (r *AccountRepository) ListByUserID(ctx context.Context, userID int64) ([]*account.Account, error) {
	query := `
		SELECT id, user_id, item_id, name, account_type, subtype, currency, balance, bank_id,
		       provider_updated_at, provider_created_at, created_at, updated_at
		FROM accounts
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []*account.Account
	for rows.Next() {
		var acc account.Account
		var itemID, subtype sql.NullString
		var bankID sql.NullInt64
		var providerUpdatedAt, providerCreatedAt sql.NullTime

		err := rows.Scan(
			&acc.ID, &acc.UserID, &itemID, &acc.Name,
			&acc.AccountType, &subtype, &acc.Currency, &acc.Balance,
			&bankID, &providerUpdatedAt, &providerCreatedAt,
			&acc.CreatedAt, &acc.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}

		if itemID.Valid {
			acc.ItemID = itemID.String
		}
		if subtype.Valid {
			acc.Subtype = subtype.String
		}
		if bankID.Valid {
			acc.BankID = bankID.Int64
		}
		if providerUpdatedAt.Valid {
			acc.ProviderUpdatedAt = providerUpdatedAt.Time
		}
		if providerCreatedAt.Valid {
			acc.ProviderCreatedAt = providerCreatedAt.Time
		}

		accounts = append(accounts, &acc)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating accounts: %w", err)
	}

	return accounts, nil
}

// Delete removes an account
func (r *AccountRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM accounts WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return account.ErrAccountNotFound
	}

	return nil
}

// Upsert creates or updates an account based on its ID
func (r *AccountRepository) Upsert(ctx context.Context, params account.UpsertParams) (*account.Account, error) {
	query := `
		INSERT INTO accounts (
			id, user_id, item_id, name, account_type, subtype, currency, balance, bank_id,
			provider_updated_at, provider_created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id)
		DO UPDATE SET
			name = EXCLUDED.name,
			account_type = EXCLUDED.account_type,
			subtype = EXCLUDED.subtype,
			currency = EXCLUDED.currency,
			balance = EXCLUDED.balance,
			item_id = EXCLUDED.item_id,
			bank_id = EXCLUDED.bank_id,
			provider_updated_at = EXCLUDED.provider_updated_at,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, user_id, item_id, name, account_type, subtype, currency, balance, bank_id,
		          provider_updated_at, provider_created_at, created_at, updated_at
	`

	var acc account.Account
	var itemIDOut, subtypeOut sql.NullString
	var bankIDOut sql.NullInt64
	var providerUpdatedAtOut, providerCreatedAtOut sql.NullTime

	// Convert nullable fields for insert
	var bankIDIn sql.NullInt64
	if params.BankID != nil {
		bankIDIn.Int64 = *params.BankID
		bankIDIn.Valid = true
	}
	var subtypeIn sql.NullString
	if params.Subtype != nil {
		subtypeIn.String = *params.Subtype
		subtypeIn.Valid = true
	}
	var providerUpdatedAtIn, providerCreatedAtIn sql.NullTime
	if params.ProviderUpdatedAt != nil {
		providerUpdatedAtIn.Time = *params.ProviderUpdatedAt
		providerUpdatedAtIn.Valid = true
	}
	if params.ProviderCreatedAt != nil {
		providerCreatedAtIn.Time = *params.ProviderCreatedAt
		providerCreatedAtIn.Valid = true
	}

	err := r.db.QueryRowContext(
		ctx, query,
		params.ID, params.UserID, nullString(params.ItemID), params.Name, params.AccountType,
		subtypeIn, params.Currency, params.Balance, bankIDIn,
		providerUpdatedAtIn, providerCreatedAtIn,
	).Scan(
		&acc.ID, &acc.UserID, &itemIDOut, &acc.Name,
		&acc.AccountType, &subtypeOut, &acc.Currency, &acc.Balance,
		&bankIDOut, &providerUpdatedAtOut, &providerCreatedAtOut,
		&acc.CreatedAt, &acc.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert account: %w", err)
	}

	if itemIDOut.Valid {
		acc.ItemID = itemIDOut.String
	}
	if subtypeOut.Valid {
		acc.Subtype = subtypeOut.String
	}
	if bankIDOut.Valid {
		acc.BankID = bankIDOut.Int64
	}
	if providerUpdatedAtOut.Valid {
		acc.ProviderUpdatedAt = providerUpdatedAtOut.Time
	}
	if providerCreatedAtOut.Valid {
		acc.ProviderCreatedAt = providerCreatedAtOut.Time
	}

	return &acc, nil
}

// Exists checks if an account with the given ID exists
func (r *AccountRepository) Exists(ctx context.Context, id string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM accounts WHERE id = $1)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check account existence: %w", err)
	}

	return exists, nil
}

// FindByMatch finds an account by matching name, account_type, and subtype for a specific user
func (r *AccountRepository) FindByMatch(ctx context.Context, userID int64, name, accountType, subtype string) (*account.Account, error) {
	query := `
		SELECT id, user_id, item_id, name, account_type, subtype, currency, balance, bank_id,
		       provider_updated_at, provider_created_at, created_at, updated_at
		FROM accounts
		WHERE user_id = $1 AND name = $2 AND account_type = $3 AND subtype = $4
		LIMIT 1
	`

	var acc account.Account
	var itemID, subtypeOut sql.NullString
	var bankID sql.NullInt64
	var providerUpdatedAt, providerCreatedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, userID, name, accountType, subtype).Scan(
		&acc.ID, &acc.UserID, &itemID, &acc.Name,
		&acc.AccountType, &subtypeOut, &acc.Currency, &acc.Balance,
		&bankID, &providerUpdatedAt, &providerCreatedAt,
		&acc.CreatedAt, &acc.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Intentionally returns (nil, nil) instead of an error
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find account by match: %w", err)
	}

	if itemID.Valid {
		acc.ItemID = itemID.String
	}
	if subtypeOut.Valid {
		acc.Subtype = subtypeOut.String
	}
	if bankID.Valid {
		acc.BankID = bankID.Int64
	}
	if providerUpdatedAt.Valid {
		acc.ProviderUpdatedAt = providerUpdatedAt.Time
	}
	if providerCreatedAt.Valid {
		acc.ProviderCreatedAt = providerCreatedAt.Time
	}

	return &acc, nil
}

// UpdateBankID updates the bank_id for an account
func (r *AccountRepository) UpdateBankID(ctx context.Context, accountID string, bankID int64) error {
	query := `UPDATE accounts SET bank_id = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`

	result, err := r.db.ExecContext(ctx, query, bankID, accountID)
	if err != nil {
		return fmt.Errorf("failed to update account bank_id: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return account.ErrAccountNotFound
	}

	return nil
}

// Helper functions

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullInt64(i int64) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: i, Valid: true}
}
