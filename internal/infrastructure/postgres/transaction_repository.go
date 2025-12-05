package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"parsa/internal/domain/transaction"
)

type TransactionRepository struct {
	db *DB
}

func NewTransactionRepository(db *DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(ctx context.Context, params transaction.CreateTransactionParams) (*transaction.Transaction, error) {
	query := `
		INSERT INTO transactions (id, account_id, amount, description, category, transaction_date, type, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, account_id, amount, description, category, transaction_date, type, status,
		          provider_created_at, provider_updated_at, created_at, updated_at
	`

	var transaction transaction.Transaction
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

func (r *TransactionRepository) GetByID(ctx context.Context, id string) (*transaction.Transaction, error) {
	query := `
		SELECT id, account_id, amount, description, category, original_description, original_category,
		       transaction_date, type, status,
		       provider_created_at, provider_updated_at, created_at, updated_at,
		       considered, is_open_finance, tags, manipulated, notes
		FROM transactions
		WHERE id = $1
	`

	var txn transaction.Transaction
	var providerCreatedAt, providerUpdatedAt sql.NullTime
	var tags []byte // PostgreSQL array comes as bytes
	var originalDescription, originalCategory sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&txn.ID, &txn.AccountID, &txn.Amount,
		&txn.Description, &txn.Category, &originalDescription, &originalCategory,
		&txn.TransactionDate,
		&txn.Type, &txn.Status,
		&providerCreatedAt, &providerUpdatedAt,
		&txn.CreatedAt, &txn.UpdatedAt,
		&txn.Considered, &txn.IsOpenFinance, &tags, &txn.Manipulated, &txn.Notes,
	)

	if providerCreatedAt.Valid {
		txn.ProviderCreatedAt = providerCreatedAt.Time
	}
	if providerUpdatedAt.Valid {
		txn.ProviderUpdatedAt = providerUpdatedAt.Time
	}
	if originalDescription.Valid {
		txn.OriginalDescription = &originalDescription.String
	}
	if originalCategory.Valid {
		txn.OriginalCategory = &originalCategory.String
	}
	// Tags are null or empty, so just set to empty slice
	txn.Tags = []string{}

	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &txn, nil
}

func (r *TransactionRepository) ListByAccountID(ctx context.Context, accountID string, limit, offset int) ([]*transaction.Transaction, error) {
	query := `
		SELECT id, account_id, amount, description, category, original_description, original_category,
		       transaction_date, type, status,
		       provider_created_at, provider_updated_at, created_at, updated_at,
		       considered, is_open_finance, tags, manipulated, notes
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

	return scanTransactions(rows)
}

// ListByUserID returns all transactions for a user across all accounts
func (r *TransactionRepository) ListByUserID(ctx context.Context, userID int64, limit, offset int) ([]*transaction.Transaction, error) {
	query := `
		SELECT t.id, t.account_id, t.amount, t.description, t.category, t.original_description, t.original_category,
		       t.transaction_date, t.type, t.status,
		       t.provider_created_at, t.provider_updated_at, t.created_at, t.updated_at,
		       t.considered, t.is_open_finance, t.tags, t.manipulated, t.notes
		FROM transactions t
		JOIN accounts a ON t.account_id = a.id
		WHERE a.user_id = $1
		ORDER BY t.transaction_date DESC, t.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions by user: %w", err)
	}
	defer rows.Close()

	return scanTransactions(rows)
}

