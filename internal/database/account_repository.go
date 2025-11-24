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
		INSERT INTO accounts (user_id, name, account_type, currency, balance)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, name, account_type, currency, balance, created_at, updated_at
	`

	var account models.Account
	err := r.db.QueryRowContext(
		ctx, query,
		params.UserID, params.Name, params.AccountType, params.Currency, params.Balance,
	).Scan(
		&account.ID, &account.UserID, &account.Name,
		&account.AccountType, &account.Currency, &account.Balance,
		&account.CreatedAt, &account.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return &account, nil
}

func (r *AccountRepository) GetByID(ctx context.Context, id int64) (*models.Account, error) {
	query := `
		SELECT id, user_id, name, account_type, currency, balance, created_at, updated_at
		FROM accounts
		WHERE id = $1
	`

	var account models.Account
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&account.ID, &account.UserID, &account.Name,
		&account.AccountType, &account.Currency, &account.Balance,
		&account.CreatedAt, &account.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("account not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return &account, nil
}

func (r *AccountRepository) ListByUserID(ctx context.Context, userID int64) ([]*models.Account, error) {
	query := `
		SELECT id, user_id, name, account_type, currency, balance, created_at, updated_at
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
		err := rows.Scan(
			&account.ID, &account.UserID, &account.Name,
			&account.AccountType, &account.Currency, &account.Balance,
			&account.CreatedAt, &account.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		accounts = append(accounts, &account)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating accounts: %w", err)
	}

	return accounts, nil
}

func (r *AccountRepository) UpdateBalance(ctx context.Context, id int64, balance float64) error {
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

func (r *AccountRepository) Delete(ctx context.Context, id int64) error {
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

// GetByProviderID retrieves an account by its provider ID and user ID
func (r *AccountRepository) GetByProviderID(ctx context.Context, userID int64, providerID string) (*models.Account, error) {
	query := `
		SELECT id, user_id, name, account_type, subtype, currency, balance, bank_id,
		       provider_id, provider_updated_at, provider_created_at, created_at, updated_at
		FROM accounts
		WHERE user_id = $1 AND provider_id = $2
	`

	var account models.Account
	var bankID sql.NullInt64
	var subtype, providerIDVal sql.NullString
	var providerUpdatedAt, providerCreatedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, userID, providerID).Scan(
		&account.ID, &account.UserID, &account.Name,
		&account.AccountType, &subtype, &account.Currency, &account.Balance,
		&bankID, &providerIDVal, &providerUpdatedAt, &providerCreatedAt,
		&account.CreatedAt, &account.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("account not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if bankID.Valid {
		account.BankID = bankID.Int64
	}
	if subtype.Valid {
		account.Subtype = subtype.String
	}
	if providerIDVal.Valid {
		account.ProviderID = providerIDVal.String
	}
	if providerUpdatedAt.Valid {
		account.ProviderUpdatedAt = providerUpdatedAt.Time
	}
	if providerCreatedAt.Valid {
		account.ProviderCreatedAt = providerCreatedAt.Time
	}

	return &account, nil
}

// UpsertByProviderID creates or updates an account based on provider_id
func (r *AccountRepository) UpsertByProviderID(ctx context.Context, params models.UpsertAccountParams) (*models.Account, error) {
	// Validate that provider_id is not empty (required for upsert)
	if params.ProviderID == "" {
		return nil, fmt.Errorf("provider_id is required for upsert")
	}

	query := `
		INSERT INTO accounts (
			user_id, name, account_type, subtype, currency, balance, bank_id,
			provider_id, provider_updated_at, provider_created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (user_id, provider_id)
		DO UPDATE SET
			name = EXCLUDED.name,
			account_type = EXCLUDED.account_type,
			subtype = EXCLUDED.subtype,
			currency = EXCLUDED.currency,
			balance = EXCLUDED.balance,
			bank_id = EXCLUDED.bank_id,
			provider_updated_at = EXCLUDED.provider_updated_at,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, user_id, name, account_type, subtype, currency, balance, bank_id,
		          provider_id, provider_updated_at, provider_created_at, created_at, updated_at
	`

	var account models.Account
	var bankIDOut sql.NullInt64
	var subtypeOut, providerIDOut sql.NullString
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
	var providerIDIn sql.NullString
	if params.ProviderID != "" {
		providerIDIn.String = params.ProviderID
		providerIDIn.Valid = true
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
		params.UserID, params.Name, params.AccountType, subtypeIn, params.Currency,
		params.Balance, bankIDIn, providerIDIn, providerUpdatedAtIn, providerCreatedAtIn,
	).Scan(
		&account.ID, &account.UserID, &account.Name,
		&account.AccountType, &subtypeOut, &account.Currency, &account.Balance,
		&bankIDOut, &providerIDOut, &providerUpdatedAtOut, &providerCreatedAtOut,
		&account.CreatedAt, &account.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert account: %w", err)
	}

	if bankIDOut.Valid {
		account.BankID = bankIDOut.Int64
	}
	if subtypeOut.Valid {
		account.Subtype = subtypeOut.String
	}
	if providerIDOut.Valid {
		account.ProviderID = providerIDOut.String
	}
	if providerUpdatedAtOut.Valid {
		account.ProviderUpdatedAt = providerUpdatedAtOut.Time
	}
	if providerCreatedAtOut.Valid {
		account.ProviderCreatedAt = providerCreatedAtOut.Time
	}

	return &account, nil
}
