package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"parsa/internal/domain/bill"
)

type BillRepository struct {
	db *DB
}

func NewBillRepository(db *DB) *BillRepository {
	return &BillRepository{db: db}
}

func (r *BillRepository) Create(ctx context.Context, params bill.CreateParams) (*bill.Bill, error) {
	query := `
		INSERT INTO bills (id, account_id, amount, due_date, status, description, biller_name,
		                   category, barcode, digitable_line, payment_date, related_transaction_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, account_id, amount, due_date, status, description, biller_name,
		          category, barcode, digitable_line, payment_date, related_transaction_id,
		          provider_created_at, provider_updated_at, created_at, updated_at, is_open_finance
	`

	var b bill.Bill
	var providerCreatedAt, providerUpdatedAt, paymentDate sql.NullTime
	var category, barcode, digitableLine, relatedTxID sql.NullString

	err := r.db.QueryRowContext(
		ctx, query,
		params.ID, params.AccountID, params.Amount, params.DueDate, params.Status,
		params.Description, params.BillerName, params.Category, params.Barcode,
		params.DigitableLine, params.PaymentDate, params.RelatedTransactionID,
	).Scan(
		&b.ID, &b.AccountID, &b.Amount, &b.DueDate, &b.Status, &b.Description, &b.BillerName,
		&category, &barcode, &digitableLine, &paymentDate, &relatedTxID,
		&providerCreatedAt, &providerUpdatedAt, &b.CreatedAt, &b.UpdatedAt, &b.IsOpenFinance,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create bill: %w", err)
	}

	applyNullableBillFields(&b, category, barcode, digitableLine, paymentDate, relatedTxID,
		providerCreatedAt, providerUpdatedAt)

	return &b, nil
}

func (r *BillRepository) GetByID(ctx context.Context, id string) (*bill.Bill, error) {
	query := `
		SELECT id, account_id, amount, due_date, status, description, biller_name,
		       category, barcode, digitable_line, payment_date, related_transaction_id,
		       provider_created_at, provider_updated_at, created_at, updated_at, is_open_finance
		FROM bills
		WHERE id = $1
	`

	var b bill.Bill
	var providerCreatedAt, providerUpdatedAt, paymentDate sql.NullTime
	var category, barcode, digitableLine, relatedTxID sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&b.ID, &b.AccountID, &b.Amount, &b.DueDate, &b.Status, &b.Description, &b.BillerName,
		&category, &barcode, &digitableLine, &paymentDate, &relatedTxID,
		&providerCreatedAt, &providerUpdatedAt, &b.CreatedAt, &b.UpdatedAt, &b.IsOpenFinance,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get bill: %w", err)
	}

	applyNullableBillFields(&b, category, barcode, digitableLine, paymentDate, relatedTxID,
		providerCreatedAt, providerUpdatedAt)

	return &b, nil
}

func (r *BillRepository) ListByAccountID(ctx context.Context, accountID string, limit, offset int) ([]*bill.Bill, error) {
	query := `
		SELECT id, account_id, amount, due_date, status, description, biller_name,
		       category, barcode, digitable_line, payment_date, related_transaction_id,
		       provider_created_at, provider_updated_at, created_at, updated_at, is_open_finance
		FROM bills
		WHERE account_id = $1
		ORDER BY due_date DESC, created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, accountID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list bills: %w", err)
	}
	defer rows.Close()

	return scanBills(rows)
}

func (r *BillRepository) ListByUserID(ctx context.Context, userID int64, limit, offset int) ([]*bill.Bill, error) {
	query := `
		SELECT b.id, b.account_id, b.amount, b.due_date, b.status, b.description, b.biller_name,
		       b.category, b.barcode, b.digitable_line, b.payment_date, b.related_transaction_id,
		       b.provider_created_at, b.provider_updated_at, b.created_at, b.updated_at, b.is_open_finance
		FROM bills b
		JOIN accounts a ON b.account_id = a.id
		WHERE a.user_id = $1
		ORDER BY b.due_date DESC, b.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list bills by user: %w", err)
	}
	defer rows.Close()

	return scanBills(rows)
}

