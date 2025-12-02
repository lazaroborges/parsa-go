package http

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"parsa/internal/domain/account"
	"parsa/internal/shared/middleware"
)

// AccountHandler is the refactored handler using the service layer
type AccountHandler struct {
	accountService *account.Service
}

// NewAccountHandler creates a new account handler with service layer
func NewAccountHandler(accountService *account.Service) *AccountHandler {
	return &AccountHandler{accountService: accountService}
}

// HTTP request/response types (transport layer concerns)
type CreateAccountRequest struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	AccountType string  `json:"accountType"`
	Currency    string  `json:"currency"`
	Balance     float64 `json:"balance"`
}

// AccountResponse is the mobile-friendly response format
type AccountResponse struct {
	AccountID     string   `json:"accountId"`
	BankName      string   `json:"bankName"`
	AccountType   string   `json:"accountType"` // "normal", "credit", "saving"
	Number        string   `json:"number"`      // empty for now
	Name          string   `json:"name"`
	InitialValue  float64  `json:"initialValue"`
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt"`
	ConnectorID   string   `json:"connectorId"`
	PrimaryColor  string   `json:"primaryColor"`
	Balance       *float64 `json:"balance"`
	IsOpenFinance bool     `json:"isOpenFinance"`
	ClosedAt      *string  `json:"closedAt"`
	Order         int      `json:"order"`
	Description   string   `json:"description"`
	Removed       bool     `json:"removed"`
	HiddenByUser  bool     `json:"hiddenByUser"`
	HasMFA        bool     `json:"hasMFA"` // false for now
}

// HandleAccounts is the single entry point for /api/accounts/ routes
// Routes: GET /api/accounts/ -> list, GET /api/accounts/{id} -> get, DELETE /api/accounts/{id} -> delete
func (h *AccountHandler) HandleAccounts(w http.ResponseWriter, r *http.Request) {
	// Extract account ID from path (empty string means list all)
	accountID := strings.TrimPrefix(r.URL.Path, "/api/accounts/")
	accountID = strings.TrimSuffix(accountID, "/") // normalize trailing slash

	switch r.Method {
	case http.MethodGet:
		if accountID == "" {
			h.handleListAccounts(w, r)
		} else {
			h.handleGetAccount(w, r, accountID)
		}
	case http.MethodDelete:
		if accountID == "" {
			http.Error(w, "Account ID is required", http.StatusBadRequest)
			return
		}
		h.handleDeleteAccount(w, r, accountID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListAccounts returns all accounts for the authenticated user
func (h *AccountHandler) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	accounts, err := h.accountService.ListAccountsWithBankByUserID(r.Context(), userID)
	if err != nil {
		log.Printf("Error listing accounts for user %d: %v", userID, err)
		http.Error(w, "Failed to list accounts", http.StatusInternalServerError)
		return
	}

	response := make([]AccountResponse, 0, len(accounts))
	for _, acc := range accounts {
		response = append(response, toAccountResponse(acc))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetAccount returns a specific account
func (h *AccountHandler) handleGetAccount(w http.ResponseWriter, r *http.Request, accountID string) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	acc, err := h.accountService.GetAccount(r.Context(), accountID, userID)
	if err != nil {
		switch err {
		case account.ErrAccountNotFound:
			http.Error(w, "Account not found", http.StatusNotFound)
		case account.ErrForbidden:
			http.Error(w, "Forbidden", http.StatusForbidden)
		default:
			log.Printf("Error getting account %s: %v", accountID, err)
			http.Error(w, "Failed to get account", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(acc)
}

// handleDeleteAccount deletes an account
func (h *AccountHandler) handleDeleteAccount(w http.ResponseWriter, r *http.Request, accountID string) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err := h.accountService.DeleteAccount(r.Context(), accountID, userID)
	if err != nil {
		switch err {
		case account.ErrAccountNotFound:
			http.Error(w, "Account not found", http.StatusNotFound)
		case account.ErrForbidden:
			http.Error(w, "Forbidden", http.StatusForbidden)
		default:
			log.Printf("Error deleting account %s: %v", accountID, err)
			http.Error(w, "Failed to delete account", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// toAccountResponse transforms an AccountWithBank to the mobile-friendly AccountResponse
func toAccountResponse(acc *account.AccountWithBank) AccountResponse {
	// Map subtype to mobile account_type
	accountType := mapAccountType(acc.Subtype)

	// Build bank_name with suffix based on subtype
	bankName := buildBankName(acc)

	// Get connector_id (default to "1" if empty)
	connectorID := acc.BankConnector
	if connectorID == "" {
		connectorID = "1"
	}

	// Get primary_color (default to "1194F6" if empty)
	primaryColor := acc.BankPrimaryColor
	if primaryColor == "" {
		primaryColor = "1194F6"
	}

	// Format timestamps
	createdAt := acc.CreatedAt.Format(time.RFC3339)
	updatedAt := acc.UpdatedAt.Format(time.RFC3339)

	// Handle nullable closed_at
	var closedAt *string
	if !acc.ClosedAt.IsZero() {
		formatted := acc.ClosedAt.Format(time.RFC3339)
		closedAt = &formatted
	}

	// Balance as pointer
	balance := acc.Balance

	return AccountResponse{
		AccountID:     acc.ID,
		BankName:      bankName,
		AccountType:   accountType,
		Number:        "", // empty for now
		Name:          acc.Name,
		InitialValue:  acc.InitialBalance,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		ConnectorID:   connectorID,
		PrimaryColor:  primaryColor,
		Balance:       &balance,
		IsOpenFinance: acc.IsOpenFinanceAccount,
		ClosedAt:      closedAt,
		Order:         acc.UIOrder,
		Description:   acc.Description,
		Removed:       acc.Removed,
		HiddenByUser:  acc.HiddenByUser,
		HasMFA:        false, // always false for now
	}
}

// mapAccountType maps the database subtype to mobile account_type
func mapAccountType(subtype string) string {
	switch subtype {
	case "SAVINGS_ACCOUNT":
		return "saving"
	case "CHECKING_ACCOUNT":
		return "normal"
	case "CREDIT_CARD":
		return "credit"
	default:
		return "normal"
	}
}

// buildBankName constructs the display name for the bank
func buildBankName(acc *account.AccountWithBank) string {
	// Use ui_name if available, otherwise fall back to name
	baseName := acc.BankUIName
	if baseName == "" {
		baseName = acc.BankName
	}
	if baseName == "" {
		baseName = "Unknown Bank"
	}

	// Append suffix based on subtype
	switch acc.Subtype {
	case "SAVINGS_ACCOUNT":
		return baseName + " - Poupança"
	case "CREDIT_CARD":
		return baseName + " - " + acc.Name
	default:
		return baseName + " - Conta Corrente"
	}
}
