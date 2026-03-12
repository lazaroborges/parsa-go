package postgres

import (
	"context"
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
		`INSERT INTO merchants (name) VALUES ($1)
		 ON CONFLICT ((LOWER(name))) DO UPDATE SET name = merchants.name
		 RETURNING id, name`,
		name,
	).Scan(&m.ID, &m.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create merchant: %w", err)
	}

	return &m, nil
}
