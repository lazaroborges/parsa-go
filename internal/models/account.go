package models

import (
	"time"
)

type Account struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"userId"`
	Name        string    `json:"name"`
	AccountType string    `json:"accountType"`
	Currency    string    `json:"currency"`
	Balance     float64   `json:"balance"`
	BankID      int64     `json:"bankId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
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
