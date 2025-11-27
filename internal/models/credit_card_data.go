package models

import "time"

type CreditCardData struct {
	ID                int64     `json:"id"`
	PurchaseDate      time.Time `json:"purchaseDate"`
	InstallmentNumber int       `json:"installmentNumber"`
	TotalInstallments int       `json:"totalInstallments"`
}

type CreateCreditCardDataParams struct {
	PurchaseDate      time.Time
	InstallmentNumber int
	TotalInstallments int
}

type UpdateCreditCardDataParams struct {
	PurchaseDate      *time.Time
	InstallmentNumber *int
	TotalInstallments *int
}
