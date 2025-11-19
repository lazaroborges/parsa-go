package handlers

import (
	"encoding/json"
	"net/http"

	"parsa/internal/database"
	"parsa/internal/middleware"
)

type UserHandler struct {
	userRepo *database.UserRepository
}

func NewUserHandler(userRepo *database.UserRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

// HandleGetMe returns the current authenticated user
func (h *UserHandler) HandleGetMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
