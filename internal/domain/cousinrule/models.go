package cousinrule

import (
	"errors"
	"time"
)

var (
	ErrRuleNotFound = errors.New("cousin rule not found")
	ErrForbidden    = errors.New("forbidden: rule does not belong to user")
	ErrNoChanges    = errors.New("no changes provided")
)

// CousinRule represents a user's rule for transactions with a specific cousin (merchant/counterparty)
type CousinRule struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"-"`
	CousinID     int64     `json:"cousinId"`
	Type         *string   `json:"type,omitempty"`        // "DEBIT" or "CREDIT", nil matches both
	Category     *string   `json:"category,omitempty"`    // Category to apply
	Description  *string   `json:"description,omitempty"` // Description to apply
	Notes        *string   `json:"notes,omitempty"`       // Notes to apply
	Considered   *bool     `json:"considered,omitempty"`  // nil = no rule, true/false = apply value
	DontAskAgain bool      `json:"dontAskAgain"`
	Tags         []string  `json:"tags"` // Tag IDs to apply
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// CreateCousinRuleParams contains the parameters for creating a cousin rule
type CreateCousinRuleParams struct {
	UserID       int64
	CousinID     int64
	Type         *string  // "DEBIT" or "CREDIT", nil matches both
	Category     *string
	Description  *string
	Notes        *string
	Considered   *bool
	DontAskAgain bool
	Tags         []string
}

// Validate validates the create parameters
func (p *CreateCousinRuleParams) Validate() error {
	if p.CousinID == 0 {
		return errors.New("cousin_id is required")
	}
	if p.Type != nil && *p.Type != "DEBIT" && *p.Type != "CREDIT" {
		return errors.New("type must be DEBIT or CREDIT")
	}
	return nil
}

// UpdateCousinRuleParams contains the parameters for updating a cousin rule
type UpdateCousinRuleParams struct {
	Category     *string
	Description  *string
	Notes        *string
	Considered   *bool
	DontAskAgain *bool
	Tags         *[]string // nil = don't update, empty slice = clear all tags
}

// ApplyRuleParams contains the parameters for applying a rule to transactions
type ApplyRuleParams struct {
	CousinID             int64
	TriggeringID         string   // Transaction ID that triggered this action
	CreateRule           bool     // Whether to persist the rule for future transactions
	DontAskAgain         bool     // Mark rule as "don't ask again"
	Changes              Changes  // Changes to apply
}

// Changes represents the modifications to apply to transactions
type Changes struct {
	Category    *string  `json:"category,omitempty"`
	Description *string  `json:"description,omitempty"`
	Notes       *string  `json:"notes,omitempty"`
	Considered  *bool    `json:"considered,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// IsEmpty returns true if no changes are specified
func (c *Changes) IsEmpty() bool {
	return c.Category == nil && c.Description == nil && c.Notes == nil && c.Considered == nil && len(c.Tags) == 0
}

// ApplyRuleResult contains the result of applying a rule
type ApplyRuleResult struct {
	TransactionsUpdated int  `json:"transactionsUpdated"`
	RuleCreated         bool `json:"ruleCreated"`
	RuleUpdated         bool `json:"ruleUpdated"`
}
