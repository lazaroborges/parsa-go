package bill

import (
	"errors"
	"time"
)

// Domain errors
var (
	ErrBillNotFound = errors.New("bill not found")
	ErrForbidden    = errors.New("access forbidden")
	ErrInvalidInput = errors.New("invalid input")
)

// Bill represents a credit card bill (fatura) domain entity
// This corresponds to the Pierre Finance get-bills endpoint which returns
// past due credit card bills (faturas de cartão de crédito vencidas)
type Bill struct {
	ID                string    `json:"id"` // Provider's bill ID
	AccountID         string    `json:"accountId"`
	DueDate           time.Time `json:"dueDate"`     // Vencimento
	TotalAmount       float64   `json:"totalAmount"` // Valor total da fatura
	ProviderCreatedAt time.Time `json:"providerCreatedAt,omitempty"`
	ProviderUpdatedAt time.Time `json:"providerUpdatedAt,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
	IsOpenFinance     bool      `json:"isOpenFinance"`
}

// BillWithAccount represents a bill with its associated account data (for API responses)
type BillWithAccount struct {
	Bill
	AccountName    string `json:"accountName"`
	AccountType    string `json:"accountType"`
	AccountSubtype string `json:"accountSubtype"`
	BankName       string `json:"bankName"`
}

// CreateParams contains parameters for creating a new bill
type CreateParams struct {
	ID          string
	AccountID   string
	DueDate     time.Time
	TotalAmount float64
}

// Validate validates the create parameters
func (p CreateParams) Validate() error {
	if p.ID == "" {
		return errors.New("bill ID is required")
	}
	if p.AccountID == "" {
		return errors.New("account ID is required")
	}
	if p.DueDate.IsZero() {
		return errors.New("due date is required")
	}
	return nil
}

// UpsertParams contains parameters for upserting a bill
type UpsertParams struct {
	ID                string
	AccountID         string
	DueDate           time.Time
	TotalAmount       float64
	ProviderCreatedAt *time.Time
	ProviderUpdatedAt *time.Time
}

// Validate validates the upsert parameters
func (p UpsertParams) Validate() error {
	if p.ID == "" {
		return errors.New("bill ID is required for upsert")
	}
	if p.AccountID == "" {
		return errors.New("account ID is required for upsert")
	}
	if p.DueDate.IsZero() {
		return errors.New("due date is required")
	}
	return nil
}

// UpdateParams contains parameters for updating a bill
type UpdateParams struct {
	DueDate     *time.Time
	TotalAmount *float64
}
