package http

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"parsa/internal/domain/user"
	"parsa/internal/infrastructure/postgres"
	"parsa/internal/shared/auth"
)

type AuthHandler struct {
	userRepo               *postgres.UserRepository
	oauthProvider          auth.OAuthProvider
	appleOAuthProvider     auth.OAuthProvider
	jwt                    *auth.JWT
	mobileCallbackURL      string
	webCallbackURL         string
	appleMobileCallbackURL string
	appleCallbackTemplate  *template.Template
	templateOnce           sync.Once
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

// SetAppleOAuthProvider sets the Apple OAuth provider (optional, called after construction)
func (h *AuthHandler) SetAppleOAuthProvider(provider auth.OAuthProvider, mobileCallbackURL string) {
	h.appleOAuthProvider = provider
	h.appleMobileCallbackURL = mobileCallbackURL
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

// HandleAppleMobileAuthStart generates Apple OAuth URL for mobile app
func (h *AuthHandler) HandleAppleMobileAuthStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.appleOAuthProvider == nil {
		http.Error(w, "Apple OAuth not configured", http.StatusServiceUnavailable)
		return
	}

	state, err := generateState()
	if err != nil {
		log.Printf("Error generating OAuth state: %v", err)
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}

	authURL := h.appleOAuthProvider.GetAuthURL(state, h.appleMobileCallbackURL)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthURLResponse{URL: authURL})
}

// HandleAppleMobileAuthCallback processes Apple OAuth callback for mobile
// Apple uses form_post response mode, so this handles POST requests
func (h *AuthHandler) HandleAppleMobileAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.appleOAuthProvider == nil {
		http.Error(w, "Apple OAuth not configured", http.StatusServiceUnavailable)
		return
	}

	// Parse form data (Apple sends as application/x-www-form-urlencoded)
	if err := r.ParseForm(); err != nil {
		log.Printf("Apple OAuth: Failed to parse form: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_form"})
		return
	}

	code := r.FormValue("code")
	oauthError := r.FormValue("error")
	userJSON := r.FormValue("user") // Only sent on first authorization

	if oauthError != "" {
		log.Printf("Apple OAuth error: %s", oauthError)
		// Render HTML redirect page with error
		h.renderAppleCallbackPage(w, r, "", oauthError)
		return
	}

	if code == "" {
		log.Printf("Apple OAuth: No code received")
		h.renderAppleCallbackPage(w, r, "", "code_required")
		return
	}

	ctx := r.Context()

	// Exchange code for token
	token, err := h.appleOAuthProvider.ExchangeCode(ctx, code, h.appleMobileCallbackURL)
	if err != nil {
		log.Printf("Apple OAuth: Failed to exchange code: %v", err)
		h.renderAppleCallbackPage(w, r, "", "token_exchange_failed")
		return
	}

	// Get user info from id_token (stored in AccessToken field)
	userInfo, err := h.appleOAuthProvider.GetUserInfo(ctx, token.AccessToken)
	if err != nil {
		log.Printf("Apple OAuth: Failed to get user info: %v", err)
		h.renderAppleCallbackPage(w, r, "", "user_info_failed")
		return
	}

	// Parse user object if provided (only on first authorization)
	var appleUser auth.AppleUserInfo
	if userJSON != "" {
		if err := json.Unmarshal([]byte(userJSON), &appleUser); err != nil {
			log.Printf("Apple OAuth: Failed to parse user JSON: %v", err)
			// Non-fatal - continue without name
		} else {
			userInfo.FirstName = appleUser.Name.FirstName
			userInfo.LastName = appleUser.Name.LastName
			if userInfo.FirstName != "" || userInfo.LastName != "" {
				userInfo.Name = strings.TrimSpace(appleUser.Name.FirstName + " " + appleUser.Name.LastName)
			}
		}
	}

	// Find or create user
	userModel, err := h.userRepo.GetByOAuth(ctx, "apple", userInfo.ID)
	if err != nil {
		// User doesn't exist, create new user
		provider := "apple"
		userModel, err = h.userRepo.Create(ctx, user.CreateUserParams{
			Email:         userInfo.Email,
			Name:          userInfo.Name,
			OAuthProvider: &provider,
			OAuthID:       &userInfo.ID,
			FirstName:     userInfo.FirstName,
			LastName:      userInfo.LastName,
			// Apple doesn't provide avatar
		})
		if err != nil {
			log.Printf("Apple OAuth: Failed to create user: %v", err)
			h.renderAppleCallbackPage(w, r, "", "user_creation_failed")
			return
		}
	}

	// Generate JWT
	jwtToken, err := h.jwt.Generate(userModel.ID, userModel.Email)
	if err != nil {
		log.Printf("Apple OAuth: Error generating JWT for user %d: %v", userModel.ID, err)
		h.renderAppleCallbackPage(w, r, "", "jwt_generation_failed")
		return
	}

	// Render HTML redirect page with token
	h.renderAppleCallbackPage(w, r, jwtToken, "")
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

// renderAppleCallbackPage renders the HTML redirect page for Apple OAuth callback
func (h *AuthHandler) renderAppleCallbackPage(w http.ResponseWriter, r *http.Request, token, errorMsg string) {
	// Thread-safe one-time template initialization
	var templateErr error
	h.templateOnce.Do(func() {
		tmplPath := filepath.Join("web", "apple-oauth-callback.html")
		h.appleCallbackTemplate, templateErr = template.ParseFiles(tmplPath)
		if templateErr != nil {
			log.Printf("Apple OAuth: Failed to load callback template: %v", templateErr)
		}
	})

	// If template failed to load, use fallback redirect
	if h.appleCallbackTemplate == nil {
		if errorMsg != "" {
			redirectURL := fmt.Sprintf("com.parsa.app://oauth-callback?error=%s", errorMsg)
			http.Redirect(w, r, redirectURL, http.StatusFound)
		} else if token != "" {
			redirectURL := fmt.Sprintf("com.parsa.app://oauth-callback?token=%s", token)
			http.Redirect(w, r, redirectURL, http.StatusFound)
		} else {
			http.Error(w, "OAuth callback failed", http.StatusInternalServerError)
		}
		return
	}

	// Template data - encode as JSON for safe embedding in JavaScript
	type templateData struct {
		TokenJSON template.JS
		ErrorJSON template.JS
	}

	var tokenJSON, errorJSON template.JS
	if token != "" {
		tokenBytes, _ := json.Marshal(token)
		tokenJSON = template.JS(tokenBytes)
	} else {
		tokenJSON = template.JS("null")
	}
	if errorMsg != "" {
		errorBytes, _ := json.Marshal(errorMsg)
		errorJSON = template.JS(errorBytes)
	} else {
		errorJSON = template.JS("null")
	}

	data := templateData{
		TokenJSON: tokenJSON,
		ErrorJSON: errorJSON,
	}

	// Set content type and render
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.appleCallbackTemplate.Execute(w, data); err != nil {
		log.Printf("Apple OAuth: Failed to render callback template: %v", err)
		// Fallback redirect
		if errorMsg != "" {
			redirectURL := fmt.Sprintf("com.parsa.app://oauth-callback?error=%s", errorMsg)
			http.Redirect(w, r, redirectURL, http.StatusFound)
		} else if token != "" {
			redirectURL := fmt.Sprintf("com.parsa.app://oauth-callback?token=%s", token)
			http.Redirect(w, r, redirectURL, http.StatusFound)
		}
	}
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
