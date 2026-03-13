package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"parsa/internal/models"
)

type DocumentRepository struct {
	db *DB
}

func NewDocumentRepository(db *DB) *DocumentRepository {
	return &DocumentRepository{db: db}
}

func (r *DocumentRepository) FindOrCreateByBusinessName(ctx context.Context, businessName string) (*models.Document, error) {
	var d models.Document
	var number sql.NullString
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO documents (type, business_name) VALUES ($1, $2)
		ON CONFLICT (LOWER(business_name)) WHERE business_name IS NOT NULL
		DO UPDATE SET business_name = documents.business_name
		RETURNING id, type, number, business_name`,
		models.DocumentTypeCNPJ, businessName,
	).Scan(&d.ID, &d.Type, &number, &d.BusinessName)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create document: %w", err)
	}
	if number.Valid {
		d.Number = &number.String
	}
	return &d, nil
}
