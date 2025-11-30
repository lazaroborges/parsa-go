package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"parsa/internal/domain"
)

type TransactionRepository struct {
	db *DB
}

func NewTransactionRepository(db *DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(ctx context.Context, params models.CreateTransactionParams) (*models.Transaction, error) {
	query := `
		INSERT INTO transactions (id, account_id, amount, description, category, transaction_date, type, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, account_id, amount, description, category, transaction_date, type, status,
		          provider_created_at, provider_updated_at, created_at, updated_at
	`

	var transaction models.Transaction
	var providerCreatedAt, providerUpdatedAt sql.NullTime

	err := r.db.QueryRowContext(
		ctx, query,
		params.ID, params.AccountID, params.Amount, params.Description, params.Category,
		params.TransactionDate, params.Type, params.Status,
	).Scan(
		&transaction.ID, &transaction.AccountID, &transaction.Amount,
		&transaction.Description, &transaction.Category, &transaction.TransactionDate,
		&transaction.Type, &transaction.Status,
		&providerCreatedAt, &providerUpdatedAt,
		&transaction.CreatedAt, &transaction.UpdatedAt,
	)

	if providerCreatedAt.Valid {
		transaction.ProviderCreatedAt = providerCreatedAt.Time
	}
	if providerUpdatedAt.Valid {
		transaction.ProviderUpdatedAt = providerUpdatedAt.Time
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return &transaction, nil
}

func (r *TransactionRepository) GetByID(ctx context.Context, id string) (*models.Transaction, error) {
	query := `
		SELECT id, account_id, amount, description, category, transaction_date, type, status,
		       provider_created_at, provider_updated_at, created_at, updated_at
		FROM transactions
		WHERE id = $1
	`

	var transaction models.Transaction
	var providerCreatedAt, providerUpdatedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&transaction.ID, &transaction.AccountID, &transaction.Amount,
		&transaction.Description, &transaction.Category, &transaction.TransactionDate,
		&transaction.Type, &transaction.Status,
		&providerCreatedAt, &providerUpdatedAt,
		&transaction.CreatedAt, &transaction.UpdatedAt,
	)

	if providerCreatedAt.Valid {
		transaction.ProviderCreatedAt = providerCreatedAt.Time
	}
	if providerUpdatedAt.Valid {
		transaction.ProviderUpdatedAt = providerUpdatedAt.Time
	}

	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &transaction, nil
}

func (r *TransactionRepository) ListByAccountID(ctx context.Context, accountID string, limit, offset int) ([]*models.Transaction, error) {
	query := `
		SELECT id, account_id, amount, description, category, transaction_date, type, status,
		       provider_created_at, provider_updated_at, created_at, updated_at
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
		var providerCreatedAt, providerUpdatedAt sql.NullTime

		err := rows.Scan(
			&transaction.ID, &transaction.AccountID, &transaction.Amount,
			&transaction.Description, &transaction.Category, &transaction.TransactionDate,
			&transaction.Type, &transaction.Status,
			&providerCreatedAt, &providerUpdatedAt,
			&transaction.CreatedAt, &transaction.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		if providerCreatedAt.Valid {
			transaction.ProviderCreatedAt = providerCreatedAt.Time
		}
		if providerUpdatedAt.Valid {
			transaction.ProviderUpdatedAt = providerUpdatedAt.Time
		}

		transactions = append(transactions, &transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
}

func (r *TransactionRepository) Update(ctx context.Context, id string, params models.UpdateTransactionParams) (*models.Transaction, error) {
	query := `
		UPDATE transactions
		SET amount = COALESCE($1, amount),
		    description = COALESCE($2, description),
		    category = COALESCE($3, category),
		    transaction_date = COALESCE($4, transaction_date),
		    type = COALESCE($5, type),
		    status = COALESCE($6, status),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $7
		RETURNING id, account_id, amount, description, category, transaction_date, type, status,
		          provider_created_at, provider_updated_at, created_at, updated_at
	`

	var transaction models.Transaction
	var providerCreatedAt, providerUpdatedAt sql.NullTime

	err := r.db.QueryRowContext(
		ctx, query,
		params.Amount, params.Description, params.Category, params.TransactionDate,
		params.Type, params.Status, id,
	).Scan(
		&transaction.ID, &transaction.AccountID, &transaction.Amount,
		&transaction.Description, &transaction.Category, &transaction.TransactionDate,
		&transaction.Type, &transaction.Status,
		&providerCreatedAt, &providerUpdatedAt,
		&transaction.CreatedAt, &transaction.UpdatedAt,
	)

	if providerCreatedAt.Valid {
		transaction.ProviderCreatedAt = providerCreatedAt.Time
	}
	if providerUpdatedAt.Valid {
		transaction.ProviderUpdatedAt = providerUpdatedAt.Time
	}

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("transaction not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update transaction: %w", err)
	}

	return &transaction, nil
}

func (r *TransactionRepository) Delete(ctx context.Context, id string) error {
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

// Upsert inserts or updates a transaction (used for syncing from provider)
func (r *TransactionRepository) Upsert(ctx context.Context, params models.UpsertTransactionParams) (*models.Transaction, error) {
	query := `
		INSERT INTO transactions (id, account_id, amount, description, category, transaction_date, 
		                          type, status, provider_created_at, provider_updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
		    amount = EXCLUDED.amount,
		    description = EXCLUDED.description,
		    category = EXCLUDED.category,
		    transaction_date = EXCLUDED.transaction_date,
		    type = EXCLUDED.type,
		    status = EXCLUDED.status,
		    provider_created_at = EXCLUDED.provider_created_at,
		    provider_updated_at = EXCLUDED.provider_updated_at,
		    updated_at = CURRENT_TIMESTAMP
		RETURNING id, account_id, amount, description, category, transaction_date, type, status,
		          provider_created_at, provider_updated_at, created_at, updated_at
	`

	var transaction models.Transaction
	var providerCreatedAt, providerUpdatedAt sql.NullTime

	err := r.db.QueryRowContext(
		ctx, query,
		params.ID, params.AccountID, params.Amount, params.Description, params.Category,
		params.TransactionDate, params.Type, params.Status,
		params.ProviderCreatedAt, params.ProviderUpdatedAt,
	).Scan(
		&transaction.ID, &transaction.AccountID, &transaction.Amount,
		&transaction.Description, &transaction.Category, &transaction.TransactionDate,
		&transaction.Type, &transaction.Status,
		&providerCreatedAt, &providerUpdatedAt,
		&transaction.CreatedAt, &transaction.UpdatedAt,
	)

	if providerCreatedAt.Valid {
		transaction.ProviderCreatedAt = providerCreatedAt.Time
	}
	if providerUpdatedAt.Valid {
		transaction.ProviderUpdatedAt = providerUpdatedAt.Time
	}

	if err != nil {
		return nil, fmt.Errorf("failed to upsert transaction: %w", err)
	}

	return &transaction, nil
}