// CountByUserID returns the total count of transactions for a user
func (r *TransactionRepository) CountByUserID(ctx context.Context, userID int64) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM transactions t
		JOIN accounts a ON t.account_id = a.id
		WHERE a.user_id = $1
	`

	var count int64
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	return count, nil
}

// scanTransactions is a helper to scan transaction rows
func scanTransactions(rows *sql.Rows) ([]*transaction.Transaction, error) {
	var transactions []*transaction.Transaction
	for rows.Next() {
		var txn transaction.Transaction
		var providerCreatedAt, providerUpdatedAt sql.NullTime
		var tags []byte
		var originalDescription, originalCategory sql.NullString

		err := rows.Scan(
			&txn.ID, &txn.AccountID, &txn.Amount,
			&txn.Description, &txn.Category, &originalDescription, &originalCategory,
			&txn.TransactionDate,
			&txn.Type, &txn.Status,
			&providerCreatedAt, &providerUpdatedAt,
			&txn.CreatedAt, &txn.UpdatedAt,
			&txn.Considered, &txn.IsOpenFinance, &tags, &txn.Manipulated, &txn.Notes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		if providerCreatedAt.Valid {
			txn.ProviderCreatedAt = providerCreatedAt.Time
		}
		if providerUpdatedAt.Valid {
			txn.ProviderUpdatedAt = providerUpdatedAt.Time
		}
		if originalDescription.Valid {
			txn.OriginalDescription = &originalDescription.String
		}
		if originalCategory.Valid {
			txn.OriginalCategory = &originalCategory.String
		}
		// Tags are null or empty, so just set to empty slice
		txn.Tags = []string{}

		transactions = append(transactions, &txn)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
}

func (r *TransactionRepository) Update(ctx context.Context, id string, params transaction.UpdateTransactionParams) (*transaction.Transaction, error) {
	query := `
		UPDATE transactions
		SET amount = COALESCE($1, amount),
		    description = COALESCE($2, description),
		    original_description = CASE
		        WHEN $2 IS NOT NULL AND $2 IS DISTINCT FROM description AND original_description IS NULL THEN description
		        ELSE original_description
		    END,
		    category = COALESCE($3, category),
		    original_category = CASE
		        WHEN $3 IS NOT NULL AND $3 IS DISTINCT FROM category AND original_category IS NULL THEN category
		        ELSE original_category
		    END,
		    transaction_date = COALESCE($4, transaction_date),
		    type = COALESCE($5, type),
		    status = COALESCE($6, status),
		    considered = COALESCE($7, considered),
		    notes = COALESCE($8, notes),
		    manipulated = CASE
		        WHEN $2 IS NOT NULL AND $2 IS DISTINCT FROM description THEN true
		        WHEN $3 IS NOT NULL AND $3 IS DISTINCT FROM category THEN true
		        WHEN $7 IS NOT NULL AND $7 IS DISTINCT FROM considered THEN true
		        WHEN $8 IS NOT NULL AND $8 IS DISTINCT FROM notes THEN true
		        ELSE manipulated
		    END,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $9
		RETURNING id, account_id, amount, description, category, original_description, original_category,
		          transaction_date, type, status,
		          provider_created_at, provider_updated_at, created_at, updated_at,
		          considered, is_open_finance, tags, manipulated, notes
	`

	var txn transaction.Transaction
	var providerCreatedAt, providerUpdatedAt sql.NullTime
	var tags []byte
	var originalDescription, originalCategory sql.NullString

	err := r.db.QueryRowContext(
		ctx, query,
		params.Amount, params.Description, params.Category, params.TransactionDate,
		params.Type, params.Status, params.Considered, params.Notes, id,
	).Scan(
		&txn.ID, &txn.AccountID, &txn.Amount,
		&txn.Description, &txn.Category, &originalDescription, &originalCategory,
		&txn.TransactionDate,
		&txn.Type, &txn.Status,
		&providerCreatedAt, &providerUpdatedAt,
		&txn.CreatedAt, &txn.UpdatedAt,
		&txn.Considered, &txn.IsOpenFinance, &tags, &txn.Manipulated, &txn.Notes,
	)

	if providerCreatedAt.Valid {
		txn.ProviderCreatedAt = providerCreatedAt.Time
	}
	if providerUpdatedAt.Valid {
		txn.ProviderUpdatedAt = providerUpdatedAt.Time
	}
	if originalDescription.Valid {
		txn.OriginalDescription = &originalDescription.String
	}
	if originalCategory.Valid {
		txn.OriginalCategory = &originalCategory.String
	}
	txn.Tags = []string{}

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("transaction not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update transaction: %w", err)
	}

	return &txn, nil
}

// UpdateBatch updates multiple transactions in a single query
// Only updates fields that are non-nil in each params entry
func (r *TransactionRepository) UpdateBatch(ctx context.Context, updates []struct {
	ID     string
	Params transaction.UpdateTransactionParams
}) ([]*transaction.Transaction, error) {
	if len(updates) == 0 {
		return nil, nil
	}

	// Process each update individually but in a transaction for consistency
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	results := make([]*transaction.Transaction, 0, len(updates))

	query := `
		UPDATE transactions
		SET description = COALESCE($1, description),
		    original_description = CASE
		        WHEN $1 IS NOT NULL AND $1 IS DISTINCT FROM description AND original_description IS NULL THEN description
		        ELSE original_description
		    END,
		    category = COALESCE($2, category),
		    original_category = CASE
		        WHEN $2 IS NOT NULL AND $2 IS DISTINCT FROM category AND original_category IS NULL THEN category
		        ELSE original_category
		    END,
		    considered = COALESCE($3, considered),
		    notes = COALESCE($4, notes),
		    manipulated = CASE
		        WHEN $1 IS NOT NULL AND $1 IS DISTINCT FROM description THEN true
		        WHEN $2 IS NOT NULL AND $2 IS DISTINCT FROM category THEN true
		        WHEN $3 IS NOT NULL AND $3 IS DISTINCT FROM considered THEN true
		        WHEN $4 IS NOT NULL AND $4 IS DISTINCT FROM notes THEN true
		        ELSE manipulated
		    END,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
		RETURNING id, account_id, amount, description, category, original_description, original_category,
		          transaction_date, type, status,
		          provider_created_at, provider_updated_at, created_at, updated_at,
		          considered, is_open_finance, tags, manipulated, notes
	`

	for _, u := range updates {
		var txn transaction.Transaction
		var providerCreatedAt, providerUpdatedAt sql.NullTime
		var tags []byte
		var originalDescription, originalCategory sql.NullString

		err := tx.QueryRowContext(
			ctx, query,
			u.Params.Description, u.Params.Category, u.Params.Considered, u.Params.Notes, u.ID,
		).Scan(
			&txn.ID, &txn.AccountID, &txn.Amount,
			&txn.Description, &txn.Category, &originalDescription, &originalCategory,
			&txn.TransactionDate,
			&txn.Type, &txn.Status,
			&providerCreatedAt, &providerUpdatedAt,
			&txn.CreatedAt, &txn.UpdatedAt,
			&txn.Considered, &txn.IsOpenFinance, &tags, &txn.Manipulated, &txn.Notes,
		)

		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction %s not found", u.ID)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to update transaction %s: %w", u.ID, err)
		}

		if providerCreatedAt.Valid {
			txn.ProviderCreatedAt = providerCreatedAt.Time
		}
		if providerUpdatedAt.Valid {
			txn.ProviderUpdatedAt = providerUpdatedAt.Time
		}
		if originalDescription.Valid {
			txn.OriginalDescription = &originalDescription.String
		}
		if originalCategory.Valid {
			txn.OriginalCategory = &originalCategory.String
		}
		txn.Tags = []string{}

		results = append(results, &txn)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return results, nil
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
func (r *TransactionRepository) Upsert(ctx context.Context, params transaction.UpsertTransactionParams) (*transaction.Transaction, error) {
	query := `
		INSERT INTO transactions (id, account_id, amount, description, category, original_description, original_category,
		                          transaction_date, type, status, provider_created_at, provider_updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
		    amount = EXCLUDED.amount,
		    description = CASE WHEN transactions.manipulated THEN transactions.description ELSE EXCLUDED.description END,
		    category = CASE WHEN transactions.manipulated THEN transactions.category ELSE EXCLUDED.category END,
		    original_description = EXCLUDED.original_description,
		    original_category = EXCLUDED.original_category,
		    transaction_date = EXCLUDED.transaction_date,
		    type = EXCLUDED.type,
		    status = CASE WHEN transactions.manipulated THEN transactions.status ELSE EXCLUDED.status END,
		    notes = CASE WHEN transactions.manipulated THEN transactions.notes ELSE transactions.notes END,
		    provider_created_at = EXCLUDED.provider_created_at,
		    provider_updated_at = EXCLUDED.provider_updated_at,
		    updated_at = CURRENT_TIMESTAMP
		RETURNING id, account_id, amount, description, category, original_description, original_category,
		          transaction_date, type, status,
		          provider_created_at, provider_updated_at, created_at, updated_at
	`

	var transaction transaction.Transaction
	var providerCreatedAt, providerUpdatedAt sql.NullTime
	var originalDescription, originalCategory sql.NullString

	err := r.db.QueryRowContext(
		ctx, query,
		params.ID, params.AccountID, params.Amount, params.Description, params.Category,
		params.OriginalDescription, params.OriginalCategory,
		params.TransactionDate, params.Type, params.Status,
		params.ProviderCreatedAt, params.ProviderUpdatedAt,
	).Scan(
		&transaction.ID, &transaction.AccountID, &transaction.Amount,
		&transaction.Description, &transaction.Category, &originalDescription, &originalCategory,
		&transaction.TransactionDate,
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
	if originalDescription.Valid {
		transaction.OriginalDescription = &originalDescription.String
	}
	if originalCategory.Valid {
		transaction.OriginalCategory = &originalCategory.String
	}

	if err != nil {
		return nil, fmt.Errorf("failed to upsert transaction: %w", err)
	}

	return &transaction, nil
}

// UpsertBatch inserts or updates multiple transactions in a single query
// Returns the count of affected rows (inserted + updated)
func (r *TransactionRepository) UpsertBatch(ctx context.Context, params []transaction.UpsertTransactionParams) (int64, error) {
	if len(params) == 0 {
		return 0, nil
	}

	// Build the VALUES clause with placeholders
	// Each transaction has 12 fields
	valueStrings := make([]string, 0, len(params))
	valueArgs := make([]any, 0, len(params)*12)

	for i, param := range params {
		// Calculate placeholder positions for this row
		offset := i * 12
		valueStrings = append(valueStrings, fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			offset+1, offset+2, offset+3, offset+4, offset+5,
			offset+6, offset+7, offset+8, offset+9, offset+10, offset+11, offset+12,
		))

		// Add values for this row
		valueArgs = append(valueArgs,
			param.ID, param.AccountID, param.Amount, param.Description, param.Category,
			param.OriginalDescription, param.OriginalCategory,
			param.TransactionDate, param.Type, param.Status,
			param.ProviderCreatedAt, param.ProviderUpdatedAt,
		)
	}

	query := fmt.Sprintf(`
		INSERT INTO transactions (id, account_id, amount, description, category, original_description, original_category,
		                          transaction_date, type, status, provider_created_at, provider_updated_at)
		VALUES %s
		ON CONFLICT (id) DO UPDATE SET
		    amount = EXCLUDED.amount,
		    description = CASE WHEN transactions.manipulated THEN transactions.description ELSE EXCLUDED.description END,
		    category = CASE WHEN transactions.manipulated THEN transactions.category ELSE EXCLUDED.category END,
		    original_description = EXCLUDED.original_description,
		    original_category = EXCLUDED.original_category,
		    transaction_date = EXCLUDED.transaction_date,
		    type = EXCLUDED.type,
		    status = CASE WHEN transactions.manipulated THEN transactions.status ELSE EXCLUDED.status END,
		    notes = CASE WHEN transactions.manipulated THEN transactions.notes ELSE transactions.notes END,
		    provider_created_at = EXCLUDED.provider_created_at,
		    provider_updated_at = EXCLUDED.provider_updated_at,
		    updated_at = CURRENT_TIMESTAMP
		WHERE
		    transactions.amount IS DISTINCT FROM EXCLUDED.amount OR
		    (NOT transactions.manipulated AND transactions.description IS DISTINCT FROM EXCLUDED.description) OR
		    (NOT transactions.manipulated AND transactions.category IS DISTINCT FROM EXCLUDED.category) OR
		    transactions.original_description IS DISTINCT FROM EXCLUDED.original_description OR
		    transactions.original_category IS DISTINCT FROM EXCLUDED.original_category OR
		    transactions.transaction_date IS DISTINCT FROM EXCLUDED.transaction_date OR
		    transactions.type IS DISTINCT FROM EXCLUDED.type OR
		    (NOT transactions.manipulated AND transactions.status IS DISTINCT FROM EXCLUDED.status) OR
		    transactions.provider_created_at IS DISTINCT FROM EXCLUDED.provider_created_at OR
		    transactions.provider_updated_at IS DISTINCT FROM EXCLUDED.provider_updated_at
	`, strings.Join(valueStrings, ", "))

	result, err := r.db.ExecContext(ctx, query, valueArgs...)
	if err != nil {
		return 0, fmt.Errorf("failed to batch upsert transactions: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return affected, nil
}
