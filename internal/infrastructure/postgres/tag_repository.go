package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"parsa/internal/domain/tag"
)

type TagRepository struct {
	db *DB
}

func NewTagRepository(db *DB) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) Create(ctx context.Context, userID int64, params tag.CreateTagParams) (*tag.Tag, error) {
	displayOrder := 99
	if params.DisplayOrder != nil {
		displayOrder = *params.DisplayOrder
	}

	description := ""
	if params.Description != nil {
		description = *params.Description
	}

	query := `
		INSERT INTO tags (user_id, name, color, display_order, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, name, color, display_order, description, created_at, updated_at
	`

	var t tag.Tag
	err := r.db.QueryRowContext(
		ctx, query,
		userID, params.Name, params.Color, displayOrder, description,
	).Scan(
		&t.ID, &t.UserID, &t.Name, &t.Color, &t.DisplayOrder, &t.Description,
		&t.CreatedAt, &t.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}

	return &t, nil
}

func (r *TagRepository) GetByID(ctx context.Context, id string) (*tag.Tag, error) {
	query := `
		SELECT id, user_id, name, color, display_order, description, created_at, updated_at
		FROM tags
		WHERE id = $1
	`

	var t tag.Tag
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID, &t.UserID, &t.Name, &t.Color, &t.DisplayOrder, &t.Description,
		&t.CreatedAt, &t.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}

	return &t, nil
}

func (r *TagRepository) ListByUserID(ctx context.Context, userID int64) ([]*tag.Tag, error) {
	query := `
		SELECT id, user_id, name, color, display_order, description, created_at, updated_at
		FROM tags
		WHERE user_id = $1
		ORDER BY display_order ASC, name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	defer rows.Close()

	var tags []*tag.Tag
	for rows.Next() {
		var t tag.Tag
		err := rows.Scan(
			&t.ID, &t.UserID, &t.Name, &t.Color, &t.DisplayOrder, &t.Description,
			&t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, &t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tags: %w", err)
	}

	return tags, nil
}

func (r *TagRepository) Update(ctx context.Context, id string, params tag.UpdateTagParams) (*tag.Tag, error) {
	query := `
		UPDATE tags
		SET name = COALESCE($1, name),
		    color = COALESCE($2, color),
		    display_order = COALESCE($3, display_order),
		    description = COALESCE($4, description),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
		RETURNING id, user_id, name, color, display_order, description, created_at, updated_at
	`

	var t tag.Tag
	err := r.db.QueryRowContext(
		ctx, query,
		params.Name, params.Color, params.DisplayOrder, params.Description, id,
	).Scan(
		&t.ID, &t.UserID, &t.Name, &t.Color, &t.DisplayOrder, &t.Description,
		&t.CreatedAt, &t.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, tag.ErrTagNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update tag: %w", err)
	}

	return &t, nil
}

func (r *TagRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM tags WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return tag.ErrTagNotFound
	}

	return nil
}
