package models

import (
	"time"
)

type User struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	OAuthProvider *string   `json:"oauth_provider,omitempty"` // Nullable for password users
	OAuthID       *string   `json:"-"`                        // Don't expose OAuth ID in JSON
	PasswordHash  *string   `json:"-"`                        // Don't expose password hash in JSON
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateUserParams struct {
	Email         string
	Name          string
	OAuthProvider *string
	OAuthID       *string
	PasswordHash  *string
}
