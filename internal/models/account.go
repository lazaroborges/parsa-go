package models

import (
	"time"
)

type Account struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	Name        string    `json:"name"`
	AccountType string    `json:"accountType"`
	Currency    string    `json:"currency"`
	Balance     float64   `json:"balance"`
	BankID      string    `json:"bankId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateAccountParams struct {
	UserID      string
	Name        string
	AccountType string
	Currency    string
	Balance     float64
	BankID      string
}

type UpdateAccountParams struct {
	Name        *string
	AccountType *string
	Balance     *float64
}
