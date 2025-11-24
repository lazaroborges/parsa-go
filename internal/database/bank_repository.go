package database

import (
	"context"
	"database/sql"
	"fmt"

	"parsa/internal/models"
)

type BankRepository struct {
	db *DB
}

func NewBankRepository(db *DB) *BankRepository {
	return &BankRepository{db: db}
}

// FindOrCreateByConnector finds a bank by connector code or creates it if it doesn't exist
func (r *BankRepository) FindOrCreateByConnector(ctx context.Context, name, connector string) (*models.Bank, error) {
	// Try to find existing bank by connector
	query := `
		SELECT id, name, connector
		FROM banks
		WHERE connector = $1
	`

	var bank models.Bank
	err := r.db.QueryRowContext(ctx, query, connector).Scan(
		&bank.ID, &bank.Name, &bank.Connector,
	)

	if err == nil {
		// Bank found, return it
		return &bank, nil
	}

	if err != sql.ErrNoRows {
		// Unexpected error
		return nil, fmt.Errorf("failed to query bank: %w", err)
	}

	// Bank not found, create it
	insertQuery := `
		INSERT INTO banks (name, connector)
		VALUES ($1, $2)
		RETURNING id, name, connector
	`

	err = r.db.QueryRowContext(ctx, insertQuery, name, connector).Scan(
		&bank.ID, &bank.Name, &bank.Connector,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create bank: %w", err)
	}

	return &bank, nil
}

// GetByID retrieves a bank by its ID
func (r *BankRepository) GetByID(ctx context.Context, id int64) (*models.Bank, error) {
	query := `
		SELECT id, name, connector
		FROM banks
		WHERE id = $1
	`

	var bank models.Bank
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&bank.ID, &bank.Name, &bank.Connector,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("bank not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get bank: %w", err)
	}

	return &bank, nil
}

// List retrieves all banks
func (r *BankRepository) List(ctx context.Context) ([]*models.Bank, error) {
	query := `
		SELECT id, name, connector
		FROM banks
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list banks: %w", err)
	}
	defer rows.Close()

	var banks []*models.Bank
	for rows.Next() {
		var bank models.Bank
		err := rows.Scan(&bank.ID, &bank.Name, &bank.Connector)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bank: %w", err)
		}
		banks = append(banks, &bank)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating banks: %w", err)
	}

	return banks, nil
}
