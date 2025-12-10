package openfinance

import (
	"context"
)

// ClientInterface defines the methods required from the Open Finance API client
type ClientInterface interface {
	GetAccounts(ctx context.Context, apiKey string) (*AccountResponse, error)
	GetAccountsWithStatus(ctx context.Context, apiKey string) (*AccountResponse, int, error) // Returns response and status code
	GetTransactions(ctx context.Context, apiKey string, startDate string) (*TransactionResponse, error)
	GetBills(ctx context.Context, apiKey string) (*BillResponse, error)
}
