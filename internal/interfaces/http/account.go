package http

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

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

// HandleListAccounts returns all accounts for the authenticated user
func (h *AccountHandler) HandleListAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user ID from context (set by auth middleware)
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Call service (business logic)
	accounts, err := h.accountService.ListAccountsByUserID(r.Context(), userID)
	if err != nil {
		log.Printf("Error listing accounts for user %d: %v", userID, err)
		http.Error(w, "Failed to list accounts", http.StatusInternalServerError)
		return
	}

	// Return HTTP response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accounts)
}

// HandleCreateAccount creates a new account
func (h *AccountHandler) HandleCreateAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse HTTP request
	var req CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Call service (validation and business logic handled there)
	acc, err := h.accountService.CreateAccount(r.Context(), account.CreateParams{
		ID:          req.ID,
		UserID:      userID,
		Name:        req.Name,
		AccountType: req.AccountType,
		Currency:    req.Currency,
		Balance:     req.Balance,
	})

	// Handle errors with appropriate HTTP status codes
	if err != nil {
		switch err {
		case account.ErrInvalidAccountType:
			http.Error(w, err.Error(), http.StatusBadRequest)
		case account.ErrInvalidInput:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			if strings.Contains(err.Error(), "required") {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				log.Printf("Error creating account for user %d: %v", userID, err)
				http.Error(w, "Failed to create account", http.StatusInternalServerError)
			}
		}
		return
	}

	// Return HTTP response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(acc)
}

// HandleGetAccount returns a specific account
func (h *AccountHandler) HandleGetAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract account ID from URL path
	accountID := strings.TrimPrefix(r.URL.Path, "/api/accounts/")
	if accountID == "" {
		http.Error(w, "Account ID is required", http.StatusBadRequest)
		return
	}

	// Service handles ownership verification
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

// HandleDeleteAccount deletes an account
func (h *AccountHandler) HandleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract account ID from URL path
	accountID := strings.TrimPrefix(r.URL.Path, "/api/accounts/")
	if accountID == "" {
		http.Error(w, "Account ID is required", http.StatusBadRequest)
		return
	}

	// Service handles ownership verification and deletion
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
