package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"parsa/internal/models"
)

type MerchantRepository struct {
	db *DB
}

func NewMerchantRepository(db *DB) *MerchantRepository {
	return &MerchantRepository{db: db}
}

func (r *MerchantRepository) FindOrCreateByName(ctx context.Context, name string) (*models.Merchant, error) {
	var m models.Merchant
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name FROM merchants WHERE LOWER(name) = LOWER($1)`,
		name,
	).Scan(&m.ID, &m.Name)

	if err == nil {
		return &m, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to find merchant: %w", err)
	}

	err = r.db.QueryRowContext(ctx,
		`INSERT INTO merchants (name) VALUES ($1) RETURNING id, name`,
		name,
	).Scan(&m.ID, &m.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create merchant: %w", err)
	}

	return &m, nil
}
