package models

import "context"

// ItemRepository defines data access for Items
type ItemRepository interface {
	FindOrCreate(ctx context.Context, id string, userID int64) (*Item, error)
	ListByUserID(ctx context.Context, userID int64) ([]*Item, error)
	Delete(ctx context.Context, id string) error
}

// BankRepository defines data access for Banks
type BankRepository interface {
	FindOrCreateByName(ctx context.Context, name string) (*Bank, error)
}

// CreditCardDataRepository defines data access for Credit Card Data
type CreditCardDataRepository interface {
	Upsert(ctx context.Context, transactionID string, params CreateCreditCardDataParams) (*CreditCardData, error)
}
