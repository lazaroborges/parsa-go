package cousinrule

import (
	"context"
)

// Repository defines the interface for cousin rule data access
type Repository interface {
	// Create creates a new cousin rule
	Create(ctx context.Context, params CreateCousinRuleParams) (*CousinRule, error)

	// GetByID returns a cousin rule by its ID
	GetByID(ctx context.Context, id int64) (*CousinRule, error)

	// GetByCousinAndType returns a rule for a specific user, cousin, and type combination
	// Type can be nil to match rules that apply to all transaction types
	GetByCousinAndType(ctx context.Context, userID, cousinID int64, txType *string) (*CousinRule, error)

	// ListByUserID returns all cousin rules for a user
	ListByUserID(ctx context.Context, userID int64) ([]*CousinRule, error)

	// ListByCousinID returns all rules for a specific cousin across a user's rules
	ListByCousinID(ctx context.Context, userID, cousinID int64) ([]*CousinRule, error)

	// Update updates a cousin rule
	Update(ctx context.Context, id int64, params UpdateCousinRuleParams) (*CousinRule, error)

	// Upsert creates or updates a cousin rule based on user_id, cousin_id, and type
	Upsert(ctx context.Context, params CreateCousinRuleParams) (*CousinRule, bool, error) // returns rule, wasCreated, error

	// Delete deletes a cousin rule
	Delete(ctx context.Context, id int64) error

	// SetRuleTags replaces all tags for a cousin rule
	SetRuleTags(ctx context.Context, ruleID int64, tagIDs []string) error

	// GetRuleTags returns all tag IDs for a cousin rule
	GetRuleTags(ctx context.Context, ruleID int64) ([]string, error)

	// ApplyRuleToTransactions applies a rule to all matching transactions
	// Returns the number of transactions updated
	ApplyRuleToTransactions(ctx context.Context, userID, cousinID int64, txType *string, changes Changes) (int, error)

	// CheckDontAskAgain checks if a user has marked "don't ask again" for a cousin/type combination
	CheckDontAskAgain(ctx context.Context, userID, cousinID int64, txType string) (bool, error)
}
