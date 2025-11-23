package models

import (
	"time"
)

type Transaction struct {
	ID              string    `json:"id"`
	ProviderID      string    `json:"providerId"`
	AccountID       string    `json:"accountId"`
	Amount          float64   `json:"amount"`
	Description     string    `json:"description"`
	Category        *string   `json:"category,omitempty"`
	TransactionDate time.Time `json:"transactionDate"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type CreateTransactionParams struct {
	AccountID       string
	Amount          float64
	Description     string
	Category        *string
	TransactionDate time.Time
}

type UpdateTransactionParams struct {
	Amount          *float64
	Description     *string
	Category        *string
	TransactionDate *time.Time
}
