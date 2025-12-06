package openfinance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	baseURL          = "https://www.pierre.finance/tools/api"
	defaultTimeout   = 180 * time.Second // Increased for large transaction fetches
	accountsPath     = "/get-accounts"
	transactionsPath = "/get-transactions"
	billsPath        = "/get-bills"
)

// Client handles communication with the Open Finance API
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// Ensure Client implements ClientInterface
var _ ClientInterface = (*Client)(nil)

// NewClient creates a new Open Finance API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		baseURL: baseURL,
	}
}

// AccountResponse represents the API response for account data
type AccountResponse struct {
	Success   bool      `json:"success"`
	Data      []Account `json:"data"`
	Count     int       `json:"count"`
	Timestamp string    `json:"timestamp"`
}

// Account represents an account from the Open Finance API
type Account struct {
	AccountID            string      `json:"id"`
	ItemID               string      `json:"itemId"` // Identifies the bank connection/relationship
	ProviderCode         string      `json:"providerCode"`
	AccountName          string      `json:"name"`
	AccountType          string      `json:"type"`
	AccountSubtype       string      `json:"subtype"`
	AccountCurrencyCode  string      `json:"currencyCode"`
	AccountMarketingName string      `json:"marketingName"`
	BalanceString        string      `json:"balance"` // API returns balance as string
	CreatedAt            string      `json:"createdAt"`
	UpdatedAt            string      `json:"updatedAt"`
	BankData             *BankData   `json:"bankData,omitempty"`
	CreditData           *CreditData `json:"creditData,omitempty"`
}

// GetBalance returns the balance as a float64
func (a *Account) GetBalance() (float64, error) {
	if a.BalanceString == "" {
		return 0, nil
	}
	balance, err := strconv.ParseFloat(a.BalanceString, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse balance '%s': %w", a.BalanceString, err)
	}
	return balance, nil
}

// GetCreatedAt parses and returns the createdAt timestamp
func (a *Account) GetCreatedAt() (*time.Time, error) {
	if a.CreatedAt == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse createdAt '%s': %w", a.CreatedAt, err)
	}
	return &t, nil
}

// GetUpdatedAt parses and returns the updatedAt timestamp
func (a *Account) GetUpdatedAt() (*time.Time, error) {
	if a.UpdatedAt == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updatedAt '%s': %w", a.UpdatedAt, err)
	}
	return &t, nil
}

// BankData represents bank-specific account data
type BankData struct {
	TransferNumber               string  `json:"transferNumber"`
	ClosingBalance               float64 `json:"closingBalance"`
	AutomaticallyInvestedBalance float64 `json:"automaticallyInvestedBalance"`
}

// CreditData represents credit card-specific account data
type CreditData struct {
	Brand                  string   `json:"brand"`
	Level                  string   `json:"level"`
	Status                 string   `json:"status"`
	CreditLimit            float64  `json:"creditLimit"`
	BalanceDueDate         string   `json:"balanceDueDate"`
	MinimumPayment         float64  `json:"minimumPayment"`
	BalanceCloseDate       string   `json:"balanceCloseDate"`
	AvailableCreditLimit   float64  `json:"availableCreditLimit"`
	BalanceForeignCurrency *float64 `json:"balanceForeignCurrency"`
}

// TransactionResponse represents the API response for transaction data
type TransactionResponse struct {
	Success   bool          `json:"success"`
	Data      []Transaction `json:"data"`
	Count     int           `json:"count"`
	Timestamp string        `json:"timestamp"`
}

// Transaction represents a transaction from the Open Finance API
type Transaction struct {
	ID             string                 `json:"id"`
	Description    string                 `json:"description"`
	Category       *string                `json:"category"`
	CurrencyCode   string                 `json:"currency_code"`
	AmountString   string                 `json:"amount"` // API returns amount as string
	DateString     string                 `json:"date"`   // "2025-09-28 03:00:00" format
	Type           string                 `json:"type"`   // "DEBIT" or "CREDIT"
	Status         string                 `json:"status"` // "PENDING" or "POSTED"
	CreditCardData *TransactionCreditData `json:"credit_card_data,omitempty"`
	AccountName    string                 `json:"account_name"`
	AccountType    string                 `json:"account_type"`
	AccountSubtype string                 `json:"account_subtype"`
	ItemBankName   string                 `json:"item_bank_name"` // Bank name from the item
}

