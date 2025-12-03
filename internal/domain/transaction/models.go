package transaction

import (
	"time"
)

type Transaction struct {
	ID                string    `json:"id"` // Provider's transaction id (UUID string)
	AccountID         string    `json:"accountId"`
	Amount            float64   `json:"amount"`
	Description       string    `json:"description"`
	Category          *string   `json:"category,omitempty"`
	TransactionDate   time.Time `json:"transactionDate"`
	Type              string    `json:"type"`   // "DEBIT" or "CREDIT"
	Status            string    `json:"status"` // "PENDING" or "POSTED"
	ProviderCreatedAt time.Time `json:"providerCreatedAt,omitempty"`
	ProviderUpdatedAt time.Time `json:"providerUpdatedAt,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
	Considered        bool      `json:"considered"`
	IsOpenFinance     bool      `json:"isOpenFinance"`
	Tags              []string  `json:"tags"`
	Manipulated       bool      `json:"manipulated"`
}

type CreateTransactionParams struct {
	ID              string // Provider's transaction id
	AccountID       string
	Amount          float64
	Description     string
	Category        *string
	TransactionDate time.Time
	Type            string
	Status          string
}

type UpdateTransactionParams struct {
	Amount          *float64
	Description     *string
	Category        *string
	TransactionDate *time.Time
	Type            *string
	Status          *string
}

// UpsertTransactionParams is used for syncing transactions from the provider
type UpsertTransactionParams struct {
	ID                string // Provider's transaction id (used as PK)
	AccountID         string
	Amount            float64
	Description       string
	Category          *string
	TransactionDate   time.Time
	Type              string
	Status            string
	ProviderCreatedAt *time.Time
	ProviderUpdatedAt *time.Time
}
