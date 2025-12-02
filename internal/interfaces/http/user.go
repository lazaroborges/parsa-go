package http

import (
	"encoding/json"
	"log"
	"net/http"

	"parsa/internal/domain/account"
	"parsa/internal/domain/user"
	"parsa/internal/shared/middleware"
)

type UserHandler struct {
	userRepo    user.Repository
	accountRepo account.Repository
}

func NewUserHandler(userRepo user.Repository, accountRepo account.Repository) *UserHandler {
	return &UserHandler{
		userRepo:    userRepo,
		accountRepo: accountRepo,
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) handleUpdateMe(w http.ResponseWriter, r *http.Request, userID int64) {
	var params user.UpdateUserParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Printf("Error decoding user update request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.Update(r.Context(), userID, params)
	if err != nil {
		log.Printf("Error updating user %d: %v", userID, err)
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
