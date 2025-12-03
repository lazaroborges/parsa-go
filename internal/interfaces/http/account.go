package http

import (
	"encoding/json"
	"log"
	"net/http"
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
	ConnectorID   string   `json:"connectorID"`
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

// HandleListAccounts returns all accounts for the authenticated user
func (h *AccountHandler) HandleListAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

	// Return all accounts
	response := make([]AccountResponse, 0, len(accounts))
	for _, acc := range accounts {
		response = append(response, toAccountResponse(acc))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleAccountByID handles operations on a specific account (GET and DELETE)
func (h *AccountHandler) HandleAccountByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Use PathValue to extract the account ID
	accountID := r.PathValue("id")
	if accountID == "" {
		http.Error(w, "Account ID is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGetAccountByID(w, r, userID, accountID)
	case http.MethodDelete:
		h.handleDeleteAccount(w, r, userID, accountID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetAccountByID returns a specific account
func (h *AccountHandler) handleGetAccountByID(w http.ResponseWriter, r *http.Request, userID int64, accountID string) {
	accounts, err := h.accountService.ListAccountsWithBankByUserID(r.Context(), userID)
	if err != nil {
		log.Printf("Error listing accounts for user %d: %v", userID, err)
		http.Error(w, "Failed to list accounts", http.StatusInternalServerError)
		return
	}

	// Find the specific account
	var found *account.AccountWithBank
	for _, acc := range accounts {
		if acc.ID == accountID {
			found = acc
			break
		}
	}
	if found == nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toAccountResponse(found))
}

// handleDeleteAccount deletes an account
func (h *AccountHandler) handleDeleteAccount(w http.ResponseWriter, r *http.Request, userID int64, accountID string) {
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

	// bankName is just the ui_name (or name if ui_name is empty)
	bankName := acc.BankUIName
	if bankName == "" {
		bankName = acc.BankName
	}
	if bankName == "" {
		bankName = "Unknown Bank"
	}

	// Name uses the concatenation logic (ui_name + suffix based on subtype)
	accountName := buildAccountName(acc)

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
		Name:          accountName,
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

// buildAccountName constructs the display name for the account
// Uses suffix based on account subtype
func buildAccountName(acc *account.AccountWithBank) string {
	// Use ui_name if available, otherwise fall back to name
	// Append suffix based on subtype
	switch acc.Subtype {
	case "SAVINGS_ACCOUNT":
		return "Poupança"
	case "CREDIT_CARD":
		return acc.Name
	default:
		return "Conta Corrente"
	}
}
