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
		`SELECT id, type, number, business_name FROM documents WHERE LOWER(business_name) = LOWER($1)`,
		businessName,
	).Scan(&d.ID, &d.Type, &number, &d.BusinessName)

	if err == nil {
		if number.Valid {
			d.Number = &number.String
		}
		return &d, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to find document: %w", err)
	}

	err = r.db.QueryRowContext(ctx,
		`INSERT INTO documents (type, business_name) VALUES ($1, $2) RETURNING id, type, number, business_name`,
		models.DocumentTypeCNPJ, businessName,
	).Scan(&d.ID, &d.Type, &number, &d.BusinessName)
	if err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}
	if number.Valid {
		d.Number = &number.String
	}

	return &d, nil
}
