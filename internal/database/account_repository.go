package database

import (
	"context"
	"database/sql"
	"fmt"

	"parsa/internal/models"
)

type AccountRepository struct {
	db *DB
}

func NewAccountRepository(db *DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) Create(ctx context.Context, params models.CreateAccountParams) (*models.Account, error) {
	query := `
		INSERT INTO accounts (id, user_id, item_id, name, account_type, currency, balance)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, item_id, name, account_type, currency, balance, created_at, updated_at
	`

	var account models.Account
	var itemID sql.NullString

	err := r.db.QueryRowContext(
		ctx, query,
		params.ID, params.UserID, nullString(params.ItemID), params.Name, params.AccountType, params.Currency, params.Balance,
	).Scan(
		&account.ID, &account.UserID, &itemID, &account.Name,
		&account.AccountType, &account.Currency, &account.Balance,
		&account.CreatedAt, &account.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	if itemID.Valid {
		account.ItemID = itemID.String
	}

	return &account, nil
}

func (r *AccountRepository) GetByID(ctx context.Context, id string) (*models.Account, error) {
	query := `
		SELECT id, user_id, item_id, name, account_type, subtype, currency, balance, bank_id,
		       provider_updated_at, provider_created_at, created_at, updated_at
		FROM accounts
		WHERE id = $1
	`

	var account models.Account
	var itemID, subtype sql.NullString
	var bankID sql.NullInt64
	var providerUpdatedAt, providerCreatedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&account.ID, &account.UserID, &itemID, &account.Name,
		&account.AccountType, &subtype, &account.Currency, &account.Balance,
		&bankID, &providerUpdatedAt, &providerCreatedAt,
		&account.CreatedAt, &account.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("account not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if itemID.Valid {
		account.ItemID = itemID.String
	}
	if subtype.Valid {
		account.Subtype = subtype.String
	}
	if bankID.Valid {
		account.BankID = bankID.Int64
	}
	if providerUpdatedAt.Valid {
		account.ProviderUpdatedAt = providerUpdatedAt.Time
	}
	if providerCreatedAt.Valid {
		account.ProviderCreatedAt = providerCreatedAt.Time
	}

	return &account, nil
}

func (r *AccountRepository) ListByUserID(ctx context.Context, userID int64) ([]*models.Account, error) {
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

	var accounts []*models.Account
	for rows.Next() {
		var account models.Account
		var itemID, subtype sql.NullString
		var bankID sql.NullInt64
		var providerUpdatedAt, providerCreatedAt sql.NullTime

		err := rows.Scan(
			&account.ID, &account.UserID, &itemID, &account.Name,
			&account.AccountType, &subtype, &account.Currency, &account.Balance,
			&bankID, &providerUpdatedAt, &providerCreatedAt,
			&account.CreatedAt, &account.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}

		if itemID.Valid {
			account.ItemID = itemID.String
		}
		if subtype.Valid {
			account.Subtype = subtype.String
		}
		if bankID.Valid {
			account.BankID = bankID.Int64
		}
		if providerUpdatedAt.Valid {
			account.ProviderUpdatedAt = providerUpdatedAt.Time
		}
		if providerCreatedAt.Valid {
			account.ProviderCreatedAt = providerCreatedAt.Time
		}

		accounts = append(accounts, &account)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating accounts: %w", err)
	}

	return accounts, nil
}

func (r *AccountRepository) UpdateBalance(ctx context.Context, id string, balance float64) error {
	query := `
		UPDATE accounts
		SET balance = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, balance, id)
	if err != nil {
		return fmt.Errorf("failed to update account balance: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("account not found")
	}

	return nil
}

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
		return fmt.Errorf("account not found")
	}

	return nil
}

// Upsert creates or updates an account based on its ID (provider's account id)
func (r *AccountRepository) Upsert(ctx context.Context, params models.UpsertAccountParams) (*models.Account, error) {
	if params.ID == "" {
		return nil, fmt.Errorf("account ID is required for upsert")
	}

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
			provider_updated_at = EXCLUDED.provider_updated_at,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, user_id, item_id, name, account_type, subtype, currency, balance, bank_id,
		          provider_updated_at, provider_created_at, created_at, updated_at
	`

	var account models.Account
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
		&account.ID, &account.UserID, &itemIDOut, &account.Name,
		&account.AccountType, &subtypeOut, &account.Currency, &account.Balance,
		&bankIDOut, &providerUpdatedAtOut, &providerCreatedAtOut,
		&account.CreatedAt, &account.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert account: %w", err)
	}

	if itemIDOut.Valid {
		account.ItemID = itemIDOut.String
	}
	if subtypeOut.Valid {
		account.Subtype = subtypeOut.String
	}
	if bankIDOut.Valid {
		account.BankID = bankIDOut.Int64
	}
	if providerUpdatedAtOut.Valid {
		account.ProviderUpdatedAt = providerUpdatedAtOut.Time
	}
	if providerCreatedAtOut.Valid {
		account.ProviderCreatedAt = providerCreatedAtOut.Time
	}

	return &account, nil
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
// This is used to match transactions from the provider API to our accounts
func (r *AccountRepository) FindByMatch(ctx context.Context, userID int64, name, accountType, subtype string) (*models.Account, error) {
	query := `
		SELECT id, user_id, item_id, name, account_type, subtype, currency, balance, bank_id,
		       provider_updated_at, provider_created_at, created_at, updated_at
		FROM accounts
		WHERE user_id = $1 AND name = $2 AND account_type = $3 AND subtype = $4
		LIMIT 1
	`

	var account models.Account
	var itemID, subtypeOut sql.NullString
	var bankID sql.NullInt64
	var providerUpdatedAt, providerCreatedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, userID, name, accountType, subtype).Scan(
		&account.ID, &account.UserID, &itemID, &account.Name,
		&account.AccountType, &subtypeOut, &account.Currency, &account.Balance,
		&bankID, &providerUpdatedAt, &providerCreatedAt,
		&account.CreatedAt, &account.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Intentionally returns (nil, nil) instead of an error.
		// This allows callers to distinguish "no match" from actual errors,
		// e.g., transaction_sync skips transactions when no matching account exists.
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find account by match: %w", err)
	}

	if itemID.Valid {
		account.ItemID = itemID.String
	}
	if subtypeOut.Valid {
		account.Subtype = subtypeOut.String
	}
	if bankID.Valid {
		account.BankID = bankID.Int64
	}
	if providerUpdatedAt.Valid {
		account.ProviderUpdatedAt = providerUpdatedAt.Time
	}
	if providerCreatedAt.Valid {
		account.ProviderCreatedAt = providerCreatedAt.Time
	}

	return &account, nil
}

// UpdateBankID updates the bank_id for an account
func (r *AccountRepository) UpdateBankID(ctx context.Context, accountID string, bankID int64) error {
	query := `UPDATE accounts SET bank_id = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, bankID, accountID)
	if err != nil {
		return fmt.Errorf("failed to update account bank_id: %w", err)
	}
	return nil
}

// nullString converts a string to sql.NullString (empty string = NULL)
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
