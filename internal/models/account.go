package models

import (
	"time"
)

type Account struct {
	ID          string  `json:"id"`
	UserID      string  `json:"user_id"`
	Name        string  `json:"name"`
	AccountType string  `json:"account_type"`
	Currency    string  `json:"currency"`
	Balance     float64 `json:"balance"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateAccountParams struct {
	UserID      string
	Name        string
	AccountType string
	Currency    string
	Balance     float64
}

type UpdateAccountParams struct {
	Name        *string
	AccountType *string
	Balance     *float64
}
