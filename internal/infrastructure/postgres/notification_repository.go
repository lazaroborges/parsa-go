package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"parsa/internal/domain/notification"
)

type NotificationRepository struct {
	db *DB
}

func NewNotificationRepository(db *DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// UpsertDeviceToken registers or updates a device token for a user.
// If the token exists for a different user, it is reassigned.
func (r *NotificationRepository) UpsertDeviceToken(ctx context.Context, params notification.CreateDeviceTokenParams) (*notification.DeviceToken, error) {
	// Reassign if the token belongs to another user
	_, err := r.db.ExecContext(ctx,
		`UPDATE fcm_device_tokens SET user_id = $1, is_active = true, last_used = NOW() WHERE token = $2 AND user_id != $1`,
		params.UserID, params.Token,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reassign device token: %w", err)
	}

	query := `
		INSERT INTO fcm_device_tokens (user_id, token, device_type)
		VALUES ($1, $2, $3)
		ON CONFLICT (token) DO UPDATE
			SET user_id = EXCLUDED.user_id,
			    device_type = EXCLUDED.device_type,
			    is_active = true,
			    last_used = NOW()
		RETURNING id, user_id, token, device_type, is_active, created_at, last_used
	`

	var dt notification.DeviceToken
	err = r.db.QueryRowContext(ctx, query, params.UserID, params.Token, params.DeviceType).Scan(
		&dt.ID, &dt.UserID, &dt.Token, &dt.DeviceType, &dt.IsActive, &dt.CreatedAt, &dt.LastUsed,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert device token: %w", err)
	}

	return &dt, nil
}

func (r *NotificationRepository) GetActiveTokensByUserID(ctx context.Context, userID int64) ([]*notification.DeviceToken, error) {
	query := `
		SELECT id, user_id, token, device_type, is_active, created_at, last_used
		FROM fcm_device_tokens
		WHERE user_id = $1 AND is_active = true
		ORDER BY last_used DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*notification.DeviceToken
	for rows.Next() {
		var dt notification.DeviceToken
		if err := rows.Scan(&dt.ID, &dt.UserID, &dt.Token, &dt.DeviceType, &dt.IsActive, &dt.CreatedAt, &dt.LastUsed); err != nil {
			return nil, fmt.Errorf("failed to scan device token: %w", err)
		}
		tokens = append(tokens, &dt)
	}

	return tokens, rows.Err()
}

func (r *NotificationRepository) GetAllActiveTokens(ctx context.Context) ([]*notification.DeviceToken, error) {
	query := `
		SELECT id, user_id, token, device_type, is_active, created_at, last_used
		FROM fcm_device_tokens
		WHERE is_active = true
		ORDER BY user_id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all active device tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*notification.DeviceToken
	for rows.Next() {
		var dt notification.DeviceToken
		if err := rows.Scan(&dt.ID, &dt.UserID, &dt.Token, &dt.DeviceType, &dt.IsActive, &dt.CreatedAt, &dt.LastUsed); err != nil {
			return nil, fmt.Errorf("failed to scan device token: %w", err)
		}
		tokens = append(tokens, &dt)
	}

	return tokens, rows.Err()
}

func (r *NotificationRepository) DeactivateToken(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE fcm_device_tokens SET is_active = false WHERE token = $1`,
		token,
	)
	if err != nil {
		return fmt.Errorf("failed to deactivate token: %w", err)
	}
	return nil
}

func (r *NotificationRepository) ReassignToken(ctx context.Context, token string, newUserID int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE fcm_device_tokens SET user_id = $1, is_active = true, last_used = NOW() WHERE token = $2`,
		newUserID, token,
	)
	if err != nil {
		return fmt.Errorf("failed to reassign token: %w", err)
	}
	return nil
}

// GetPreferences returns notification preferences for a user.
// Returns notification.ErrPreferencesNotFound if no preferences exist.
func (r *NotificationRepository) GetPreferences(ctx context.Context, userID int64) (*notification.NotificationPreference, error) {
	query := `
		SELECT id, user_id, budgets_enabled, general_enabled, accounts_enabled, transactions_enabled, updated_at
		FROM fcm_notification_preferences
		WHERE user_id = $1
	`

	var pref notification.NotificationPreference
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&pref.ID, &pref.UserID, &pref.BudgetsEnabled, &pref.GeneralEnabled,
		&pref.AccountsEnabled, &pref.TransactionsEnabled, &pref.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, notification.ErrPreferencesNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get notification preferences: %w", err)
	}

	return &pref, nil
}

