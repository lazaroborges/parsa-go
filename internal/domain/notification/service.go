package notification

import (
	"context"
	"errors"
	"log"
)

// Service contains the business logic for notification operations
type Service struct {
	repo      Repository
	messenger Messenger
}

// NewService creates a new notification service
func NewService(repo Repository, messenger Messenger) *Service {
	return &Service{repo: repo, messenger: messenger}
}

// RegisterDevice registers a device token for the authenticated user.
// If the token already belongs to another user, it is reassigned.
// Creates default notification preferences if none exist.
func (s *Service) RegisterDevice(ctx context.Context, params CreateDeviceTokenParams) (*DeviceToken, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	token, err := s.repo.UpsertDeviceToken(ctx, params)
	if err != nil {
		return nil, err
	}

	// Ensure notification preferences exist for this user
	_, err = s.repo.GetPreferences(ctx, params.UserID)
	if err != nil {
		// Create default preferences
		_, err = s.repo.UpsertPreferences(ctx, params.UserID, UpdatePreferenceParams{})
		if err != nil {
			log.Printf("Warning: failed to create default notification preferences for user %d: %v", params.UserID, err)
		}
	}

	return token, nil
}

// GetPreferences returns the notification preferences for a user.
// Returns default (all-enabled) preferences if none have been created yet.
func (s *Service) GetPreferences(ctx context.Context, userID int64) (*NotificationPreference, error) {
	if userID <= 0 {
		return nil, errors.New("valid user ID is required")
	}

	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		// Return defaults if not found
		return &NotificationPreference{
			UserID:              userID,
			BudgetsEnabled:      true,
			GeneralEnabled:      true,
			AccountsEnabled:     true,
			TransactionsEnabled: true,
		}, nil
	}

	return prefs, nil
}

// UpdatePreferences updates notification preferences for a user
func (s *Service) UpdatePreferences(ctx context.Context, userID int64, params UpdatePreferenceParams) (*NotificationPreference, error) {
	if userID <= 0 {
		return nil, errors.New("valid user ID is required")
	}

	return s.repo.UpsertPreferences(ctx, userID, params)
}

// ListNotifications returns paginated notifications for a user
func (s *Service) ListNotifications(ctx context.Context, userID int64, page, perPage int) ([]*Notification, int, error) {
	if userID <= 0 {
		return nil, 0, errors.New("valid user ID is required")
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	return s.repo.ListByUserID(ctx, userID, page, perPage)
}

// MarkNotificationOpened marks a notification as opened by the authenticated user
func (s *Service) MarkNotificationOpened(ctx context.Context, notificationID string, userID int64) error {
	if notificationID == "" {
		return errors.New("notification ID is required")
	}
	if userID <= 0 {
		return errors.New("valid user ID is required")
	}

	return s.repo.MarkOpened(ctx, notificationID, userID)
}

// SendToUser sends a push notification to a specific user.
// Respects notification preferences and creates a notification record.
func (s *Service) SendToUser(ctx context.Context, userID int64, title, body, category string, data map[string]string) error {
	if !IsValidCategory(category) {
		return ErrInvalidCategory
	}

	// Check preferences
	prefs, err := s.GetPreferences(ctx, userID)
	if err != nil {
		return err
	}

	if !prefs.IsCategoryEnabled(category) {
		log.Printf("Notification skipped for user %d: category %q disabled", userID, category)
		return nil
	}

	// Get active device tokens
	tokens, err := s.repo.GetActiveTokensByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if len(tokens) == 0 {
		log.Printf("No active device tokens for user %d", userID)
		return nil
	}

	// Add route from category if not present
	if data == nil {
		data = make(map[string]string)
	}
	if _, ok := data["route"]; !ok {
		data["route"] = category
	}

	// Send to all active tokens via FCM (if messenger is configured)
	if s.messenger != nil {
		tokenStrings := make([]string, len(tokens))
		for i, t := range tokens {
			tokenStrings[i] = t.Token
		}

		if err := s.messenger.SendMulticast(ctx, tokenStrings, title, body, data); err != nil {
			log.Printf("Error sending notification to user %d: %v", userID, err)
		}
	}

	// Store notification record
	_, err = s.repo.CreateNotification(ctx, CreateNotificationParams{
		UserID:   userID,
		Title:    title,
		Message:  body,
		Category: category,
		Data:     data,
	})
	if err != nil {
		log.Printf("Error storing notification for user %d: %v", userID, err)
	}

	return nil
}

// SendToToken sends a push notification to a specific device token
func (s *Service) SendToToken(ctx context.Context, token, title, body, category string, data map[string]string) error {
	if token == "" {
		return ErrInvalidToken
	}
	if !IsValidCategory(category) {
		return ErrInvalidCategory
	}

	if data == nil {
		data = make(map[string]string)
	}
	if _, ok := data["route"]; !ok {
		data["route"] = category
	}

	if s.messenger == nil {
		return nil
	}

	return s.messenger.Send(ctx, token, title, body, data)
}

// SendToAll sends a push notification to all users with active device tokens.
// This is intended for staff/admin use only (enforced at the handler level).
func (s *Service) SendToAll(ctx context.Context, title, body, category string, data map[string]string) error {
	if !IsValidCategory(category) {
		return ErrInvalidCategory
	}

	if data == nil {
		data = make(map[string]string)
	}
	if _, ok := data["route"]; !ok {
		data["route"] = category
	}

	allTokens, err := s.repo.GetAllActiveTokens(ctx)
	if err != nil {
		return err
	}

	if len(allTokens) == 0 {
		log.Println("SendToAll: no active device tokens found")
		return nil
	}

	if s.messenger == nil {
		return nil
	}

	tokenStrings := make([]string, len(allTokens))
	for i, t := range allTokens {
		tokenStrings[i] = t.Token
	}

	return s.messenger.SendMulticast(ctx, tokenStrings, title, body, data)
}
