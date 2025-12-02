package http

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"parsa/internal/domain/user"
	"parsa/internal/infrastructure/postgres"
	"parsa/internal/shared/auth"
)

type AuthHandler struct {
	userRepo          *postgres.UserRepository
	oauthProvider     auth.OAuthProvider
	jwt               *auth.JWT
	mobileCallbackURL string
	webCallbackURL    string
}

func NewAuthHandler(userRepo *postgres.UserRepository, oauthProvider auth.OAuthProvider, jwt *auth.JWT, mobileCallbackURL, webCallbackURL string) *AuthHandler {
	return &AuthHandler{
		userRepo:          userRepo,
		oauthProvider:     oauthProvider,
		jwt:               jwt,
		mobileCallbackURL: mobileCallbackURL,
		webCallbackURL:    webCallbackURL,
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
	Token string     `json:"token"`
	User  *user.User `json:"user"`
}

// HandleAuthURL generates the OAuth authorization URL (for web)
func (h *AuthHandler) HandleAuthURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	state, err := generateState()
	if err != nil {
		log.Printf("Error generating OAuth state: %v", err)
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}

	authURL := h.oauthProvider.GetAuthURL(state)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthURLResponse{URL: authURL})
}

// HandleMobileAuthStart generates OAuth URL for mobile app
func (h *AuthHandler) HandleMobileAuthStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	state, err := generateState()
	if err != nil {
		log.Printf("Error generating OAuth state: %v", err)
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}

	// Generate OAuth URL with mobile callback URL
	authURL := h.oauthProvider.GetAuthURL(state, h.mobileCallbackURL)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthURLResponse{URL: authURL})
}

func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// HandleCallback processes the OAuth callback for web (issues a JWT and sets cookie)
func (h *AuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := r.URL.Query().Get("code")
	oauthError := r.URL.Query().Get("error")

	if oauthError != "" {
		http.Error(w, fmt.Sprintf("OAuth error: %s", oauthError), http.StatusBadRequest)
		return
	}

	if code == "" {
		http.Error(w, "Code is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Exchange code for token
	token, err := h.oauthProvider.ExchangeCode(ctx, code, h.webCallbackURL)
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
	userModel, err := h.userRepo.GetByOAuth(ctx, "google", userInfo.ID)
	if err != nil {
		// User doesn't exist, create new user
		provider := "google"
		userModel, err = h.userRepo.Create(ctx, user.CreateUserParams{
			Email:         userInfo.Email,
			Name:          userInfo.Name,
			OAuthProvider: &provider,
			OAuthID:       &userInfo.ID,
			FirstName:     userInfo.FirstName,
			LastName:      userInfo.LastName,
			AvatarURL:     &userInfo.AvatarURL,
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create user: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Generate JWT
	jwtToken, err := h.jwt.Generate(userModel.ID, userModel.Email)
	if err != nil {
		log.Printf("Error generating JWT for user %d: %v", userModel.ID, err)
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Set HttpOnly cookie with JWT
	setAuthCookie(w, r, jwtToken)

	// Redirect to callback page
	http.Redirect(w, r, "/oauth-callback", http.StatusFound)
}

// HandleMobileAuthCallback processes OAuth callback for mobile (returns JSON with JWT)
func (h *AuthHandler) HandleMobileAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := r.URL.Query().Get("code")
	oauthError := r.URL.Query().Get("error")

	if oauthError != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": oauthError})
		return
	}

	if code == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "code_required"})
		return
	}

	ctx := r.Context()

	// Exchange code for token (using mobile callback URL)
	token, err := h.oauthProvider.ExchangeCode(ctx, code, h.mobileCallbackURL)
	if err != nil {
		log.Printf("Mobile OAuth: Failed to exchange code: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "token_exchange_failed"})
		return
	}

	// Get user info from OAuth provider
	userInfo, err := h.oauthProvider.GetUserInfo(ctx, token.AccessToken)
	if err != nil {
		log.Printf("Mobile OAuth: Failed to get user info: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "user_info_failed"})
		return
	}

	// Find or create user
	userModel, err := h.userRepo.GetByOAuth(ctx, "google", userInfo.ID)
	if err != nil {
		// User doesn't exist, create new user
		provider := "google"
		userModel, err = h.userRepo.Create(ctx, user.CreateUserParams{
			Email:         userInfo.Email,
			Name:          userInfo.Name,
			OAuthProvider: &provider,
			OAuthID:       &userInfo.ID,
			FirstName:     userInfo.FirstName,
			LastName:      userInfo.LastName,
			AvatarURL:     &userInfo.AvatarURL,
		})
		if err != nil {
			log.Printf("Mobile OAuth: Failed to create user: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "user_creation_failed"})
			return
		}
	}

	// Generate JWT
	jwtToken, err := h.jwt.Generate(userModel.ID, userModel.Email)
	if err != nil {
		log.Printf("Mobile OAuth: Error generating JWT for user %d: %v", userModel.ID, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "jwt_generation_failed"})
		return
	}

	// Redirect to mobile app with token
	redirectURL := fmt.Sprintf("com.parsa.app://oauth-callback?token=%s", jwtToken)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// HandleRegister creates a new user with password authentication
func (h *AuthHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" || req.Name == "" {
		http.Error(w, "Email, password, and name are required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Check if user already exists
	existingUser, err := h.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		http.Error(w, "User with this email already exists", http.StatusConflict)
		return
	}
	// Note: If err is a "not found" type error, we proceed with creation.
	// Other errors (e.g., database failures) should ideally be handled.

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("Error hashing password during registration: %v", err)
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Create user
	userModel, err := h.userRepo.Create(ctx, user.CreateUserParams{
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: &passwordHash,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create user: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate JWT
	token, err := h.jwt.Generate(userModel.ID, userModel.Email)
	if err != nil {
		log.Printf("Error generating JWT for new user %d: %v", userModel.ID, err)
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	setAuthCookie(w, r, token)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		Token: token,
		User:  userModel,
	})
}

// HandleLogin authenticates a user with email and password
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Get user by email
	userModel, err := h.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// Check if user has password authentication
	if userModel.PasswordHash == nil {
		http.Error(w, "This account uses OAuth authentication. Please sign in with Google.", http.StatusBadRequest)
		return
	}

	// Verify password
	if err := auth.VerifyPassword(*userModel.PasswordHash, req.Password); err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// Generate JWT
	token, err := h.jwt.Generate(userModel.ID, userModel.Email)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	setAuthCookie(w, r, token)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		Token: token,
		User:  userModel,
	})
}

// HandleLogout clears the auth cookie
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Only set Secure flag when actually using HTTPS
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

	// Clear the cookie by setting MaxAge to -1
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	w.WriteHeader(http.StatusNoContent)
}

// setAuthCookie sets the JWT as an HttpOnly cookie
func setAuthCookie(w http.ResponseWriter, r *http.Request, token string) {
	// Only set Secure flag when actually using HTTPS
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400, // 24 hours (matches JWT expiration)
	})
}