func (r *BillRepository) CountByUserID(ctx context.Context, userID int64) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM bills b
		JOIN accounts a ON b.account_id = a.id
		WHERE a.user_id = $1
	`

	var count int64
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count bills: %w", err)
	}

	return count, nil
}

func scanBills(rows *sql.Rows) ([]*bill.Bill, error) {
	var bills []*bill.Bill
	for rows.Next() {
		var b bill.Bill
		var providerCreatedAt, providerUpdatedAt, paymentDate sql.NullTime
		var category, barcode, digitableLine, relatedTxID sql.NullString

		err := rows.Scan(
			&b.ID, &b.AccountID, &b.Amount, &b.DueDate, &b.Status, &b.Description, &b.BillerName,
			&category, &barcode, &digitableLine, &paymentDate, &relatedTxID,
			&providerCreatedAt, &providerUpdatedAt, &b.CreatedAt, &b.UpdatedAt, &b.IsOpenFinance,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bill: %w", err)
		}

		applyNullableBillFields(&b, category, barcode, digitableLine, paymentDate, relatedTxID,
			providerCreatedAt, providerUpdatedAt)

		bills = append(bills, &b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating bills: %w", err)
	}

	return bills, nil
}

func applyNullableBillFields(b *bill.Bill, category, barcode, digitableLine, relatedTxID sql.NullString,
	paymentDate, providerCreatedAt, providerUpdatedAt sql.NullTime) {
	if category.Valid {
		b.Category = &category.String
	}
	if barcode.Valid {
		b.Barcode = &barcode.String
	}
	if digitableLine.Valid {
		b.DigitableLine = &digitableLine.String
	}
	if relatedTxID.Valid {
		b.RelatedTransactionID = &relatedTxID.String
	}
	if paymentDate.Valid {
		b.PaymentDate = &paymentDate.Time
	}
	if providerCreatedAt.Valid {
		b.ProviderCreatedAt = providerCreatedAt.Time
	}
	if providerUpdatedAt.Valid {
		b.ProviderUpdatedAt = providerUpdatedAt.Time
	}
}

func (r *BillRepository) Update(ctx context.Context, id string, params bill.UpdateParams) (*bill.Bill, error) {
	query := `
		UPDATE bills
		SET amount = COALESCE($1, amount),
		    due_date = COALESCE($2, due_date),
		    status = COALESCE($3, status),
		    description = COALESCE($4, description),
		    biller_name = COALESCE($5, biller_name),
		    category = COALESCE($6, category),
		    payment_date = COALESCE($7, payment_date),
		    related_transaction_id = COALESCE($8, related_transaction_id),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $9
		RETURNING id, account_id, amount, due_date, status, description, biller_name,
		          category, barcode, digitable_line, payment_date, related_transaction_id,
		          provider_created_at, provider_updated_at, created_at, updated_at, is_open_finance
	`

	var b bill.Bill
	var providerCreatedAt, providerUpdatedAt, paymentDate sql.NullTime
	var category, barcode, digitableLine, relatedTxID sql.NullString

	err := r.db.QueryRowContext(
		ctx, query,
		params.Amount, params.DueDate, params.Status, params.Description, params.BillerName,
		params.Category, params.PaymentDate, params.RelatedTransactionID, id,
	).Scan(
		&b.ID, &b.AccountID, &b.Amount, &b.DueDate, &b.Status, &b.Description, &b.BillerName,
		&category, &barcode, &digitableLine, &paymentDate, &relatedTxID,
		&providerCreatedAt, &providerUpdatedAt, &b.CreatedAt, &b.UpdatedAt, &b.IsOpenFinance,
	)

	if err == sql.ErrNoRows {
		return nil, bill.ErrBillNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update bill: %w", err)
	}

	applyNullableBillFields(&b, category, barcode, digitableLine, paymentDate, relatedTxID,
		providerCreatedAt, providerUpdatedAt)

	return &b, nil
}

func (r *BillRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM bills WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete bill: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return bill.ErrBillNotFound
	}

	return nil
}

func (r *BillRepository) Upsert(ctx context.Context, params bill.UpsertParams) (*bill.Bill, error) {
	query := `
		INSERT INTO bills (id, account_id, amount, due_date, status, description, biller_name,
		                   category, barcode, digitable_line, payment_date, related_transaction_id,
		                   provider_created_at, provider_updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO UPDATE SET
		    amount = EXCLUDED.amount,
		    due_date = EXCLUDED.due_date,
		    status = EXCLUDED.status,
		    description = EXCLUDED.description,
		    biller_name = EXCLUDED.biller_name,
		    category = EXCLUDED.category,
		    barcode = EXCLUDED.barcode,
		    digitable_line = EXCLUDED.digitable_line,
		    payment_date = EXCLUDED.payment_date,
		    related_transaction_id = EXCLUDED.related_transaction_id,
		    provider_created_at = EXCLUDED.provider_created_at,
		    provider_updated_at = EXCLUDED.provider_updated_at,
		    updated_at = CURRENT_TIMESTAMP
		RETURNING id, account_id, amount, due_date, status, description, biller_name,
		          category, barcode, digitable_line, payment_date, related_transaction_id,
		          provider_created_at, provider_updated_at, created_at, updated_at, is_open_finance
	`

	var b bill.Bill
	var providerCreatedAt, providerUpdatedAt, paymentDate sql.NullTime
	var category, barcode, digitableLine, relatedTxID sql.NullString

	err := r.db.QueryRowContext(
		ctx, query,
		params.ID, params.AccountID, params.Amount, params.DueDate, params.Status,
		params.Description, params.BillerName, params.Category, params.Barcode,
		params.DigitableLine, params.PaymentDate, params.RelatedTransactionID,
		params.ProviderCreatedAt, params.ProviderUpdatedAt,
	).Scan(
		&b.ID, &b.AccountID, &b.Amount, &b.DueDate, &b.Status, &b.Description, &b.BillerName,
		&category, &barcode, &digitableLine, &paymentDate, &relatedTxID,
		&providerCreatedAt, &providerUpdatedAt, &b.CreatedAt, &b.UpdatedAt, &b.IsOpenFinance,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert bill: %w", err)
	}

	applyNullableBillFields(&b, category, barcode, digitableLine, paymentDate, relatedTxID,
		providerCreatedAt, providerUpdatedAt)

	return &b, nil
}
