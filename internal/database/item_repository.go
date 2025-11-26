package database

import (
	"context"
	"fmt"

	"parsa/internal/models"
)

type ItemRepository struct {
	db *DB
}

func NewItemRepository(db *DB) *ItemRepository {
	return &ItemRepository{db: db}
}

// FindOrCreate performs an upsert on the items table.
// The items.id column is globally unique (PRIMARY KEY constraint in schema).
// On conflict (id already exists):
//   - The existing row's user_id is preserved (NOT updated from the incoming userID parameter)
//   - Only updated_at is refreshed to CURRENT_TIMESTAMP
//
// This means the userID parameter is only used for initial insertion;
// subsequent calls with the same id but different userID will NOT change ownership.
func (r *ItemRepository) FindOrCreate(ctx context.Context, id string, userID int64) (*models.Item, error) {
	// Try to insert, on conflict do nothing and return existing
	query := `
		INSERT INTO items (id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET updated_at = CURRENT_TIMESTAMP
		RETURNING id, user_id, created_at, updated_at
	`

	var item models.Item
	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(
		&item.ID, &item.UserID, &item.CreatedAt, &item.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to find or create item: %w", err)
	}

	return &item, nil
}

// ListByUserID retrieves all items for a user
func (r *ItemRepository) ListByUserID(ctx context.Context, userID int64) ([]*models.Item, error) {
	query := `
		SELECT id, user_id, created_at, updated_at
		FROM items
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		var item models.Item
		err := rows.Scan(
			&item.ID, &item.UserID, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		items = append(items, &item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating items: %w", err)
	}

	return items, nil
}

// Delete removes an item by ID
func (r *ItemRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM items WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("item not found")
	}

	return nil
}
