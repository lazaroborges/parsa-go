package models

import (
	"time"
)

var (
	// Allowed account types and subtypes for validation (from Open Finance API)
	accountTypes = map[string]struct{}{
		"BANK":       {},
		"CREDIT":     {},
		"INVESTMENT": {},
		"LOAN":       {},
	}
	accountSubtypes = map[string]struct{}{
		"CHECKING_ACCOUNT": {},
		"SAVINGS_ACCOUNT":  {},
		"CREDIT_CARD":      {},
		"PAYMENT_ACCOUNT":  {},
	}
)

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

type Account struct {
	ID                int64     `json:"id"`
	UserID            int64     `json:"userId"`
	Name              string    `json:"name"`
	AccountType       string    `json:"accountType"`
	Subtype           string    `json:"subtype"`
	Currency          string    `json:"currency"`
	Balance           float64   `json:"balance"`
	BankID            int64     `json:"bankId"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
	ProviderID        string    `json:"providerId"`
	ProviderUpdatedAt time.Time `json:"providerUpdatedAt"`
	ProviderCreatedAt time.Time `json:"providerCreatedAt"`
}

type CreateAccountParams struct {
	UserID      int64
	Name        string
	AccountType string
	Currency    string
	Balance     float64
	BankID      int64
}

type UpdateAccountParams struct {
	Name        *string
	AccountType *string
	Balance     *float64
}

type UpsertAccountParams struct {
	UserID            int64
	Name              string
	AccountType       string
	Subtype           *string
	Currency          string
	Balance           float64
	BankID            *int64
	ProviderID        string
	ProviderUpdatedAt *time.Time
	ProviderCreatedAt *time.Time
}
