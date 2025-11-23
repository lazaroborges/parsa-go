package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"parsa/internal/database"
	"parsa/internal/middleware"
	"parsa/internal/models"
)

type AccountHandler struct {
	accountRepo *database.AccountRepository
}

func NewAccountHandler(accountRepo *database.AccountRepository) *AccountHandler {
	return &AccountHandler{accountRepo: accountRepo}
}

type CreateAccountRequest struct {
	Name        string  `json:"name"`
	AccountType string  `json:"account_type"`
	Currency    string  `json:"currency"`
	Balance     float64 `json:"balance"`
}

// HandleListAccounts returns all accounts for the authenticated user
func (h *AccountHandler) HandleListAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	accounts, err := h.accountRepo.ListByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to list accounts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accounts)
}

// HandleCreateAccount creates a new account
func (h *AccountHandler) HandleCreateAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.AccountType == "" {
		http.Error(w, "Name and account_type are required", http.StatusBadRequest)
		return
	}

	if req.Currency == "" {
		req.Currency = "USD"
	}

	account, err := h.accountRepo.Create(r.Context(), models.CreateAccountParams{
		UserID:      userID,
		Name:        req.Name,
		AccountType: req.AccountType,
		Currency:    req.Currency,
		Balance:     req.Balance,
	})

	if err != nil {
		http.Error(w, "Failed to create account", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(account)
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
	accountIDStr := strings.TrimPrefix(r.URL.Path, "/api/accounts/")
	if accountIDStr == "" {
		http.Error(w, "Account ID is required", http.StatusBadRequest)
		return
	}

	accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid account ID", http.StatusBadRequest)
		return
	}

	account, err := h.accountRepo.GetByID(r.Context(), accountID)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	// Verify ownership
	if account.UserID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
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

	accountIDStr := strings.TrimPrefix(r.URL.Path, "/api/accounts/")
	if accountIDStr == "" {
		http.Error(w, "Account ID is required", http.StatusBadRequest)
		return
	}

	accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid account ID", http.StatusBadRequest)
		return
	}

	// Verify ownership before deleting
	account, err := h.accountRepo.GetByID(r.Context(), accountID)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	if account.UserID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.accountRepo.Delete(r.Context(), accountID); err != nil {
		http.Error(w, "Failed to delete account", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
