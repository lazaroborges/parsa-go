package http

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"parsa/internal/domain/account"
	"parsa/internal/domain/notification"
	"parsa/internal/domain/openfinance"
	"parsa/internal/domain/user"
	ofclient "parsa/internal/infrastructure/openfinance"
	"parsa/internal/shared/messages"
	"parsa/internal/shared/middleware"
)

type UserHandler struct {
	userRepo               user.Repository
	accountRepo            account.Repository
	ofClient               ofclient.ClientInterface
	accountSyncService     *openfinance.AccountSyncService
	transactionSyncService *openfinance.TransactionSyncService
	billSyncService        *openfinance.BillSyncService
	notificationService    *notification.Service
	msgs                   *messages.Messages
}

func NewUserHandler(
	userRepo user.Repository,
	accountRepo account.Repository,
	ofClient ofclient.ClientInterface,
	accountSyncService *openfinance.AccountSyncService,
	transactionSyncService *openfinance.TransactionSyncService,
	billSyncService *openfinance.BillSyncService,
	notificationService *notification.Service,
	msgs *messages.Messages,
) *UserHandler {
	return &UserHandler{
		userRepo:               userRepo,
		accountRepo:            accountRepo,
		ofClient:               ofClient,
		accountSyncService:     accountSyncService,
		transactionSyncService: transactionSyncService,
		billSyncService:        billSyncService,
		notificationService:    notificationService,
		msgs:                   msgs,
	}
}

// HandleMe handles both GET and PATCH requests for the current user
func (h *UserHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGetMe(w, r, userID)
	case http.MethodPatch:
		h.handleUpdateMe(w, r, userID)
	default:
		log.Printf("Method not allowed for /api/users/me: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

type UserResponse struct {
	*user.User
	HasValidKey bool `json:"hasValidKey"`
}

func (h *UserHandler) handleGetMe(w http.ResponseWriter, r *http.Request, userID int64) {
	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Calculate balanceAvailable: SUM of ABS balance of CHECKING_ACCOUNT
	balanceAvailable, err := h.accountRepo.GetBalanceSumBySubtype(r.Context(), userID, []string{"CHECKING_ACCOUNT"})
	if err != nil {
		log.Printf("Error calculating checking account balance for user %d: %v", userID, err)
		balanceAvailable = 0
	}
	user.BalanceAvailable = &balanceAvailable

	// Calculate balanceTotal: SUM of ABS balance of CHECKING_ACCOUNT and SAVINGS_ACCOUNT
	// minus the ABS balance sum of all CREDIT_CARD
	checkingAndSavingsBalance, err := h.accountRepo.GetBalanceSumBySubtype(r.Context(), userID, []string{"CHECKING_ACCOUNT", "SAVINGS_ACCOUNT"})
	if err != nil {
		log.Printf("Error calculating checking and savings account balance for user %d: %v", userID, err)
		checkingAndSavingsBalance = 0
	}

	creditCardBalance, err := h.accountRepo.GetBalanceSumBySubtype(r.Context(), userID, []string{"CREDIT_CARD"})
	if err != nil {
		log.Printf("Error calculating credit card balance for user %d: %v", userID, err)
		creditCardBalance = 0
	}

	balanceTotal := checkingAndSavingsBalance - creditCardBalance
	user.BalanceTotal = &balanceTotal

	// Check if provider_key exists (not null/empty)
	hasValidKey := user.ProviderKey != nil && *user.ProviderKey != ""

	response := UserResponse{
		User:        user,
		HasValidKey: hasValidKey,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *UserHandler) handleUpdateMe(w http.ResponseWriter, r *http.Request, userID int64) {
	var params user.UpdateUserParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Printf("Error decoding user update request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// If provider_key is being updated, verify it and fetch accounts in one call
	if params.ProviderKey != nil && *params.ProviderKey != "" {
		// Fetch accounts - this verifies the key AND gets the data we need
		accountResp, statusCode, err := h.ofClient.GetAccountsWithStatus(r.Context(), *params.ProviderKey)
		if err != nil {
			// Check if it's a 401 (unauthorized)
			if statusCode == http.StatusUnauthorized {
				log.Printf("Invalid provider key for user %d: OpenFinance returned 401", userID)
				http.Error(w, "Invalid provider key", http.StatusUnauthorized)
				return
			}
			log.Printf("Error fetching accounts for user %d (status %d): %v", userID, statusCode, err)
			http.Error(w, "Failed to verify provider key", http.StatusBadGateway)
			return
		}

		// Status code is 200, accountResp is parsed and ready to use
		// Save the user first
		updatedUser, err := h.userRepo.Update(r.Context(), userID, params)
		if err != nil {
			log.Printf("Error updating user %d: %v", userID, err)
			http.Error(w, "Failed to update user", http.StatusInternalServerError)
			return
		}

		// Return 202 Accepted immediately with the updated user
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(updatedUser)

		// Use the already-fetched account data to sync in background
		// This avoids making another API call - we parse AND use the data concurrently
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			log.Printf("Starting account sync for user %d using pre-fetched data", userID)
			accountResult, err := h.accountSyncService.SyncUserAccountsWithData(ctx, userID, accountResp)
			if err != nil {
				log.Printf("Error syncing accounts for user %d: %v", userID, err)
				return
			}
			log.Printf("Account sync completed for user %d: created=%d, updated=%d", userID, accountResult.Created, accountResult.Updated)

			// New key insertion — always fetch full history
			log.Printf("Starting transaction sync for user %d after new key insertion (full history)", userID)
			txResult, err := h.transactionSyncService.SyncUserTransactions(ctx, userID, true)
			if err != nil {
				log.Printf("Error syncing transactions for user %d: %v", userID, err)
				return
			}
			log.Printf("Transaction sync completed for user %d: created=%d, updated=%d", userID, txResult.Created, txResult.Updated)

			// After transaction sync, sync bills
			log.Printf("Starting bill sync for user %d after transaction sync", userID)
			billResult, err := h.billSyncService.SyncUserBills(ctx, userID)
			if err != nil {
				log.Printf("Error syncing bills for user %d: %v", userID, err)
				return
			}
			log.Printf("Bill sync completed for user %d: created=%d, updated=%d", userID, billResult.Created, billResult.Updated)

			if err := h.userRepo.SetHasFinishedOpenfinanceFlow(ctx, userID, true); err != nil {
				log.Printf("Error setting has_finished_openfinance_flow for user %d: %v", userID, err)
				return
			}

			h.notificationService.SendSyncComplete(ctx, userID, h.msgs)
		}()

		return
	}

	// No provider_key update, proceed with normal update
	updatedUser, err := h.userRepo.Update(r.Context(), userID, params)
	if err != nil {
		log.Printf("Error updating user %d: %v", userID, err)
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedUser)
}
