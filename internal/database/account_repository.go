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

func (r *AccountRepository) GetByID(ctx context.Context, id string) (*models.Account, error) {
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

func (r *AccountRepository) ListByUserID(ctx context.Context, userID string) ([]*models.Account, error) {
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
