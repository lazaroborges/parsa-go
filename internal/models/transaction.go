package models

import (
	"time"
)

type Transaction struct {
	ID              string    `json:"id"`
	AccountID       string    `json:"account_id"`
	Amount          float64   `json:"amount"`
	Description     string    `json:"description"`
	Category        *string   `json:"category,omitempty"`
	TransactionDate time.Time `json:"transaction_date"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
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
