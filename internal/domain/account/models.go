package account

import (
	"errors"
	"time"
)

var (
	// Allowed account types and subtypes for validation (from Open Finance API)
	accountTypes = map[string]struct{}{
		"BANK":       {},
		"CREDIT":     {},
		"INVESTMENT": {},
	}
	accountSubtypes = map[string]struct{}{
		"CHECKING_ACCOUNT": {},
		"SAVINGS_ACCOUNT":  {},
		"CREDIT_CARD":      {},
	}
	// Common ISO 4217 currency codes
	validCurrencies = map[string]struct{}{
		"BRL": {}, "USD": {}, "EUR": {}, "GBP": {}, "JPY": {},
		"CHF": {}, "CAD": {}, "AUD": {}, "NZD": {}, "CNY": {},
		"INR": {}, "MXN": {}, "ZAR": {}, "SEK": {}, "NOK": {},
		"DKK": {}, "PLN": {}, "TRY": {}, "RUB": {}, "KRW": {},
		"SGD": {}, "HKD": {}, "ARS": {}, "CLP": {}, "COP": {},
	}
)

// Domain errors
var (
	ErrInvalidAccountType    = errors.New("invalid account type")
	ErrInvalidAccountSubtype = errors.New("invalid account subtype")
	ErrAccountNotFound       = errors.New("account not found")
	ErrForbidden             = errors.New("access forbidden")
	ErrInvalidInput          = errors.New("invalid input")
	ErrInvalidCurrency       = errors.New("valid ISO 4217 currency is required")
)

// Account represents a financial account domain entity
type Account struct {
	ID                   string    `json:"id"`
	UserID               int64     `json:"userId"`
	ItemID               string    `json:"itemId"`
	Name                 string    `json:"name"`
	AccountType          string    `json:"accountType"`
	Subtype              string    `json:"subtype"`
	Currency             string    `json:"currency"`
	Balance              float64   `json:"balance"`
	BankID               int64     `json:"bankId"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
	ProviderUpdatedAt    time.Time `json:"providerUpdatedAt"`
	ProviderCreatedAt    time.Time `json:"providerCreatedAt"`
	InitialBalance       float64   `json:"initialBalance"`       // default to 0.00
	IsOpenFinanceAccount bool      `json:"isOpenFinanceAccount"` // default to true
	ClosedAt             time.Time `json:"closedAt"`             // default to NULL
	UIOrder              int       `json:"order"`                // default to 90
	Description          string    `json:"description"`          // default to NULL
	Removed              bool      `json:"removed"`              // default to false
	HiddenByUser         bool      `json:"hiddenByUser"`         // default to false
}

// AccountWithBank represents an account with its associated bank data (for API responses)
type AccountWithBank struct {
	Account
	BankName         string `json:"bankName"`
	BankUIName       string `json:"bankUIName"`
	BankConnector    string `json:"bankConnector"`
	BankPrimaryColor string `json:"bankPrimaryColor"`
}

// CreateParams contains parameters for creating a new account
type CreateParams struct {
	ID             string
	UserID         int64
	ItemID         string
	Name           string
	AccountType    string
	Currency       string
	Balance        float64
	BankID         int64
	InitialBalance float64
	Description    string
}

// Validate validates the create parameters
func (p CreateParams) Validate() error {
	if p.ID == "" {
		return errors.New("account ID is required")
	}
	if p.UserID <= 0 {
		return errors.New("valid user ID is required")
	}
	if p.Name == "" {
		return errors.New("account name is required")
	}
	if p.AccountType == "" {
		return errors.New("account type is required")
	}
	if !IsValidAccountType(p.AccountType) {
		return ErrInvalidAccountType
	}
	if p.Currency == "" {
		return errors.New("currency is required")
	}
	if !IsValidCurrency(p.Currency) {
		return ErrInvalidCurrency
	}
	return nil
}

// UpdateParams contains parameters for updating an account
type UpdateParams struct {
	Name        *string
	AccountType *string
	Balance     *float64
	Order       *int  // UIOrder field
	HiddenByUser *bool
}

// UpsertParams contains parameters for upserting an account
type UpsertParams struct {
	ID                   string
	UserID               int64
	ItemID               string
	Name                 string
	AccountType          string
	Subtype              *string
	Currency             string
	Balance              float64
	BankID               *int64
	ProviderUpdatedAt    *time.Time
	ProviderCreatedAt    *time.Time
	InitialBalance       *float64
	IsOpenFinanceAccount *bool
	ClosedAt             *time.Time
	UIOrder              *int
	Description          *string
	Removed              *bool
	HiddenByUser         *bool
}

// Validate validates the upsert parameters
func (p UpsertParams) Validate() error {
	if p.ID == "" {
		return errors.New("account ID is required for upsert")
	}
	if p.UserID <= 0 {
		return errors.New("valid user ID is required for upsert")
	}
	if p.Name == "" {
		return errors.New("account name is required")
	}
	if p.AccountType == "" {
		return errors.New("account type is required")
	}
	if !IsValidAccountType(p.AccountType) {
		return ErrInvalidAccountType
	}
	if p.Subtype != nil && !IsValidAccountSubtype(*p.Subtype) {
		return ErrInvalidAccountSubtype
	}
	if p.Currency == "" || !IsValidCurrency(p.Currency) {
		return ErrInvalidCurrency
	}
	return nil
}

// IsValidAccountType checks if the provided account type is valid.
func IsValidAccountType(t string) bool {
	_, ok := accountTypes[t]
	return ok
}

// IsValidAccountSubtype checks if the provided subtype is valid.
func IsValidAccountSubtype(s string) bool {
	_, ok := accountSubtypes[s]
	return ok
}

// IsValidCurrency checks if the provided currency is a valid ISO 4217 code.
func IsValidCurrency(c string) bool {
	if len(c) != 3 {
		return false
	}
	_, ok := validCurrencies[c]
	return ok
}
