package notification

import (
	"errors"
	"time"
)

// Notification categories
const (
	CategoryAccounts     = "accounts"
	CategoryBudgets      = "budgets"
	CategoryGeneral      = "general"
	CategoryTransactions = "transactions"
)

var validCategories = map[string]struct{}{
	CategoryAccounts:     {},
	CategoryBudgets:      {},
	CategoryGeneral:      {},
	CategoryTransactions: {},
}

var validDeviceTypes = map[string]struct{}{
	"ios":     {},
	"android": {},
}

// Domain errors
var (
	ErrDeviceTokenNotFound    = errors.New("device token not found")
	ErrNotificationNotFound   = errors.New("notification not found")
	ErrPreferencesNotFound    = errors.New("notification preferences not found")
	ErrInvalidCategory      = errors.New("invalid notification category")
	ErrInvalidDeviceType    = errors.New("device type must be 'ios' or 'android'")
	ErrInvalidToken         = errors.New("device token is required")
	ErrForbidden            = errors.New("access forbidden")
)

// DeviceToken represents a registered FCM device token
type DeviceToken struct {
	ID         string    `json:"id"`
	UserID     int64     `json:"userId"`
	Token      string    `json:"token"`
	DeviceType string    `json:"deviceType"`
	IsActive   bool      `json:"isActive"`
	CreatedAt  time.Time `json:"createdAt"`
	LastUsed   time.Time `json:"lastUsed"`
}

// NotificationPreference stores per-category notification toggles for a user
type NotificationPreference struct {
	ID                  string    `json:"id"`
	UserID              int64     `json:"-"`
	BudgetsEnabled      bool      `json:"budgets_enabled"`
	GeneralEnabled      bool      `json:"general_enabled"`
	AccountsEnabled     bool      `json:"accounts_enabled"`
	TransactionsEnabled bool      `json:"transactions_enabled"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

// Notification represents a stored notification record
type Notification struct {
	ID        string            `json:"id"`
	UserID    int64             `json:"-"`
	Title     string            `json:"title"`
	Message   string            `json:"message"`
	Category  string            `json:"category"`
	Data      map[string]string `json:"data"`
	OpenedAt  *time.Time        `json:"opened_at"`
	CreatedAt time.Time         `json:"created_at"`
}

// CreateDeviceTokenParams contains parameters for registering a device
type CreateDeviceTokenParams struct {
	UserID     int64
	Token      string
	DeviceType string
}

func (p CreateDeviceTokenParams) Validate() error {
	if p.UserID <= 0 {
		return errors.New("valid user ID is required")
	}
	if p.Token == "" {
		return ErrInvalidToken
	}
	if !IsValidDeviceType(p.DeviceType) {
		return ErrInvalidDeviceType
	}
	return nil
}

// UpdatePreferenceParams contains fields for updating notification preferences
type UpdatePreferenceParams struct {
	BudgetsEnabled      *bool
	GeneralEnabled      *bool
	AccountsEnabled     *bool
	TransactionsEnabled *bool
}

// CreateNotificationParams contains parameters for storing a notification
type CreateNotificationParams struct {
	UserID   int64
	Title    string
	Message  string
	Category string
	Data     map[string]string
}

func (p CreateNotificationParams) Validate() error {
	if p.UserID <= 0 {
		return errors.New("valid user ID is required")
	}
	if p.Title == "" {
		return errors.New("notification title is required")
	}
	if p.Message == "" {
		return errors.New("notification message is required")
	}
	if !IsValidCategory(p.Category) {
		return ErrInvalidCategory
	}
	return nil
}

func IsValidCategory(c string) bool {
	_, ok := validCategories[c]
	return ok
}

func IsValidDeviceType(dt string) bool {
	_, ok := validDeviceTypes[dt]
	return ok
}

// IsCategoryEnabled checks if a specific category is enabled in preferences
func (p *NotificationPreference) IsCategoryEnabled(category string) bool {
	switch category {
	case CategoryAccounts:
		return p.AccountsEnabled
	case CategoryBudgets:
		return p.BudgetsEnabled
	case CategoryGeneral:
		return p.GeneralEnabled
	case CategoryTransactions:
		return p.TransactionsEnabled
	default:
		return false
	}
}
