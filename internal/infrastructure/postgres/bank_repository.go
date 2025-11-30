package postgres

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
		SELECT id, name, ui_name, connector
		FROM banks
		WHERE connector = $1
	`

	var bank models.Bank
	var uiName sql.NullString
	err := r.db.QueryRowContext(ctx, query, connector).Scan(
		&bank.ID, &bank.Name, &uiName, &bank.Connector,
	)

	if err == nil {
		bank.UIName = uiName.String
		return &bank, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query bank: %w", err)
	}

	// Bank not found, create it
	insertQuery := `
		INSERT INTO banks (name, connector)
		VALUES ($1, $2)
		RETURNING id, name, ui_name, connector
	`

	err = r.db.QueryRowContext(ctx, insertQuery, name, connector).Scan(
		&bank.ID, &bank.Name, &uiName, &bank.Connector,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create bank: %w", err)
	}

	bank.UIName = uiName.String
	return &bank, nil
}

// FindOrCreateByName finds a bank by name or creates it if it doesn't exist
func (r *BankRepository) FindOrCreateByName(ctx context.Context, name string) (*models.Bank, error) {
	query := `
		SELECT id, name, ui_name, connector
		FROM banks
		WHERE name = $1
	`

	var bank models.Bank
	var uiName, connector sql.NullString
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&bank.ID, &bank.Name, &uiName, &connector,
	)

	if err == nil {
		bank.UIName = uiName.String
		bank.Connector = connector.String
		return &bank, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query bank: %w", err)
	}

	// Bank not found, create it (connector will be empty)
	insertQuery := `
		INSERT INTO banks (name)
		VALUES ($1)
		RETURNING id, name, ui_name, connector
	`

	err = r.db.QueryRowContext(ctx, insertQuery, name).Scan(
		&bank.ID, &bank.Name, &uiName, &connector,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create bank: %w", err)
	}

	bank.UIName = uiName.String
	bank.Connector = connector.String
	return &bank, nil
}

// GetByID retrieves a bank by its ID
func (r *BankRepository) GetByID(ctx context.Context, id int64) (*models.Bank, error) {
	query := `
		SELECT id, name, ui_name, connector
		FROM banks
		WHERE id = $1
	`

	var bank models.Bank
	var uiName, connector sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&bank.ID, &bank.Name, &uiName, &connector,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("bank not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get bank: %w", err)
	}

	bank.UIName = uiName.String
	bank.Connector = connector.String
	return &bank, nil
}

// List retrieves all banks
func (r *BankRepository) List(ctx context.Context) ([]*models.Bank, error) {
	query := `
		SELECT id, name, ui_name, connector
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
		var uiName, connector sql.NullString
		err := rows.Scan(&bank.ID, &bank.Name, &uiName, &connector)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bank: %w", err)
		}
		bank.UIName = uiName.String
		bank.Connector = connector.String
		banks = append(banks, &bank)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating banks: %w", err)
	}

	return banks, nil
}
