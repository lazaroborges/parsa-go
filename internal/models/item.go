package models

import (
	"time"
)

// Item represents a connection/relationship with a financial institution via the provider.
// One Item can have multiple Accounts (e.g., checking + credit card from same bank).
type Item struct {
	ID        string    `json:"id"` // Provider's itemId (UUID string)
	UserID    int64     `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