// TransactionCreditData represents credit card specific data for a transaction
type TransactionCreditData struct {
	PurchaseDateString string `json:"purchaseDate"`      // ISO 8601 format
	InstallmentNumber  string `json:"installmentNumber"` // API returns as string
	TotalInstallments  string `json:"totalInstallments"` // API returns as string
}

// GetAmount returns the amount as a float64
func (t *Transaction) GetAmount() (float64, error) {
	if t.AmountString == "" {
		return 0, nil
	}
	amount, err := strconv.ParseFloat(t.AmountString, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse amount '%s': %w", t.AmountString, err)
	}
	return amount, nil
}

// GetDate parses and returns the transaction date
func (t *Transaction) GetDate() (*time.Time, error) {
	if t.DateString == "" {
		return nil, nil
	}
	// Format: "2025-09-28 03:00:00"
	parsed, err := time.Parse("2006-01-02 15:04:05", t.DateString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse date '%s': %w", t.DateString, err)
	}
	return &parsed, nil
}

// GetPurchaseDate parses and returns the purchase date from credit card data
func (c *TransactionCreditData) GetPurchaseDate() (*time.Time, error) {
	if c == nil || c.PurchaseDateString == "" {
		return nil, nil
	}
	// Format: "2025-03-23T21:40:57.001Z"
	parsed, err := time.Parse(time.RFC3339Nano, c.PurchaseDateString)
	if err != nil {
		// Try without nanoseconds
		parsed, err = time.Parse(time.RFC3339, c.PurchaseDateString)
		if err != nil {
			return nil, fmt.Errorf("failed to parse purchaseDate '%s': %w", c.PurchaseDateString, err)
		}
	}
	return &parsed, nil
}

// GetInstallmentNumber returns the installment number as an int
func (c *TransactionCreditData) GetInstallmentNumber() (int, error) {
	if c == nil || c.InstallmentNumber == "" {
		return 0, nil
	}
	num, err := strconv.Atoi(c.InstallmentNumber)
	if err != nil {
		return 0, fmt.Errorf("failed to parse installmentNumber '%s': %w", c.InstallmentNumber, err)
	}
	return num, nil
}

// GetTotalInstallments returns the total installments as an int
func (c *TransactionCreditData) GetTotalInstallments() (int, error) {
	if c == nil || c.TotalInstallments == "" {
		return 0, nil
	}
	num, err := strconv.Atoi(c.TotalInstallments)
	if err != nil {
		return 0, fmt.Errorf("failed to parse totalInstallments '%s': %w", c.TotalInstallments, err)
	}
	return num, nil
}

// BillResponse represents the API response for past due credit card bills
// Endpoint: GET /tools/api/get-bills
type BillResponse struct {
	Success   bool        `json:"success"`
	Data      []Bill      `json:"data"`
	Count     int         `json:"count"`
	Filters   BillFilters `json:"filters"`
	Timestamp string      `json:"timestamp"`
}

// BillFilters represents the filters applied to the bills query
type BillFilters struct {
	AccountID   *string `json:"accountId"`
	OnlyPastDue bool    `json:"onlyPastDue"`
}

