package notification

import "context"

// Repository defines the interface for notification data access.
// Defined in the domain layer, implemented in the infrastructure layer.
type Repository interface {
	// Device tokens
	UpsertDeviceToken(ctx context.Context, params CreateDeviceTokenParams) (*DeviceToken, error)
	GetActiveTokensByUserID(ctx context.Context, userID int64) ([]*DeviceToken, error)
	GetAllActiveTokens(ctx context.Context) ([]*DeviceToken, error)
	DeactivateToken(ctx context.Context, token string) error
	ReassignToken(ctx context.Context, token string, newUserID int64) error

	// Notification preferences
	GetPreferences(ctx context.Context, userID int64) (*NotificationPreference, error)
	UpsertPreferences(ctx context.Context, userID int64, params UpdatePreferenceParams) (*NotificationPreference, error)

	// Notifications
	CreateNotification(ctx context.Context, params CreateNotificationParams) (*Notification, error)
	ListByUserID(ctx context.Context, userID int64, page, perPage int) ([]*Notification, int, error)
	MarkOpened(ctx context.Context, notificationID string, userID int64) error
}
