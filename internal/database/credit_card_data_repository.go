package database

import (
	"context"
	"database/sql"
	"fmt"

	"parsa/internal/models"
)

type CreditCardDataRepository struct {
	db *DB
}

func NewCreditCardDataRepository(db *DB) *CreditCardDataRepository {
	return &CreditCardDataRepository{db: db}
}

// Upsert creates or updates credit card data for a transaction
func (r *CreditCardDataRepository) Upsert(ctx context.Context, transactionID string, params models.CreateCreditCardDataParams) (*models.CreditCardData, error) {
	// First try to find existing record
	query := `
		INSERT INTO credit_card_data (transaction_id, purchase_date, installment_number, total_installments)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (transaction_id) DO UPDATE SET
		    purchase_date = EXCLUDED.purchase_date,
		    installment_number = EXCLUDED.installment_number,
		    total_installments = EXCLUDED.total_installments
		RETURNING id, purchase_date, installment_number, total_installments
	`

	var data models.CreditCardData
	err := r.db.QueryRowContext(
		ctx, query,
		transactionID, params.PurchaseDate, params.InstallmentNumber, params.TotalInstallments,
	).Scan(
		&data.ID, &data.PurchaseDate, &data.InstallmentNumber, &data.TotalInstallments,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert credit card data: %w", err)
	}

	return &data, nil
}

// GetByTransactionID retrieves credit card data for a transaction
func (r *CreditCardDataRepository) GetByTransactionID(ctx context.Context, transactionID string) (*models.CreditCardData, error) {
	query := `
		SELECT id, purchase_date, installment_number, total_installments
		FROM credit_card_data
		WHERE transaction_id = $1
	`

	var data models.CreditCardData
	err := r.db.QueryRowContext(ctx, query, transactionID).Scan(
		&data.ID, &data.PurchaseDate, &data.InstallmentNumber, &data.TotalInstallments,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get credit card data: %w", err)
	}

	return &data, nil
}

// Delete removes credit card data for a transaction
func (r *CreditCardDataRepository) Delete(ctx context.Context, transactionID string) error {
	query := `DELETE FROM credit_card_data WHERE transaction_id = $1`

	_, err := r.db.ExecContext(ctx, query, transactionID)
	if err != nil {
		return fmt.Errorf("failed to delete credit card data: %w", err)
	}

	return nil
}

