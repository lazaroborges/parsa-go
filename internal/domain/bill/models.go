package bill

import (
	"errors"
	"time"
)

// Bill status values
var billStatuses = map[string]struct{}{
	"OPEN":      {},
	"PAID":      {},
	"OVERDUE":   {},
	"CANCELLED": {},
}

// Domain errors
var (
	ErrBillNotFound     = errors.New("bill not found")
	ErrForbidden        = errors.New("access forbidden")
	ErrInvalidInput     = errors.New("invalid input")
	ErrInvalidStatus    = errors.New("invalid bill status")
)

// Bill represents a bill/boleto domain entity
type Bill struct {
	ID                   string     `json:"id"` // Provider's bill ID (UUID string)
	AccountID            string     `json:"accountId"`
	Amount               float64    `json:"amount"`
	DueDate              time.Time  `json:"dueDate"`
	Status               string     `json:"status"` // OPEN, PAID, OVERDUE, CANCELLED
	Description          string     `json:"description"`
	BillerName           string     `json:"billerName"`
	Category             *string    `json:"category,omitempty"`
	Barcode              *string    `json:"barcode,omitempty"`       // Brazilian boleto barcode
	DigitableLine        *string    `json:"digitableLine,omitempty"` // Boleto digitable line
	PaymentDate          *time.Time `json:"paymentDate,omitempty"`
	RelatedTransactionID *string    `json:"relatedTransactionId,omitempty"`
	ProviderCreatedAt    time.Time  `json:"providerCreatedAt,omitempty"`
	ProviderUpdatedAt    time.Time  `json:"providerUpdatedAt,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
	IsOpenFinance        bool       `json:"isOpenFinance"`
}

// BillWithAccount represents a bill with its associated account data (for API responses)
type BillWithAccount struct {
	Bill
	AccountName    string `json:"accountName"`
	AccountType    string `json:"accountType"`
	AccountSubtype string `json:"accountSubtype"`
}

// CreateParams contains parameters for creating a new bill
type CreateParams struct {
	ID                   string
	AccountID            string
	Amount               float64
	DueDate              time.Time
	Status               string
	Description          string
	BillerName           string
	Category             *string
	Barcode              *string
	DigitableLine        *string
	PaymentDate          *time.Time
	RelatedTransactionID *string
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
	if p.Status == "" {
		return errors.New("status is required")
	}
	if !IsValidStatus(p.Status) {
		return ErrInvalidStatus
	}
	return nil
}

// UpsertParams contains parameters for upserting a bill
type UpsertParams struct {
	ID                   string
	AccountID            string
	Amount               float64
	DueDate              time.Time
	Status               string
	Description          string
	BillerName           string
	Category             *string
	Barcode              *string
	DigitableLine        *string
	PaymentDate          *time.Time
	RelatedTransactionID *string
	ProviderCreatedAt    *time.Time
	ProviderUpdatedAt    *time.Time
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
	if p.Status == "" {
		return errors.New("status is required")
	}
	if !IsValidStatus(p.Status) {
		return ErrInvalidStatus
	}
	return nil
}

// UpdateParams contains parameters for updating a bill
type UpdateParams struct {
	Amount               *float64
	DueDate              *time.Time
	Status               *string
	Description          *string
	BillerName           *string
	Category             *string
	PaymentDate          *time.Time
	RelatedTransactionID *string
}

// IsValidStatus checks if the provided status is valid
func IsValidStatus(s string) bool {
	_, ok := billStatuses[s]
	return ok
}
