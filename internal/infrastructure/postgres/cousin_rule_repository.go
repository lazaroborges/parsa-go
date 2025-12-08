package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"parsa/internal/domain/cousinrule"
)

type CousinRuleRepository struct {
	db *DB
}

func NewCousinRuleRepository(db *DB) *CousinRuleRepository {
	return &CousinRuleRepository{db: db}
}

func (r *CousinRuleRepository) Create(ctx context.Context, params cousinrule.CreateCousinRuleParams) (*cousinrule.CousinRule, error) {
	query := `
		INSERT INTO user_ck_values (user_id, cousin_id, type, category, description, notes, considered, dont_ask_again)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, user_id, cousin_id, type, category, description, notes, considered, dont_ask_again, created_at, updated_at
	`

	var rule cousinrule.CousinRule
	var txType, category, description, notes sql.NullString
	var considered sql.NullBool

	err := r.db.QueryRowContext(ctx, query,
		params.UserID, params.CousinID, params.Type, params.Category,
		params.Description, params.Notes, params.Considered, params.DontAskAgain,
	).Scan(
		&rule.ID, &rule.UserID, &rule.CousinID, &txType,
		&category, &description, &notes, &considered,
		&rule.DontAskAgain, &rule.CreatedAt, &rule.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create cousin rule: %w", err)
	}

	if txType.Valid {
		rule.Type = &txType.String
	}
	if category.Valid {
		rule.Category = &category.String
	}
	if description.Valid {
		rule.Description = &description.String
	}
	if notes.Valid {
		rule.Notes = &notes.String
	}
	if considered.Valid {
		rule.Considered = &considered.Bool
	}
	rule.Tags = []string{}

	// Set tags if provided
	if len(params.Tags) > 0 {
		if err := r.SetRuleTags(ctx, rule.ID, params.Tags); err != nil {
			return nil, fmt.Errorf("failed to set rule tags: %w", err)
		}
		rule.Tags = params.Tags
	}

	return &rule, nil
}

func (r *CousinRuleRepository) GetByID(ctx context.Context, id int64) (*cousinrule.CousinRule, error) {
	query := `
		SELECT id, user_id, cousin_id, type, category, description, notes, considered, dont_ask_again, created_at, updated_at
		FROM user_ck_values
		WHERE id = $1
	`

	var rule cousinrule.CousinRule
	var txType, category, description, notes sql.NullString
	var considered sql.NullBool

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&rule.ID, &rule.UserID, &rule.CousinID, &txType,
		&category, &description, &notes, &considered,
		&rule.DontAskAgain, &rule.CreatedAt, &rule.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cousin rule: %w", err)
	}

	if txType.Valid {
		rule.Type = &txType.String
	}
	if category.Valid {
		rule.Category = &category.String
	}
	if description.Valid {
		rule.Description = &description.String
	}
	if notes.Valid {
		rule.Notes = &notes.String
	}
	if considered.Valid {
		rule.Considered = &considered.Bool
	}
	rule.Tags = []string{}

	return &rule, nil
}

func (r *CousinRuleRepository) GetByCousinAndType(ctx context.Context, userID, cousinID int64, txType *string) (*cousinrule.CousinRule, error) {
	var query string
	var args []any

	if txType == nil {
		query = `
			SELECT id, user_id, cousin_id, type, category, description, notes, considered, dont_ask_again, created_at, updated_at
			FROM user_ck_values
			WHERE user_id = $1 AND cousin_id = $2 AND type IS NULL
		`
		args = []any{userID, cousinID}
	} else {
		query = `
			SELECT id, user_id, cousin_id, type, category, description, notes, considered, dont_ask_again, created_at, updated_at
			FROM user_ck_values
			WHERE user_id = $1 AND cousin_id = $2 AND type = $3
		`
		args = []any{userID, cousinID, *txType}
	}

	var rule cousinrule.CousinRule
	var ruleType, category, description, notes sql.NullString
	var considered sql.NullBool

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&rule.ID, &rule.UserID, &rule.CousinID, &ruleType,
		&category, &description, &notes, &considered,
		&rule.DontAskAgain, &rule.CreatedAt, &rule.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cousin rule by cousin and type: %w", err)
	}

	if ruleType.Valid {
		rule.Type = &ruleType.String
	}
	if category.Valid {
		rule.Category = &category.String
	}
	if description.Valid {
		rule.Description = &description.String
	}
	if notes.Valid {
		rule.Notes = &notes.String
	}
	if considered.Valid {
		rule.Considered = &considered.Bool
	}
	rule.Tags = []string{}

	return &rule, nil
}

