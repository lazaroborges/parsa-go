package openfinance

import (
	"context"
)

// ClientInterface defines the methods required from the Open Finance API client
type ClientInterface interface {
	GetAccounts(ctx context.Context, apiKey string) (*AccountResponse, error)
	GetTransactions(ctx context.Context, apiKey string) (*TransactionResponse, error)
}