// Bill represents a past due credit card bill (fatura vencida) from the Open Finance API
type Bill struct {
	ID                   string  `json:"id"`
	AccountID            string  `json:"accountId"`
	DueDateString        string  `json:"dueDate"`        // Vencimento
	CloseDateString      *string `json:"closeDate"`      // Fechamento
	TotalAmountString    string  `json:"totalAmount"`    // Valor total da fatura
	MinimumPaymentString *string `json:"minimumPayment"` // Pagamento mínimo
	Status               string  `json:"status"`         // OPEN, CLOSED, OVERDUE, PAID
	IsOverdue            bool    `json:"isOverdue"`      // Always true from get-bills endpoint
	// Account context from API
	AccountName    string `json:"account_name"`
	AccountType    string `json:"account_type"`
	AccountSubtype string `json:"account_subtype"`
	ItemBankName   string `json:"item_bank_name"`
	// Timestamps
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// GetTotalAmount returns the total amount as a float64
func (b *Bill) GetTotalAmount() (float64, error) {
	if b.TotalAmountString == "" {
		return 0, nil
	}
	amount, err := strconv.ParseFloat(b.TotalAmountString, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse totalAmount '%s': %w", b.TotalAmountString, err)
	}
	return amount, nil
}

// GetMinimumPayment returns the minimum payment as a float64
func (b *Bill) GetMinimumPayment() (*float64, error) {
	if b.MinimumPaymentString == nil || *b.MinimumPaymentString == "" {
		return nil, nil
	}
	amount, err := strconv.ParseFloat(*b.MinimumPaymentString, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse minimumPayment '%s': %w", *b.MinimumPaymentString, err)
	}
	return &amount, nil
}

// GetDueDate parses and returns the due date
func (b *Bill) GetDueDate() (*time.Time, error) {
	if b.DueDateString == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, b.DueDateString)
	if err != nil {
		parsed, err = time.Parse("2006-01-02", b.DueDateString)
		if err != nil {
			parsed, err = time.Parse("2006-01-02 15:04:05", b.DueDateString)
			if err != nil {
				return nil, fmt.Errorf("failed to parse dueDate '%s': %w", b.DueDateString, err)
			}
		}
	}
	return &parsed, nil
}

// GetCloseDate parses and returns the close date if present
func (b *Bill) GetCloseDate() (*time.Time, error) {
	if b.CloseDateString == nil || *b.CloseDateString == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, *b.CloseDateString)
	if err != nil {
		parsed, err = time.Parse("2006-01-02", *b.CloseDateString)
		if err != nil {
			parsed, err = time.Parse("2006-01-02 15:04:05", *b.CloseDateString)
			if err != nil {
				return nil, fmt.Errorf("failed to parse closeDate '%s': %w", *b.CloseDateString, err)
			}
		}
	}
	return &parsed, nil
}

// GetCreatedAt parses and returns the createdAt timestamp
func (b *Bill) GetCreatedAt() (*time.Time, error) {
	if b.CreatedAt == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, b.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse createdAt '%s': %w", b.CreatedAt, err)
	}
	return &t, nil
}

// GetUpdatedAt parses and returns the updatedAt timestamp
func (b *Bill) GetUpdatedAt() (*time.Time, error) {
	if b.UpdatedAt == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, b.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updatedAt '%s': %w", b.UpdatedAt, err)
	}
	return &t, nil
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

// GetAccountsWithStatus fetches accounts and returns both the response and HTTP status code.
// This allows callers to handle different status codes (e.g., 401) while still parsing successful responses.
func (c *Client) GetAccountsWithStatus(ctx context.Context, apiKey string) (*AccountResponse, int, error) {
	url := c.baseURL + accountsPath

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return nil, resp.StatusCode, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}
		return nil, resp.StatusCode, fmt.Errorf("API error (status %d): %s - %s", resp.StatusCode, errResp.Error, errResp.Message)
	}

	var accountResp AccountResponse
	if err := json.Unmarshal(body, &accountResp); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !accountResp.Success {
		return nil, resp.StatusCode, fmt.Errorf("API returned success=false")
	}

	return &accountResp, resp.StatusCode, nil
}

// GetAccounts fetches all accounts for a user using their API key
func (c *Client) GetAccounts(ctx context.Context, apiKey string) (*AccountResponse, error) {
	resp, _, err := c.GetAccountsWithStatus(ctx, apiKey)
	return resp, err
}

// GetTransactions fetches all transactions for a user using their API key
func (c *Client) GetTransactions(ctx context.Context, apiKey string) (*TransactionResponse, error) {
	url := c.baseURL + transactionsPath

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add Bearer token authentication
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle non-200 status codes
	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("API error (status %d): %s - %s", resp.StatusCode, errResp.Error, errResp.Message)
	}

	var txResp TransactionResponse
	if err := json.Unmarshal(body, &txResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !txResp.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &txResp, nil
}

// GetBills fetches all bills for a user using their API key
func (c *Client) GetBills(ctx context.Context, apiKey string) (*BillResponse, error) {
	url := c.baseURL + billsPath

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("API error (status %d): %s - %s", resp.StatusCode, errResp.Error, errResp.Message)
	}

	var billResp BillResponse
	if err := json.Unmarshal(body, &billResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !billResp.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &billResp, nil
}