func (r *CousinRuleRepository) ListByUserID(ctx context.Context, userID int64) ([]*cousinrule.CousinRule, error) {
	query := `
		SELECT id, user_id, cousin_id, type, category, description, notes, considered, dont_ask_again, created_at, updated_at
		FROM user_ck_values
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list cousin rules: %w", err)
	}
	defer rows.Close()

	return scanCousinRules(rows)
}

func (r *CousinRuleRepository) ListByCousinID(ctx context.Context, userID, cousinID int64) ([]*cousinrule.CousinRule, error) {
	query := `
		SELECT id, user_id, cousin_id, type, category, description, notes, considered, dont_ask_again, created_at, updated_at
		FROM user_ck_values
		WHERE user_id = $1 AND cousin_id = $2
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID, cousinID)
	if err != nil {
		return nil, fmt.Errorf("failed to list cousin rules by cousin: %w", err)
	}
	defer rows.Close()

	return scanCousinRules(rows)
}

func scanCousinRules(rows *sql.Rows) ([]*cousinrule.CousinRule, error) {
	var rules []*cousinrule.CousinRule
	for rows.Next() {
		var rule cousinrule.CousinRule
		var txType, category, description, notes sql.NullString
		var considered sql.NullBool

		err := rows.Scan(
			&rule.ID, &rule.UserID, &rule.CousinID, &txType,
			&category, &description, &notes, &considered,
			&rule.DontAskAgain, &rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan cousin rule: %w", err)
		}

		if txType.Valid {
			rule.Type = &txType.String
		}
		if category.Valid {
			rule.Category = &category.String
		}
		if description.Valid {
			rule.Description = &description.String
		}
		if notes.Valid {
			rule.Notes = &notes.String
		}
		if considered.Valid {
			rule.Considered = &considered.Bool
		}
		rule.Tags = []string{}

		rules = append(rules, &rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating cousin rules: %w", err)
	}

	return rules, nil
}

func (r *CousinRuleRepository) Update(ctx context.Context, id int64, params cousinrule.UpdateCousinRuleParams) (*cousinrule.CousinRule, error) {
	query := `
		UPDATE user_ck_values
		SET category = COALESCE($1, category),
		    description = COALESCE($2, description),
		    notes = COALESCE($3, notes),
		    considered = COALESCE($4, considered),
		    dont_ask_again = COALESCE($5, dont_ask_again),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $6
		RETURNING id, user_id, cousin_id, type, category, description, notes, considered, dont_ask_again, created_at, updated_at
	`

	var rule cousinrule.CousinRule
	var txType, category, description, notes sql.NullString
	var considered sql.NullBool

	err := r.db.QueryRowContext(ctx, query,
		params.Category, params.Description, params.Notes, params.Considered, params.DontAskAgain, id,
	).Scan(
		&rule.ID, &rule.UserID, &rule.CousinID, &txType,
		&category, &description, &notes, &considered,
		&rule.DontAskAgain, &rule.CreatedAt, &rule.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, cousinrule.ErrRuleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update cousin rule: %w", err)
	}

	if txType.Valid {
		rule.Type = &txType.String
	}
	if category.Valid {
		rule.Category = &category.String
	}
	if description.Valid {
		rule.Description = &description.String
	}
	if notes.Valid {
		rule.Notes = &notes.String
	}
	if considered.Valid {
		rule.Considered = &considered.Bool
	}
	rule.Tags = []string{}

	// Update tags if provided
	if params.Tags != nil {
		if err := r.SetRuleTags(ctx, rule.ID, *params.Tags); err != nil {
			return nil, fmt.Errorf("failed to set rule tags: %w", err)
		}
		rule.Tags = *params.Tags
	}

	return &rule, nil
}

func (r *CousinRuleRepository) Upsert(ctx context.Context, params cousinrule.CreateCousinRuleParams) (*cousinrule.CousinRule, bool, error) {
	// Use INSERT ... ON CONFLICT to handle upsert
	query := `
		INSERT INTO user_ck_values (user_id, cousin_id, type, category, description, notes, considered, dont_ask_again)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id, cousin_id, type) DO UPDATE SET
		    category = COALESCE(EXCLUDED.category, user_ck_values.category),
		    description = COALESCE(EXCLUDED.description, user_ck_values.description),
		    notes = COALESCE(EXCLUDED.notes, user_ck_values.notes),
		    considered = COALESCE(EXCLUDED.considered, user_ck_values.considered),
		    dont_ask_again = COALESCE(EXCLUDED.dont_ask_again, user_ck_values.dont_ask_again),
		    updated_at = CURRENT_TIMESTAMP
		RETURNING id, user_id, cousin_id, type, category, description, notes, considered, dont_ask_again, created_at, updated_at,
		          (xmax = 0) AS was_inserted
	`

	var rule cousinrule.CousinRule
	var txType, category, description, notes sql.NullString
	var considered sql.NullBool
	var wasInserted bool

	err := r.db.QueryRowContext(ctx, query,
		params.UserID, params.CousinID, params.Type, params.Category,
		params.Description, params.Notes, params.Considered, params.DontAskAgain,
	).Scan(
		&rule.ID, &rule.UserID, &rule.CousinID, &txType,
		&category, &description, &notes, &considered,
		&rule.DontAskAgain, &rule.CreatedAt, &rule.UpdatedAt, &wasInserted,
	)

	if err != nil {
		return nil, false, fmt.Errorf("failed to upsert cousin rule: %w", err)
	}

	if txType.Valid {
		rule.Type = &txType.String
	}
	if category.Valid {
		rule.Category = &category.String
	}
	if description.Valid {
		rule.Description = &description.String
	}
	if notes.Valid {
		rule.Notes = &notes.String
	}
	if considered.Valid {
		rule.Considered = &considered.Bool
	}
	rule.Tags = []string{}

	// Set tags if provided
	if len(params.Tags) > 0 {
		if err := r.SetRuleTags(ctx, rule.ID, params.Tags); err != nil {
			return nil, false, fmt.Errorf("failed to set rule tags: %w", err)
		}
		rule.Tags = params.Tags
	}

	return &rule, wasInserted, nil
}

func (r *CousinRuleRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM user_ck_values WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete cousin rule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return cousinrule.ErrRuleNotFound
	}

	return nil
}

