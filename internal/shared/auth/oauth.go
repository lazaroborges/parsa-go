package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type OAuthProvider interface {
	GetAuthURL(state string, redirectURI ...string) string
	ExchangeCode(ctx context.Context, code string, redirectURI ...string) (*OAuthToken, error)
	GetUserInfo(ctx context.Context, token string) (*OAuthUserInfo, error)
}

type OAuthToken struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type OAuthUserInfo struct {
	ID        string
	Email     string
	Name      string
	FirstName string
	LastName  string
	AvatarURL string
}

// GoogleOAuthProvider implements Google OAuth 2.0
type GoogleOAuthProvider struct {
	clientID     string
	clientSecret string
	redirectURL  string
	httpClient   *http.Client
}

func NewGoogleOAuthProvider(clientID, clientSecret, redirectURL string) *GoogleOAuthProvider {
	return &GoogleOAuthProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (g *GoogleOAuthProvider) GetAuthURL(state string, redirectURI ...string) string {
	baseURL := "https://accounts.google.com/o/oauth2/v2/auth"
	params := url.Values{}
	params.Add("client_id", g.clientID)

	// Use custom redirect URI if provided, otherwise use default
	targetRedirectURI := g.redirectURL
	if len(redirectURI) > 0 && redirectURI[0] != "" {
		targetRedirectURI = redirectURI[0]
	}
	params.Add("redirect_uri", targetRedirectURI)

	params.Add("response_type", "code")
	params.Add("scope", "openid email profile")
	params.Add("state", state)
	params.Add("access_type", "offline")

	return baseURL + "?" + params.Encode()
}

func (g *GoogleOAuthProvider) ExchangeCode(ctx context.Context, code string, redirectURI ...string) (*OAuthToken, error) {
	tokenURL := "https://oauth2.googleapis.com/token"

	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", g.clientID)
	data.Set("client_secret", g.clientSecret)

	// Use custom redirect URI if provided, otherwise use default
	targetRedirectURI := g.redirectURL
	if len(redirectURI) > 0 && redirectURI[0] != "" {
		targetRedirectURI = redirectURI[0]
	}
	data.Set("redirect_uri", targetRedirectURI)

	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var token OAuthToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &token, nil
}

func (g *GoogleOAuthProvider) GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	userInfoURL := "https://www.googleapis.com/oauth2/v2/userinfo"

	req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := g.httpClient.Do(req)

	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info: %s", string(body))
	}

	var googleUser struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		FirstName string `json:"given_name"`
		LastName  string `json:"family_name"`
		AvatarURL string `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &OAuthUserInfo{
		ID:        googleUser.ID,
		Email:     googleUser.Email,
		Name:      googleUser.Name,
		FirstName: googleUser.FirstName,
		LastName:  googleUser.LastName,
		AvatarURL: googleUser.AvatarURL,
	}, nil
}

// AppleOAuthProvider implements Apple Sign In OAuth 2.0
type AppleOAuthProvider struct {
	teamID         string
	keyID          string
	clientID       string
	privateKey     *ecdsa.PrivateKey
	redirectURL    string
	httpClient     *http.Client
}

func NewAppleOAuthProvider(teamID, keyID, clientID, privateKeyPath, redirectURL string) (*AppleOAuthProvider, error) {
	// Read private key file
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Apple private key: %w", err)
	}

	// Parse PEM block
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from Apple private key")
	}

	// Parse PKCS8 private key
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Apple private key: %w", err)
	}

	ecdsaKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("Apple private key is not an ECDSA key")
	}

	return &AppleOAuthProvider{
		teamID:      teamID,
		keyID:       keyID,
		clientID:    clientID,
		privateKey:  ecdsaKey,
		redirectURL: redirectURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// generateClientSecret creates a signed JWT to use as client_secret for Apple
func (a *AppleOAuthProvider) generateClientSecret() (string, error) {
	now := time.Now()
	exp := now.Add(5 * time.Minute) // Short-lived for security

	// Header
	header := map[string]string{
		"alg": "ES256",
		"kid": a.keyID,
	}
	headerJSON, _ := json.Marshal(header)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	// Claims
	claims := map[string]interface{}{
		"iss": a.teamID,
		"iat": now.Unix(),
		"exp": exp.Unix(),
		"aud": "https://appleid.apple.com",
		"sub": a.clientID,
	}
	claimsJSON, _ := json.Marshal(claims)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	// Sign
	message := headerB64 + "." + claimsB64
	signature, err := signES256([]byte(message), a.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign client secret: %w", err)
	}

	return message + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (a *AppleOAuthProvider) GetAuthURL(state string, redirectURI ...string) string {
	baseURL := "https://appleid.apple.com/auth/authorize"
	params := url.Values{}
	params.Add("client_id", a.clientID)

	targetRedirectURI := a.redirectURL
	if len(redirectURI) > 0 && redirectURI[0] != "" {
		targetRedirectURI = redirectURI[0]
	}
	params.Add("redirect_uri", targetRedirectURI)

	params.Add("response_type", "code")
	params.Add("response_mode", "form_post")
	params.Add("scope", "name email")
	params.Add("state", state)

	return baseURL + "?" + params.Encode()
}

func (a *AppleOAuthProvider) ExchangeCode(ctx context.Context, code string, redirectURI ...string) (*OAuthToken, error) {
	tokenURL := "https://appleid.apple.com/auth/token"

	clientSecret, err := a.generateClientSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate client secret: %w", err)
	}

	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", a.clientID)
	data.Set("client_secret", clientSecret)

	targetRedirectURI := a.redirectURL
	if len(redirectURI) > 0 && redirectURI[0] != "" {
		targetRedirectURI = redirectURI[0]
	}
	data.Set("redirect_uri", targetRedirectURI)

	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token,omitempty"`
		IDToken      string `json:"id_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &OAuthToken{
		AccessToken:  tokenResp.IDToken, // Store id_token as AccessToken for GetUserInfo
		TokenType:    tokenResp.TokenType,
		ExpiresIn:    tokenResp.ExpiresIn,
		RefreshToken: tokenResp.RefreshToken,
	}, nil
}

// GetUserInfo parses Apple's id_token to extract user info
// Note: Apple doesn't have a userinfo endpoint - all info is in the id_token
func (a *AppleOAuthProvider) GetUserInfo(ctx context.Context, idToken string) (*OAuthUserInfo, error) {
	// Parse id_token (JWT) - we trust Apple's token, no signature verification needed
	// since we just received it directly from Apple's token endpoint
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid id_token format")
	}

	// Decode claims (second part)
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Try with padding
		claimsJSON, err = base64.URLEncoding.DecodeString(parts[1] + "==")
		if err != nil {
			return nil, fmt.Errorf("failed to decode id_token claims: %w", err)
		}
	}

	var claims struct {
		Sub           string `json:"sub"`   // Unique user ID
		Email         string `json:"email"` // May be private relay email
		EmailVerified any    `json:"email_verified"`
	}
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse id_token claims: %w", err)
	}

	return &OAuthUserInfo{
		ID:    claims.Sub,
		Email: claims.Email,
		// Name fields will be populated from the user object in the callback
		// Apple only sends name on first authorization
	}, nil
}

// AppleUserInfo represents the user object Apple sends on first authorization
type AppleUserInfo struct {
	Name struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
	} `json:"name"`
	Email string `json:"email"`
}

// signES256 signs a message using ECDSA with SHA-256 (ES256 algorithm)
func signES256(message []byte, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	hash := sha256.Sum256(message)
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return nil, err
	}

	// ES256 signature is R || S, each padded to 32 bytes
	curveBits := privateKey.Curve.Params().BitSize
	keyBytes := (curveBits + 7) / 8

	signature := make([]byte, 2*keyBytes)
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	// Pad R and S to keyBytes length
	copy(signature[keyBytes-len(rBytes):keyBytes], rBytes)
	copy(signature[2*keyBytes-len(sBytes):], sBytes)

	return signature, nil
}

// verifyES256 verifies an ES256 signature (for Apple id_token verification if needed)
func verifyES256(message, signature []byte, publicKey *ecdsa.PublicKey) bool {
	if len(signature) != 64 {
		return false
	}

	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])

	hash := sha256.Sum256(message)
	return ecdsa.Verify(publicKey, hash[:], r, s)
}
