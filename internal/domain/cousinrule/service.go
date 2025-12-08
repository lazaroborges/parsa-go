package cousinrule

import (
	"context"
	"fmt"

	"parsa/internal/domain/transaction"
)

// Service handles business logic for cousin rules
type Service struct {
	repo            Repository
	transactionRepo transaction.Repository
}

// NewService creates a new cousin rule service
func NewService(repo Repository, transactionRepo transaction.Repository) *Service {
	return &Service{
		repo:            repo,
		transactionRepo: transactionRepo,
	}
}

// ApplyRule applies changes to transactions and optionally creates/updates a rule
func (s *Service) ApplyRule(ctx context.Context, userID int64, params ApplyRuleParams) (*ApplyRuleResult, error) {
	result := &ApplyRuleResult{}

	// Get the triggering transaction to determine its type
	var txType *string
	if params.TriggeringID != "" {
		txn, err := s.transactionRepo.GetByID(ctx, params.TriggeringID)
		if err != nil {
			return nil, fmt.Errorf("failed to get triggering transaction: %w", err)
		}
		if txn != nil {
			txType = &txn.Type
		}
	}

	// Handle "don't ask again" without changes
	if params.Changes.IsEmpty() {
		if params.DontAskAgain {
			// Create or update rule with just dont_ask_again flag
			_, wasCreated, err := s.repo.Upsert(ctx, CreateCousinRuleParams{
				UserID:       userID,
				CousinID:     params.CousinID,
				Type:         txType,
				DontAskAgain: true,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to set dont_ask_again: %w", err)
			}
			result.RuleCreated = wasCreated
			result.RuleUpdated = !wasCreated
			return result, nil
		}
		return nil, ErrNoChanges
	}

	// Apply changes to existing transactions with this cousin
	count, err := s.repo.ApplyRuleToTransactions(ctx, userID, params.CousinID, txType, params.Changes)
	if err != nil {
		return nil, fmt.Errorf("failed to apply rule to transactions: %w", err)
	}
	result.TransactionsUpdated = count

	// If tags were specified, apply them to transactions
	if len(params.Changes.Tags) > 0 {
		// Tags are applied within ApplyRuleToTransactions
	}

	// Create or update rule if requested
	if params.CreateRule {
		ruleParams := CreateCousinRuleParams{
			UserID:       userID,
			CousinID:     params.CousinID,
			Type:         txType,
			Category:     params.Changes.Category,
			Description:  params.Changes.Description,
			Notes:        params.Changes.Notes,
			Considered:   params.Changes.Considered,
			DontAskAgain: params.DontAskAgain,
			Tags:         params.Changes.Tags,
		}

		rule, wasCreated, err := s.repo.Upsert(ctx, ruleParams)
		if err != nil {
			return nil, fmt.Errorf("failed to create/update rule: %w", err)
		}

		// Set tags for the rule
		if len(params.Changes.Tags) > 0 {
			if err := s.repo.SetRuleTags(ctx, rule.ID, params.Changes.Tags); err != nil {
				return nil, fmt.Errorf("failed to set rule tags: %w", err)
			}
		}

		result.RuleCreated = wasCreated
		result.RuleUpdated = !wasCreated
	}

	return result, nil
}

// GetRule returns a cousin rule by ID, verifying ownership
func (s *Service) GetRule(ctx context.Context, ruleID, userID int64) (*CousinRule, error) {
	rule, err := s.repo.GetByID(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, ErrRuleNotFound
	}
	if rule.UserID != userID {
		return nil, ErrForbidden
	}

	// Load tags
	tags, err := s.repo.GetRuleTags(ctx, rule.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule tags: %w", err)
	}
	rule.Tags = tags

	return rule, nil
}

// ListRulesByUser returns all cousin rules for a user
func (s *Service) ListRulesByUser(ctx context.Context, userID int64) ([]*CousinRule, error) {
	rules, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Load tags for each rule
	for _, rule := range rules {
		tags, err := s.repo.GetRuleTags(ctx, rule.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get rule tags: %w", err)
		}
		rule.Tags = tags
	}

	return rules, nil
}

// GetRuleForCousin returns the rule for a specific cousin and transaction type
func (s *Service) GetRuleForCousin(ctx context.Context, userID, cousinID int64, txType string) (*CousinRule, error) {
	// First try to find a type-specific rule
	rule, err := s.repo.GetByCousinAndType(ctx, userID, cousinID, &txType)
	if err != nil {
		return nil, err
	}

	// If no type-specific rule, try to find a rule that applies to all types
	if rule == nil {
		rule, err = s.repo.GetByCousinAndType(ctx, userID, cousinID, nil)
		if err != nil {
			return nil, err
		}
	}

	if rule == nil {
		return nil, nil
	}

	// Load tags
	tags, err := s.repo.GetRuleTags(ctx, rule.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule tags: %w", err)
	}
	rule.Tags = tags

	return rule, nil
}

// CheckDontAskAgain checks if user has set "don't ask again" for a cousin/type
func (s *Service) CheckDontAskAgain(ctx context.Context, userID, cousinID int64, txType string) (bool, error) {
	return s.repo.CheckDontAskAgain(ctx, userID, cousinID, txType)
}

// DeleteRule deletes a cousin rule, verifying ownership
func (s *Service) DeleteRule(ctx context.Context, ruleID, userID int64) error {
	rule, err := s.repo.GetByID(ctx, ruleID)
	if err != nil {
		return err
	}
	if rule == nil {
		return ErrRuleNotFound
	}
	if rule.UserID != userID {
		return ErrForbidden
	}

	return s.repo.Delete(ctx, ruleID)
}
