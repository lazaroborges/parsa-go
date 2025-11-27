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
	}
	accountSubtypes = map[string]struct{}{
		"CHECKING_ACCOUNT": {},
		"SAVINGS_ACCOUNT":  {},
		"CREDIT_CARD":      {},
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
	ID                string    `json:"id"` // Provider's account id (UUID string)
	UserID            int64     `json:"userId"`
	ItemID            string    `json:"itemId"` // FK to items table
	Name              string    `json:"name"`
	AccountType       string    `json:"accountType"`
	Subtype           string    `json:"subtype"`
	Currency          string    `json:"currency"`
	Balance           float64   `json:"balance"`
	BankID            int64     `json:"bankId"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
	ProviderUpdatedAt time.Time `json:"providerUpdatedAt"`
	ProviderCreatedAt time.Time `json:"providerCreatedAt"`
}

type CreateAccountParams struct {
	ID          string // Provider's account id
	UserID      int64
	ItemID      string
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
	ID                string // Provider's account id (used as PK)
	UserID            int64
	ItemID            string
	Name              string
	AccountType       string
	Subtype           *string
	Currency          string
	Balance           float64
	BankID            *int64
	ProviderUpdatedAt *time.Time
	ProviderCreatedAt *time.Time
}