func (r *CousinRuleRepository) SetRuleTags(ctx context.Context, ruleID int64, tagIDs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete existing tags
	_, err = tx.ExecContext(ctx, `DELETE FROM user_ck_value_tags WHERE user_ck_value_id = $1`, ruleID)
	if err != nil {
		return fmt.Errorf("failed to delete existing tags: %w", err)
	}

	// Insert new tags
	if len(tagIDs) > 0 {
		valueStrings := make([]string, 0, len(tagIDs))
		valueArgs := make([]any, 0, len(tagIDs)*2)

		for i, tagID := range tagIDs {
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
			valueArgs = append(valueArgs, ruleID, tagID)
		}

		query := fmt.Sprintf(
			`INSERT INTO user_ck_value_tags (user_ck_value_id, tag_id) VALUES %s ON CONFLICT DO NOTHING`,
			strings.Join(valueStrings, ", "),
		)

		_, err = tx.ExecContext(ctx, query, valueArgs...)
		if err != nil {
			return fmt.Errorf("failed to insert tags: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *CousinRuleRepository) GetRuleTags(ctx context.Context, ruleID int64) ([]string, error) {
	query := `SELECT tag_id FROM user_ck_value_tags WHERE user_ck_value_id = $1`

	rows, err := r.db.QueryContext(ctx, query, ruleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule tags: %w", err)
	}
	defer rows.Close()

	var tagIDs []string
	for rows.Next() {
		var tagID string
		if err := rows.Scan(&tagID); err != nil {
			return nil, fmt.Errorf("failed to scan tag id: %w", err)
		}
		tagIDs = append(tagIDs, tagID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tag ids: %w", err)
	}

	return tagIDs, nil
}

func (r *CousinRuleRepository) ApplyRuleToTransactions(ctx context.Context, userID, cousinID int64, txType *string, changes cousinrule.Changes) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Build dynamic UPDATE query based on which changes are provided
	setClauses := []string{"manipulated = true", "updated_at = CURRENT_TIMESTAMP"}
	args := []any{}
	argIndex := 1

	if changes.Category != nil {
		setClauses = append(setClauses, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, *changes.Category)
		argIndex++
	}

	if changes.Description != nil {
		// Store original description before overwriting
		setClauses = append(setClauses, fmt.Sprintf("original_description = COALESCE(original_description, description)"))
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, *changes.Description)
		argIndex++
	}

	if changes.Notes != nil {
		setClauses = append(setClauses, fmt.Sprintf("notes = $%d", argIndex))
		args = append(args, *changes.Notes)
		argIndex++
	}

	if changes.Considered != nil {
		setClauses = append(setClauses, fmt.Sprintf("considered = $%d", argIndex))
		args = append(args, *changes.Considered)
		argIndex++
	}

	// Add WHERE clause parameters
	args = append(args, cousinID, userID)
	cousinArgIndex := argIndex
	userArgIndex := argIndex + 1

	// Build WHERE clause
	whereClause := fmt.Sprintf(`
		WHERE t.cousin = $%d
		  AND a.user_id = $%d
	`, cousinArgIndex, userArgIndex)

	if txType != nil {
		args = append(args, *txType)
		whereClause += fmt.Sprintf(" AND t.type = $%d", userArgIndex+1)
	}

	query := fmt.Sprintf(`
		UPDATE transactions t
		SET %s
		FROM accounts a
		WHERE t.account_id = a.id
		  AND t.cousin = $%d
		  AND a.user_id = $%d
	`, strings.Join(setClauses, ", "), cousinArgIndex, userArgIndex)

	if txType != nil {
		query += fmt.Sprintf(" AND t.type = $%d", userArgIndex+1)
	}

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to update transactions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Apply tags if specified
	if len(changes.Tags) > 0 {
		// Get all transaction IDs that were updated
		selectQuery := `
			SELECT t.id FROM transactions t
			JOIN accounts a ON t.account_id = a.id
			WHERE t.cousin = $1 AND a.user_id = $2
		`
		selectArgs := []any{cousinID, userID}
		if txType != nil {
			selectQuery += " AND t.type = $3"
			selectArgs = append(selectArgs, *txType)
		}

		rows, err := tx.QueryContext(ctx, selectQuery, selectArgs...)
		if err != nil {
			return 0, fmt.Errorf("failed to get transaction IDs: %w", err)
		}

		var txnIDs []string
		for rows.Next() {
			var txnID string
			if err := rows.Scan(&txnID); err != nil {
				rows.Close()
				return 0, fmt.Errorf("failed to scan transaction ID: %w", err)
			}
			txnIDs = append(txnIDs, txnID)
		}
		rows.Close()

		// For each transaction, add tags (not replace, to match Django behavior)
		for _, txnID := range txnIDs {
			for _, tagID := range changes.Tags {
				_, err = tx.ExecContext(ctx,
					`INSERT INTO transaction_tags (transaction_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
					txnID, tagID,
				)
				if err != nil {
					return 0, fmt.Errorf("failed to add tag to transaction: %w", err)
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return int(rowsAffected), nil
}

func (r *CousinRuleRepository) CheckDontAskAgain(ctx context.Context, userID, cousinID int64, txType string) (bool, error) {
	// Check for type-specific rule first, then fall back to type-agnostic rule
	query := `
		SELECT dont_ask_again FROM user_ck_values
		WHERE user_id = $1 AND cousin_id = $2 AND (type = $3 OR type IS NULL)
		ORDER BY type NULLS LAST
		LIMIT 1
	`

	var dontAskAgain bool
	err := r.db.QueryRowContext(ctx, query, userID, cousinID, txType).Scan(&dontAskAgain)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check dont_ask_again: %w", err)
	}

	return dontAskAgain, nil
}
