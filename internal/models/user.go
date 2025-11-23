package models

import (
	"time"
)

type User struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	FirstName     string    `json:"firstName"`
	LastName      string    `json:"lastName"`
	OAuthProvider *string   `json:"oauthProvider,omitempty"` // Nullable for password users
	OAuthID       *string   `json:"-"`                       // Don't expose OAuth ID in JSON
	PasswordHash  *string   `json:"-"`
	AvatarURL     *string   `json:"avatarUrl,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	ProviderKey   string    `json:"providerKey"`
}

type CreateUserParams struct {
	Email         string
	Name          string
	OAuthProvider *string
	OAuthID       *string
	PasswordHash  *string
	FirstName     string
	LastName      string
	AvatarURL     *string
}

type UpdateUserParams struct {
	Name        *string
	FirstName   *string
	LastName    *string
	AvatarURL   *string
	ProviderKey *string
}
