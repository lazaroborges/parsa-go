package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"parsa/internal/auth"
	"parsa/internal/database"
	"parsa/internal/models"
)

type AuthHandler struct {
	userRepo      *database.UserRepository
	oauthProvider auth.OAuthProvider
	jwt           *auth.JWT
}

func NewAuthHandler(userRepo *database.UserRepository, oauthProvider auth.OAuthProvider, jwt *auth.JWT) *AuthHandler {
	return &AuthHandler{
		userRepo:      userRepo,
		oauthProvider: oauthProvider,
		jwt:           jwt,
	}
}

type AuthURLResponse struct {
	URL string `json:"url"`
}

type AuthCallbackRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

// HandleAuthURL generates the OAuth authorization URL
func (h *AuthHandler) HandleAuthURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	state, err := generateState()
	if err != nil {
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}

	authURL := h.oauthProvider.GetAuthURL(state)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthURLResponse{URL: authURL})
}

// HandleCallback processes the OAuth callback and issues a JWT
func (h *AuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AuthCallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Code == "" {
		http.Error(w, "Code is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Exchange code for token
	token, err := h.oauthProvider.ExchangeCode(ctx, req.Code)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to exchange code: %v", err), http.StatusBadRequest)
		return
	}

	// Get user info from OAuth provider
	userInfo, err := h.oauthProvider.GetUserInfo(ctx, token.AccessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user info: %v", err), http.StatusBadRequest)
		return
	}

	// Find or create user
	user, err := h.userRepo.GetByOAuth(ctx, "google", userInfo.ID)
	if err != nil {
		// User doesn't exist, create new user
		user, err = h.userRepo.Create(ctx, models.CreateUserParams{
			Email:         userInfo.Email,
			Name:          userInfo.Name,
			OAuthProvider: "google",
			OAuthID:       userInfo.ID,
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create user: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Generate JWT
	jwtToken, err := h.jwt.Generate(user.ID, user.Email)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		Token: jwtToken,
		User:  user,
	})
}

func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
