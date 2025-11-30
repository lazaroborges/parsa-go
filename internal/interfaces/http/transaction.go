package http

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"parsa/internal/infrastructure/postgres"
	"parsa/internal/shared/middleware"
	"parsa/internal/domain/transaction"

	"github.com/google/uuid"
)

type TransactionHandler struct {
	transactionRepo *postgres.TransactionRepository
	accountRepo     *postgres.AccountRepository
}

func NewTransactionHandler(transactionRepo *postgres.TransactionRepository, accountRepo *postgres.AccountRepository) *TransactionHandler {
	return &TransactionHandler{
		transactionRepo: transactionRepo,
		accountRepo:     accountRepo,
	}
}

type CreateTransactionRequest struct {
	AccountID       string  `json:"accountId"`
	Amount          float64 `json:"amount"`
	Description     string  `json:"description"`
	Category        *string `json:"category,omitempty"`
	TransactionDate string  `json:"transactionDate"`
	Type            string  `json:"type,omitempty"`   // DEBIT or CREDIT, defaults to DEBIT
	Status          string  `json:"status,omitempty"` // PENDING or POSTED, defaults to POSTED
}

// HandleListTransactions returns transactions for a specific account
func (h *TransactionHandler) HandleListTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	accountID := r.URL.Query().Get("accountId")
	if accountID == "" {
		http.Error(w, "accountId is required", http.StatusBadRequest)
		return
	}

	// Verify account ownership
	account, err := h.accountRepo.GetByID(r.Context(), accountID)
	if err != nil {
		log.Printf("Error getting account %s for transaction list: %v", accountID, err)
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	if account.UserID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Parse pagination parameters
	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	transactions, err := h.transactionRepo.ListByAccountID(r.Context(), accountID, limit, offset)
	if err != nil {
		log.Printf("Error listing transactions for account %s: %v", accountID, err)
		http.Error(w, "Failed to list transactions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transactions)
}

// HandleCreateTransaction creates a new transaction
func (h *TransactionHandler) HandleCreateTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding create transaction request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AccountID == "" || req.Description == "" || req.TransactionDate == "" {
		http.Error(w, "accountId, description, and transactionDate are required", http.StatusBadRequest)
		return
	}

	// Verify account ownership
	account, err := h.accountRepo.GetByID(r.Context(), req.AccountID)
	if err != nil {
		log.Printf("Error getting account %s for transaction creation: %v", req.AccountID, err)
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	if account.UserID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Parse transaction date
	transactionDate, err := time.Parse("2006-01-02", req.TransactionDate)
	if err != nil {
		http.Error(w, "Invalid transactionDate format (use YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	// Set defaults
	txType := req.Type
	if txType == "" {
		txType = "DEBIT"
	}
	txStatus := req.Status
	if txStatus == "" {
		txStatus = "POSTED"
	}

	// Generate UUID for manual transactions
	txID := uuid.New().String()

	transaction, err := h.transactionRepo.Create(r.Context(), transaction.CreateTransactionParams{
		ID:              txID,
		AccountID:       req.AccountID,
		Amount:          req.Amount,
		Description:     req.Description,
		Category:        req.Category,
		TransactionDate: transactionDate,
		Type:            txType,
		Status:          txStatus,
	})

	if err != nil {
		log.Printf("Error creating transaction for account %s: %v", req.AccountID, err)
		http.Error(w, "Failed to create transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(transaction)
}

// HandleGetTransaction returns a specific transaction
func (h *TransactionHandler) HandleGetTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	transactionID := strings.TrimPrefix(r.URL.Path, "/api/transactions/")
	if transactionID == "" {
		http.Error(w, "Transaction ID is required", http.StatusBadRequest)
		return
	}

	transaction, err := h.transactionRepo.GetByID(r.Context(), transactionID)
	if err != nil {
		log.Printf("Error getting transaction %s: %v", transactionID, err)
		http.Error(w, "Failed to get transaction", http.StatusInternalServerError)
		return
	}
	if transaction == nil {
		http.Error(w, "Transaction not found", http.StatusNotFound)
		return
	}

	// Verify ownership through account
	account, err := h.accountRepo.GetByID(r.Context(), transaction.AccountID)
	if err != nil {
		log.Printf("Error getting account %s for transaction %s: %v", transaction.AccountID, transactionID, err)
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	if account.UserID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transaction)
}

// HandleDeleteTransaction deletes a transaction
func (h *TransactionHandler) HandleDeleteTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	transactionID := strings.TrimPrefix(r.URL.Path, "/api/transactions/")
	if transactionID == "" {
		http.Error(w, "Transaction ID is required", http.StatusBadRequest)
		return
	}

	transaction, err := h.transactionRepo.GetByID(r.Context(), transactionID)
	if err != nil {
		log.Printf("Error getting transaction %s for deletion: %v", transactionID, err)
		http.Error(w, "Failed to get transaction", http.StatusInternalServerError)
		return
	}
	if transaction == nil {
		http.Error(w, "Transaction not found", http.StatusNotFound)
		return
	}

	// Verify ownership through account
	account, err := h.accountRepo.GetByID(r.Context(), transaction.AccountID)
	if err != nil {
		log.Printf("Error getting account %s for transaction deletion: %v", transaction.AccountID, err)
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	if account.UserID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.transactionRepo.Delete(r.Context(), transactionID); err != nil {
		log.Printf("Error deleting transaction %s: %v", transactionID, err)
		http.Error(w, "Failed to delete transaction", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
