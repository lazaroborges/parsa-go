package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