func (r *NotificationRepository) UpsertPreferences(ctx context.Context, userID int64, params notification.UpdatePreferenceParams) (*notification.NotificationPreference, error) {
	var budgets any
	var general any
	var accounts any
	var transactions any

	if params.BudgetsEnabled != nil {
		budgets = *params.BudgetsEnabled
	}
	if params.GeneralEnabled != nil {
		general = *params.GeneralEnabled
	}
	if params.AccountsEnabled != nil {
		accounts = *params.AccountsEnabled
	}
	if params.TransactionsEnabled != nil {
		transactions = *params.TransactionsEnabled
	}

	query := `
		INSERT INTO fcm_notification_preferences (user_id, budgets_enabled, general_enabled, accounts_enabled, transactions_enabled)
		VALUES (
			$1,
			COALESCE($2::boolean, true),
			COALESCE($3::boolean, true),
			COALESCE($4::boolean, true),
			COALESCE($5::boolean, true)
		)
		ON CONFLICT (user_id) DO UPDATE
			SET budgets_enabled = COALESCE($2::boolean, fcm_notification_preferences.budgets_enabled),
			    general_enabled = COALESCE($3::boolean, fcm_notification_preferences.general_enabled),
			    accounts_enabled = COALESCE($4::boolean, fcm_notification_preferences.accounts_enabled),
			    transactions_enabled = COALESCE($5::boolean, fcm_notification_preferences.transactions_enabled),
			    updated_at = NOW()
		RETURNING id, user_id, budgets_enabled, general_enabled, accounts_enabled, transactions_enabled, updated_at
	`

	var pref notification.NotificationPreference
	err := r.db.QueryRowContext(ctx, query, userID, budgets, general, accounts, transactions).Scan(
		&pref.ID, &pref.UserID, &pref.BudgetsEnabled, &pref.GeneralEnabled,
		&pref.AccountsEnabled, &pref.TransactionsEnabled, &pref.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert notification preferences: %w", err)
	}

	return &pref, nil
}

func (r *NotificationRepository) CreateNotification(ctx context.Context, params notification.CreateNotificationParams) (*notification.Notification, error) {
	dataJSON, err := json.Marshal(params.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal notification data: %w", err)
	}

	query := `
		INSERT INTO fcm_notifications (user_id, title, message, category, data)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, title, message, category, data, opened_at, created_at
	`

	var n notification.Notification
	var dataBytes []byte
	var openedAt sql.NullTime

	err = r.db.QueryRowContext(ctx, query, params.UserID, params.Title, params.Message, params.Category, dataJSON).Scan(
		&n.ID, &n.UserID, &n.Title, &n.Message, &n.Category, &dataBytes, &openedAt, &n.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	if openedAt.Valid {
		n.OpenedAt = &openedAt.Time
	}

	if len(dataBytes) > 0 {
		if err := json.Unmarshal(dataBytes, &n.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal notification data: %w", err)
		}
	}

	return &n, nil
}

func (r *NotificationRepository) ListByUserID(ctx context.Context, userID int64, page, perPage int) ([]*notification.Notification, int, error) {
	// Get total count
	var total int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM fcm_notifications WHERE user_id = $1`,
		userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	offset := (page - 1) * perPage
	query := `
		SELECT id, user_id, title, message, category, data, opened_at, created_at
		FROM fcm_notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*notification.Notification
	for rows.Next() {
		var n notification.Notification
		var dataBytes []byte
		var openedAt sql.NullTime

		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Message, &n.Category, &dataBytes, &openedAt, &n.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification: %w", err)
		}

		if openedAt.Valid {
			n.OpenedAt = &openedAt.Time
		}

		if len(dataBytes) > 0 {
			if err := json.Unmarshal(dataBytes, &n.Data); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal notification data: %w", err)
			}
		}

		notifications = append(notifications, &n)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating notifications: %w", err)
	}

	return notifications, total, nil
}

func (r *NotificationRepository) MarkOpened(ctx context.Context, notificationID string, userID int64) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE fcm_notifications
		SET opened_at = COALESCE(opened_at, $1)
		WHERE id = $2 AND user_id = $3`,
		time.Now(), notificationID, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to mark notification as opened: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return notification.ErrNotificationNotFound
	}

	return nil
}
