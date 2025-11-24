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
	defaultTimeout   = 30 * time.Second
	accountsPath     = "/get-accounts"
	transactionsPath = "/get-transactions"
)

// Client handles communication with the Open Finance API
type Client struct {
	httpClient *http.Client
	baseURL    string
}

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
	ProviderCode         string      `json:"providerCode"`
	AccountName          string      `json:"name"`
	AccountType          string      `json:"type"`
	AccountSubtype       string      `json:"subtype"`
	AccountCurrencyCode  string      `json:"currencyCode"`
	AccountMarketingName string      `json:"marketingName"`
	BalanceString        string      `json:"balance"` // API returns balance as string
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

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

// GetAccounts fetches all accounts for a user using their API key
func (c *Client) GetAccounts(ctx context.Context, apiKey string) (*AccountResponse, error) {
	url := c.baseURL + accountsPath

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

	var accountResp AccountResponse
	if err := json.Unmarshal(body, &accountResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !accountResp.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return &accountResp, nil
}
