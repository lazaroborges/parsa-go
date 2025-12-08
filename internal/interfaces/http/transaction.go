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
	Cousin              *int64   `json:"cousin"`
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

// BatchCreateRequest wraps multiple transaction create requests
type BatchCreateRequest struct {
	Transactions []CreateTransactionRequest `json:"transactions"`
}

// PatchTransactionItem represents a single transaction patch with ID
type PatchTransactionItem struct {
	ID          string    `json:"id"`
	Description *string   `json:"description,omitempty"`
	Category    *string   `json:"category,omitempty"`
	Considered  *bool     `json:"considered,omitempty"`
	Notes       *string   `json:"notes,omitempty"`
	Tags        *[]string `json:"tags,omitempty"` // nil = don't update, empty = clear all
}

// BatchPatchRequest wraps multiple transaction patch requests
type BatchPatchRequest struct {
	Transactions []PatchTransactionItem `json:"transactions"`
}

// BatchItemResult represents the result of a single batch operation item
type BatchItemResult struct {
	Index       int                     `json:"index"`
	Success     bool                    `json:"success"`
	Transaction *TransactionAPIResponse `json:"transaction,omitempty"`
	Error       string                  `json:"error,omitempty"`
}

// BatchResponse is the response for batch operations
type BatchResponse struct {
	TotalCount   int               `json:"totalCount"`
	SuccessCount int               `json:"successCount"`
	FailureCount int               `json:"failureCount"`
	Results      []BatchItemResult `json:"results"`
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

	// Fetch tags for each transaction and transform to API response format
	results := make([]TransactionAPIResponse, 0, len(transactions))
	for _, txn := range transactions {
		tags, err := h.transactionRepo.GetTransactionTags(r.Context(), txn.ID)
		if err != nil {
			log.Printf("Error getting tags for transaction %s: %v", txn.ID, err)
			txn.Tags = []string{}
		} else if tags != nil {
			txn.Tags = tags
		} else {
			txn.Tags = []string{}
		}
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

	// Return cousin ID as integer pointer (null if nil or 0)
	var cousin *int64
	if txn.Cousin != nil && *txn.Cousin != 0 {
		cousin = txn.Cousin
	}

	// Use actual tags or empty slice
	tags := txn.Tags
	if tags == nil {
		tags = []string{}
	}

	return TransactionAPIResponse{
		ID:                  txn.ID,
		Description:         txn.Description,
		Amount:              amount,
		Notes:               txn.Notes,
		Currency:            "BRL",
		Account:             txn.AccountID,
		Category:            category,
		Type:                strings.ToLower(txn.Type),
		TransactionDate:     txn.TransactionDate.Format(time.RFC3339),
		Status:              strings.ToLower(txn.Status),
		Considered:          txn.Considered,
		IsOpenFinance:       txn.IsOpenFinance,
		Tags:                tags,
		Manipulated:         txn.Manipulated,
		LastUpdateDateParsa: txn.UpdatedAt.Format(time.RFC3339),
		Cousin:              cousin,
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

// HandleBatchTransactions routes batch requests to POST or PATCH handlers
func (h *TransactionHandler) HandleBatchTransactions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handleBatchCreate(w, r)
	case http.MethodPatch:
		h.handleBatchPatch(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleBatchCreate creates multiple transactions
func (h *TransactionHandler) handleBatchCreate(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req BatchCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding batch create request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Transactions) == 0 {
		http.Error(w, "transactions array is required and cannot be empty", http.StatusBadRequest)
		return
	}

	// Collect unique account IDs and verify ownership
	accountIDs := make(map[string]bool)
	for _, txReq := range req.Transactions {
		if txReq.AccountID == "" {
			http.Error(w, "accountId is required for all transactions", http.StatusBadRequest)
			return
		}
		accountIDs[txReq.AccountID] = true
	}

	// Verify ownership for all accounts
	for accountID := range accountIDs {
		acc, err := h.accountRepo.GetByID(r.Context(), accountID)
		if err != nil || acc == nil {
			http.Error(w, fmt.Sprintf("Account %s not found", accountID), http.StatusNotFound)
			return
		}
		if acc.UserID != userID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	// Create all transactions, collecting results for each
	results := make([]BatchItemResult, 0, len(req.Transactions))
	successCount := 0

	for idx, txReq := range req.Transactions {
		// Validate required fields
		if txReq.Description == "" || txReq.TransactionDate == "" {
			results = append(results, BatchItemResult{
				Index:   idx,
				Success: false,
				Error:   "description and transactionDate are required",
			})
			continue
		}

		transactionDate, err := time.Parse("2006-01-02", txReq.TransactionDate)
		if err != nil {
			log.Printf("Error parsing transactionDate at index %d for account %s: %v", idx, txReq.AccountID, err)
			results = append(results, BatchItemResult{
				Index:   idx,
				Success: false,
				Error:   "Invalid transactionDate format",
			})
			continue
		}

		txType := txReq.Type
		if txType == "" {
			txType = "DEBIT"
		}
		txStatus := txReq.Status
		if txStatus == "" {
			txStatus = "POSTED"
		}

		txID := uuid.New().String()

		txn, err := h.transactionRepo.Create(r.Context(), transaction.CreateTransactionParams{
			ID:              txID,
			AccountID:       txReq.AccountID,
			Amount:          txReq.Amount,
			Description:     txReq.Description,
			Category:        txReq.Category,
			TransactionDate: transactionDate,
			Type:            txType,
			Status:          txStatus,
		})

		if err != nil {
			log.Printf("Error creating transaction in batch at index %d for account %s: %v", idx, txReq.AccountID, err)
			results = append(results, BatchItemResult{
				Index:   idx,
				Success: false,
				Error:   "Failed to create transaction",
			})
			continue
		}

		successCount++
		txnResponse := toTransactionAPIResponse(txn)
		results = append(results, BatchItemResult{
			Index:       idx,
			Success:     true,
			Transaction: &txnResponse,
		})
	}

	// Determine appropriate HTTP status code
	status := http.StatusCreated
	if successCount == 0 {
		status = http.StatusBadRequest
	} else if successCount < len(req.Transactions) {
		status = 207 // Multi-Status
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(BatchResponse{
		TotalCount:   len(req.Transactions),
		SuccessCount: successCount,
		FailureCount: len(req.Transactions) - successCount,
		Results:      results,
	})
}

// handleBatchPatch updates multiple transactions
func (h *TransactionHandler) handleBatchPatch(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req BatchPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding batch patch request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Transactions) == 0 {
		http.Error(w, "transactions array is required and cannot be empty", http.StatusBadRequest)
		return
	}

	// Update all transactions, collecting results for each
	results := make([]BatchItemResult, 0, len(req.Transactions))
	successCount := 0

	for idx, patchReq := range req.Transactions {
		// Validate transaction ID
		if patchReq.ID == "" {
			results = append(results, BatchItemResult{
				Index:   idx,
				Success: false,
				Error:   "id is required",
			})
			continue
		}

		// Verify transaction exists and ownership
		txn, err := h.transactionRepo.GetByID(r.Context(), patchReq.ID)
		if err != nil {
			log.Printf("Error getting transaction %s for batch patch at index %d: %v", patchReq.ID, idx, err)
			results = append(results, BatchItemResult{
				Index:   idx,
				Success: false,
				Error:   "Failed to get transaction",
			})
			continue
		}
		if txn == nil {
			results = append(results, BatchItemResult{
				Index:   idx,
				Success: false,
				Error:   fmt.Sprintf("Transaction %s not found", patchReq.ID),
			})
			continue
		}

		acc, err := h.accountRepo.GetByID(r.Context(), txn.AccountID)
		if err != nil || acc == nil {
			log.Printf("Error getting account %s for transaction %s in batch patch at index %d: %v", txn.AccountID, patchReq.ID, idx, err)
			results = append(results, BatchItemResult{
				Index:   idx,
				Success: false,
				Error:   "Account not found",
			})
			continue
		}
		if acc.UserID != userID {
			results = append(results, BatchItemResult{
				Index:   idx,
				Success: false,
				Error:   "Forbidden: transaction does not belong to user",
			})
			continue
		}

		// Update transaction
		updatedTxn, err := h.transactionRepo.Update(r.Context(), patchReq.ID, transaction.UpdateTransactionParams{
			Description: patchReq.Description,
			Category:    patchReq.Category,
			Considered:  patchReq.Considered,
			Notes:       patchReq.Notes,
		})

		if err != nil {
			log.Printf("Error updating transaction %s in batch at index %d: %v", patchReq.ID, idx, err)
			results = append(results, BatchItemResult{
				Index:   idx,
				Success: false,
				Error:   "Failed to update transaction",
			})
			continue
		}

		// Update tags if provided
		if patchReq.Tags != nil {
			if err := h.transactionRepo.SetTransactionTags(r.Context(), patchReq.ID, *patchReq.Tags); err != nil {
				log.Printf("Error setting tags for transaction %s in batch at index %d: %v", patchReq.ID, idx, err)
				results = append(results, BatchItemResult{
					Index:   idx,
					Success: false,
					Error:   "Failed to update tags",
				})
				continue
			}
			// Update the Tags field in response
			updatedTxn.Tags = *patchReq.Tags
		} else {
			// Fetch current tags for response
			tags, err := h.transactionRepo.GetTransactionTags(r.Context(), patchReq.ID)
			if err != nil {
				log.Printf("Error getting tags for transaction %s: %v", patchReq.ID, err)
				updatedTxn.Tags = []string{}
			} else {
				updatedTxn.Tags = tags
			}
		}

		successCount++
		txnResponse := toTransactionAPIResponse(updatedTxn)
		results = append(results, BatchItemResult{
			Index:       idx,
			Success:     true,
			Transaction: &txnResponse,
		})
	}

	// Determine appropriate HTTP status code
	status := http.StatusOK
	if successCount == 0 {
		status = http.StatusBadRequest
	} else if successCount < len(req.Transactions) {
		status = 207 // Multi-Status
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(BatchResponse{
		TotalCount:   len(req.Transactions),
		SuccessCount: successCount,
		FailureCount: len(req.Transactions) - successCount,
		Results:      results,
	})
}
