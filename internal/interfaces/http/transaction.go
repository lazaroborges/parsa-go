package http

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"parsa/internal/domain/account"
	"parsa/internal/domain/transaction"
	"parsa/internal/shared/middleware"

	"github.com/google/uuid"
)

const pageSize = 100

// TransactionListResponse is the paginated response for transaction list
type TransactionListResponse struct {
	Count    int64                    `json:"count"`
	Next     *string                  `json:"next"`
	Previous *string                  `json:"previous"`
	Results  []TransactionAPIResponse `json:"results"`
}

// TransactionAPIResponse is the API response format for a transaction
type TransactionAPIResponse struct {
	ID                  string   `json:"id"`
	Description         string   `json:"description"`
	Amount              float64  `json:"amount"`
	Notes               *string  `json:"notes"`
	Currency            string   `json:"currency"`
	Account             string   `json:"account"`
	Category            string   `json:"category"`
	Type                string   `json:"type"`
	TransactionDate     string   `json:"transactionDate"`
	Status              string   `json:"status"`
	Considered          bool     `json:"considered"`
	IsOpenFinance       bool     `json:"isOpenFinance"`
	Tags                []string `json:"tags"`
	Manipulated         bool     `json:"manipulated"`
	LastUpdateDateParsa string   `json:"lastUpdateDateParsa"`
	Cousin              *string  `json:"cousin"`
	DontAskAgain        bool     `json:"dont_ask_again"`
}

type TransactionHandler struct {
	transactionRepo transaction.Repository
	accountRepo     account.Repository
}

func NewTransactionHandler(transactionRepo transaction.Repository, accountRepo account.Repository) *TransactionHandler {
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

// HandleListTransactions returns paginated transactions for a user (optionally filtered by account)
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

	// Parse page parameter (default 1)
	page := 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if parsedPage, err := strconv.Atoi(pageStr); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	offset := (page - 1) * pageSize

	// Get total count and transactions
	count, err := h.transactionRepo.CountByUserID(r.Context(), userID)
	if err != nil {
		log.Printf("Error counting transactions for user %d: %v", userID, err)
		http.Error(w, "Failed to count transactions", http.StatusInternalServerError)
		return
	}

	transactions, err := h.transactionRepo.ListByUserID(r.Context(), userID, pageSize, offset)
	if err != nil {
		log.Printf("Error listing transactions for user %d: %v", userID, err)
		http.Error(w, "Failed to list transactions", http.StatusInternalServerError)
		return
	}

	// Build pagination URLs
	baseURL := fmt.Sprintf("%s://%s%s", getScheme(r), r.Host, r.URL.Path)
	totalPages := int(math.Ceil(float64(count) / float64(pageSize)))

	var next, previous *string
	if page < totalPages {
		nextURL := fmt.Sprintf("%s?page=%d", baseURL, page+1)
		next = &nextURL
	}
	if page > 1 {
		prevURL := fmt.Sprintf("%s?page=%d", baseURL, page-1)
		previous = &prevURL
	}

	// Transform transactions to API response format
	results := make([]TransactionAPIResponse, 0, len(transactions))
	for _, txn := range transactions {
		results = append(results, toTransactionAPIResponse(txn))
	}

	response := TransactionListResponse{
		Count:    count,
		Next:     next,
		Previous: previous,
		Results:  results,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// toTransactionAPIResponse converts a domain Transaction to the API response format
func toTransactionAPIResponse(txn *transaction.Transaction) TransactionAPIResponse {
	// Amount should be absolute value
	amount := txn.Amount
	if txn.Type == "DEBIT" {
		amount = -math.Abs(amount)
	} else if txn.Type == "CREDIT" {
		amount = math.Abs(amount)
	} else {
		amount = math.Abs(amount)
	}

	// Handle nil Category pointer safely
	category := ""
	if txn.Category != nil {
		category = *txn.Category
	}

	return TransactionAPIResponse{
		ID:                  txn.ID,
		Description:         txn.Description,
		Amount:              amount,
		Notes:               nil, // Not stored yet
		Currency:            "BRL",
		Account:             txn.AccountID,
		Category:            category,
		Type:                strings.ToLower(txn.Type),
		TransactionDate:     txn.TransactionDate.Format(time.RFC3339),
		Status:              strings.ToLower(txn.Status),
		Considered:          txn.Considered,
		IsOpenFinance:       txn.IsOpenFinance,
		Tags:                []string{}, // Return empty array for now
		Manipulated:         txn.Manipulated,
		LastUpdateDateParsa: txn.UpdatedAt.Format(time.RFC3339),
		Cousin:              nil,   // Return null for now
		DontAskAgain:        false, // Default false
	}
}

// getScheme returns the request scheme (http or https)
func getScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	// Check X-Forwarded-Proto header for proxy support
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	return "http"
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

	txn, err := h.transactionRepo.Create(r.Context(), transaction.CreateTransactionParams{
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
	json.NewEncoder(w).Encode(txn)
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

	// Use PathValue instead of TrimPrefix
	transactionID := r.PathValue("id")
	if transactionID == "" {
		http.Error(w, "Transaction ID is required", http.StatusBadRequest)
		return
	}

	txn, err := h.transactionRepo.GetByID(r.Context(), transactionID)
	if err != nil {
		log.Printf("Error getting transaction %s: %v", transactionID, err)
		http.Error(w, "Failed to get transaction", http.StatusInternalServerError)
		return
	}
	if txn == nil {
		http.Error(w, "Transaction not found", http.StatusNotFound)
		return
	}

	// Verify ownership through account
	account, err := h.accountRepo.GetByID(r.Context(), txn.AccountID)
	if err != nil {
		log.Printf("Error getting account %s for transaction %s: %v", txn.AccountID, transactionID, err)
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	if account.UserID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txn)
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

	txn, err := h.transactionRepo.GetByID(r.Context(), transactionID)
	if err != nil {
		log.Printf("Error getting transaction %s for deletion: %v", transactionID, err)
		http.Error(w, "Failed to get transaction", http.StatusInternalServerError)
		return
	}
	if txn == nil {
		http.Error(w, "Transaction not found", http.StatusNotFound)
		return
	}

	// Verify ownership through account
	account, err := h.accountRepo.GetByID(r.Context(), txn.AccountID)
	if err != nil {
		log.Printf("Error getting account %s for transaction deletion: %v", txn.AccountID, err)
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
