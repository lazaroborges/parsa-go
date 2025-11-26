package database

import (
	"context"
	"database/sql"
	"fmt"

	"parsa/internal/models"
)

type TransactionRepository struct {
	db *DB
}

func NewTransactionRepository(db *DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(ctx context.Context, params models.CreateTransactionParams) (*models.Transaction, error) {
	query := `
		INSERT INTO transactions (account_id, amount, description, category, transaction_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, account_id, amount, description, category, transaction_date, created_at, updated_at
	`

	var transaction models.Transaction
	err := r.db.QueryRowContext(
		ctx, query,
		params.AccountID, params.Amount, params.Description, params.Category, params.TransactionDate,
	).Scan(
		&transaction.ID, &transaction.AccountID, &transaction.Amount,
		&transaction.Description, &transaction.Category, &transaction.TransactionDate,
		&transaction.CreatedAt, &transaction.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return &transaction, nil
}

func (r *TransactionRepository) GetByID(ctx context.Context, id int64) (*models.Transaction, error) {
	query := `
		SELECT id, account_id, amount, description, category, transaction_date, created_at, updated_at
		FROM transactions
		WHERE id = $1
	`

	var transaction models.Transaction
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&transaction.ID, &transaction.AccountID, &transaction.Amount,
		&transaction.Description, &transaction.Category, &transaction.TransactionDate,
		&transaction.CreatedAt, &transaction.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("transaction not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &transaction, nil
}

func (r *TransactionRepository) ListByAccountID(ctx context.Context, accountID string, limit, offset int) ([]*models.Transaction, error) {
	query := `
		SELECT id, account_id, amount, description, category, transaction_date, created_at, updated_at
		FROM transactions
		WHERE account_id = $1
		ORDER BY transaction_date DESC, created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, accountID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*models.Transaction
	for rows.Next() {
		var transaction models.Transaction
		err := rows.Scan(
			&transaction.ID, &transaction.AccountID, &transaction.Amount,
			&transaction.Description, &transaction.Category, &transaction.TransactionDate,
			&transaction.CreatedAt, &transaction.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, &transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
}

func (r *TransactionRepository) Update(ctx context.Context, id int64, params models.UpdateTransactionParams) (*models.Transaction, error) {
	query := `
		UPDATE transactions
		SET amount = COALESCE($1, amount),
		    description = COALESCE($2, description),
		    category = COALESCE($3, category),
		    transaction_date = COALESCE($4, transaction_date),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
		RETURNING id, account_id, amount, description, category, transaction_date, created_at, updated_at
	`

	var transaction models.Transaction
	err := r.db.QueryRowContext(
		ctx, query,
		params.Amount, params.Description, params.Category, params.TransactionDate, id,
	).Scan(
		&transaction.ID, &transaction.AccountID, &transaction.Amount,
		&transaction.Description, &transaction.Category, &transaction.TransactionDate,
		&transaction.CreatedAt, &transaction.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("transaction not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update transaction: %w", err)
	}

	return &transaction, nil
}

func (r *TransactionRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM transactions WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("transaction not found")
	}

	return nil
}
