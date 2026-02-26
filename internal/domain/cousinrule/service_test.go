package cousinrule

import (
	"context"
	"errors"
	"testing"
	"time"

	"parsa/internal/domain/transaction"
)

// MockCousinRuleRepo implements Repository for testing
type MockCousinRuleRepo struct {
	CreateFunc                  func(ctx context.Context, params CreateCousinRuleParams) (*CousinRule, error)
	GetByIDFunc                 func(ctx context.Context, id int64) (*CousinRule, error)
	GetByCousinAndTypeFunc      func(ctx context.Context, userID, cousinID int64, txType *string) (*CousinRule, error)
	ListByUserIDFunc            func(ctx context.Context, userID int64) ([]*CousinRule, error)
	ListByCousinIDFunc          func(ctx context.Context, userID, cousinID int64) ([]*CousinRule, error)
	UpdateFunc                  func(ctx context.Context, id int64, params UpdateCousinRuleParams) (*CousinRule, error)
	UpsertFunc                  func(ctx context.Context, params CreateCousinRuleParams) (*CousinRule, bool, error)
	DeleteFunc                  func(ctx context.Context, id int64) error
	SetRuleTagsFunc             func(ctx context.Context, ruleID int64, tagIDs []string) error
	GetRuleTagsFunc             func(ctx context.Context, ruleID int64) ([]string, error)
	ApplyRuleToTransactionsFunc func(ctx context.Context, userID, cousinID int64, txType *string, changes Changes) (int, error)
	CheckDontAskAgainFunc       func(ctx context.Context, userID, cousinID int64, txType string) (bool, error)
}

func (m *MockCousinRuleRepo) Create(ctx context.Context, params CreateCousinRuleParams) (*CousinRule, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, params)
	}
	return nil, nil
}
func (m *MockCousinRuleRepo) GetByID(ctx context.Context, id int64) (*CousinRule, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *MockCousinRuleRepo) GetByCousinAndType(ctx context.Context, userID, cousinID int64, txType *string) (*CousinRule, error) {
	if m.GetByCousinAndTypeFunc != nil {
		return m.GetByCousinAndTypeFunc(ctx, userID, cousinID, txType)
	}
	return nil, nil
}
func (m *MockCousinRuleRepo) ListByUserID(ctx context.Context, userID int64) ([]*CousinRule, error) {
	if m.ListByUserIDFunc != nil {
		return m.ListByUserIDFunc(ctx, userID)
	}
	return nil, nil
}
func (m *MockCousinRuleRepo) ListByCousinID(ctx context.Context, userID, cousinID int64) ([]*CousinRule, error) {
	if m.ListByCousinIDFunc != nil {
		return m.ListByCousinIDFunc(ctx, userID, cousinID)
	}
	return nil, nil
}
func (m *MockCousinRuleRepo) Update(ctx context.Context, id int64, params UpdateCousinRuleParams) (*CousinRule, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, id, params)
	}
	return nil, nil
}
func (m *MockCousinRuleRepo) Upsert(ctx context.Context, params CreateCousinRuleParams) (*CousinRule, bool, error) {
	if m.UpsertFunc != nil {
		return m.UpsertFunc(ctx, params)
	}
	return nil, false, nil
}
func (m *MockCousinRuleRepo) Delete(ctx context.Context, id int64) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}
func (m *MockCousinRuleRepo) SetRuleTags(ctx context.Context, ruleID int64, tagIDs []string) error {
	if m.SetRuleTagsFunc != nil {
		return m.SetRuleTagsFunc(ctx, ruleID, tagIDs)
	}
	return nil
}
func (m *MockCousinRuleRepo) GetRuleTags(ctx context.Context, ruleID int64) ([]string, error) {
	if m.GetRuleTagsFunc != nil {
		return m.GetRuleTagsFunc(ctx, ruleID)
	}
	return []string{}, nil
}
func (m *MockCousinRuleRepo) ApplyRuleToTransactions(ctx context.Context, userID, cousinID int64, txType *string, changes Changes) (int, error) {
	if m.ApplyRuleToTransactionsFunc != nil {
		return m.ApplyRuleToTransactionsFunc(ctx, userID, cousinID, txType, changes)
	}
	return 0, nil
}
func (m *MockCousinRuleRepo) CheckDontAskAgain(ctx context.Context, userID, cousinID int64, txType string) (bool, error) {
	if m.CheckDontAskAgainFunc != nil {
		return m.CheckDontAskAgainFunc(ctx, userID, cousinID, txType)
	}
	return false, nil
}

// MockTransactionRepo implements transaction.Repository for testing
type MockTransactionRepo struct {
	GetByIDFunc func(ctx context.Context, id string) (*transaction.Transaction, error)
}

func (m *MockTransactionRepo) GetByID(ctx context.Context, id string) (*transaction.Transaction, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *MockTransactionRepo) Create(ctx context.Context, params transaction.CreateTransactionParams) (*transaction.Transaction, error) {
	return nil, nil
}
func (m *MockTransactionRepo) ListByAccountID(ctx context.Context, accountID string, limit, offset int) ([]*transaction.Transaction, error) {
	return nil, nil
}
func (m *MockTransactionRepo) ListByUserID(ctx context.Context, userID int64, limit, offset int) ([]*transaction.Transaction, error) {
	return nil, nil
}
func (m *MockTransactionRepo) CountByUserID(ctx context.Context, userID int64) (int64, error) {
	return 0, nil
}
func (m *MockTransactionRepo) Update(ctx context.Context, id string, params transaction.UpdateTransactionParams) (*transaction.Transaction, error) {
	return nil, nil
}
func (m *MockTransactionRepo) Delete(ctx context.Context, id string) error { return nil }
func (m *MockTransactionRepo) Upsert(ctx context.Context, params transaction.UpsertTransactionParams) (*transaction.Transaction, error) {
	return nil, nil
}
func (m *MockTransactionRepo) FindPotentialDuplicates(ctx context.Context, criteria transaction.DuplicateCriteria) ([]*transaction.Transaction, error) {
	return nil, nil
}
func (m *MockTransactionRepo) FindPotentialDuplicatesForBill(ctx context.Context, criteria transaction.DuplicateCriteria) ([]*transaction.Transaction, error) {
	return nil, nil
}
func (m *MockTransactionRepo) SetTransactionTags(ctx context.Context, transactionID string, tagIDs []string) error {
	return nil
}
func (m *MockTransactionRepo) GetTransactionTags(ctx context.Context, transactionID string) ([]string, error) {
	return nil, nil
}

func TestChanges_IsEmpty(t *testing.T) {
	tests := []struct {
		name    string
		changes Changes
		want    bool
	}{
		{"all nil", Changes{}, true},
		{"category set", Changes{Category: strPtr("Food")}, false},
		{"description set", Changes{Description: strPtr("desc")}, false},
		{"notes set", Changes{Notes: strPtr("note")}, false},
		{"considered set", Changes{Considered: boolPtr(true)}, false},
		{"tags set", Changes{Tags: &[]string{"tag1"}}, false},
		{"empty tags set", Changes{Tags: &[]string{}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.changes.IsEmpty()
			if got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateCousinRuleParams_Validate(t *testing.T) {
	debit := "DEBIT"
	invalid := "INVALID"

	tests := []struct {
		name    string
		params  CreateCousinRuleParams
		wantErr bool
	}{
		{
			name:    "valid with type",
			params:  CreateCousinRuleParams{CousinID: 1, Type: &debit},
			wantErr: false,
		},
		{
			name:    "valid without type",
			params:  CreateCousinRuleParams{CousinID: 1},
			wantErr: false,
		},
		{
			name:    "missing cousin ID",
			params:  CreateCousinRuleParams{CousinID: 0},
			wantErr: true,
		},
		{
			name:    "invalid type",
			params:  CreateCousinRuleParams{CousinID: 1, Type: &invalid},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()
			if tt.wantErr && err == nil {
				t.Error("Validate() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestApplyRule_EmptyChangesNoFlag(t *testing.T) {
	repo := &MockCousinRuleRepo{}
	txRepo := &MockTransactionRepo{}
	svc := NewService(repo, txRepo)

	_, err := svc.ApplyRule(context.Background(), 1, ApplyRuleParams{
		CousinID: 1,
		Changes:  Changes{},
	})
	if err != ErrNoChanges {
		t.Errorf("ApplyRule() error = %v, want ErrNoChanges", err)
	}
}

func TestApplyRule_DontAskAgainOnly(t *testing.T) {
	upsertCalled := false
	repo := &MockCousinRuleRepo{
		UpsertFunc: func(ctx context.Context, params CreateCousinRuleParams) (*CousinRule, bool, error) {
			upsertCalled = true
			if !params.DontAskAgain {
				t.Error("expected DontAskAgain=true")
			}
			return &CousinRule{ID: 1}, true, nil
		},
	}
	txRepo := &MockTransactionRepo{}
	svc := NewService(repo, txRepo)

	result, err := svc.ApplyRule(context.Background(), 1, ApplyRuleParams{
		CousinID:     1,
		DontAskAgain: true,
		Changes:      Changes{},
	})
	if err != nil {
		t.Fatalf("ApplyRule() error: %v", err)
	}
	if !upsertCalled {
		t.Error("expected Upsert to be called")
	}
	if !result.RuleCreated {
		t.Error("expected RuleCreated=true")
	}
}

func TestApplyRule_WithChangesAndCreateRule(t *testing.T) {
	repo := &MockCousinRuleRepo{
		ApplyRuleToTransactionsFunc: func(ctx context.Context, userID, cousinID int64, txType *string, changes Changes) (int, error) {
			return 5, nil
		},
		UpsertFunc: func(ctx context.Context, params CreateCousinRuleParams) (*CousinRule, bool, error) {
			return &CousinRule{ID: 1}, true, nil
		},
	}
	txRepo := &MockTransactionRepo{
		GetByIDFunc: func(ctx context.Context, id string) (*transaction.Transaction, error) {
			return &transaction.Transaction{ID: id, Type: "DEBIT"}, nil
		},
	}
	svc := NewService(repo, txRepo)

	result, err := svc.ApplyRule(context.Background(), 1, ApplyRuleParams{
		CousinID:     42,
		TriggeringID: "tx-1",
		CreateRule:   true,
		Changes:      Changes{Category: strPtr("Food")},
	})
	if err != nil {
		t.Fatalf("ApplyRule() error: %v", err)
	}
	if result.TransactionsUpdated != 5 {
		t.Errorf("TransactionsUpdated = %d, want 5", result.TransactionsUpdated)
	}
	if !result.RuleCreated {
		t.Error("expected RuleCreated=true")
	}
}

func TestGetRule_Success(t *testing.T) {
	repo := &MockCousinRuleRepo{
		GetByIDFunc: func(ctx context.Context, id int64) (*CousinRule, error) {
			return &CousinRule{ID: id, UserID: 1, CousinID: 42}, nil
		},
		GetRuleTagsFunc: func(ctx context.Context, ruleID int64) ([]string, error) {
			return []string{"tag1"}, nil
		},
	}
	txRepo := &MockTransactionRepo{}
	svc := NewService(repo, txRepo)

	rule, err := svc.GetRule(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("GetRule() error: %v", err)
	}
	if rule.ID != 1 {
		t.Errorf("rule.ID = %d, want 1", rule.ID)
	}
	if len(rule.Tags) != 1 {
		t.Errorf("rule.Tags length = %d, want 1", len(rule.Tags))
	}
}

func TestGetRule_NotFound(t *testing.T) {
	repo := &MockCousinRuleRepo{
		GetByIDFunc: func(ctx context.Context, id int64) (*CousinRule, error) {
			return nil, nil
		},
	}
	txRepo := &MockTransactionRepo{}
	svc := NewService(repo, txRepo)

	_, err := svc.GetRule(context.Background(), 1, 1)
	if err != ErrRuleNotFound {
		t.Errorf("GetRule() error = %v, want ErrRuleNotFound", err)
	}
}

func TestGetRule_Forbidden(t *testing.T) {
	repo := &MockCousinRuleRepo{
		GetByIDFunc: func(ctx context.Context, id int64) (*CousinRule, error) {
			return &CousinRule{ID: id, UserID: 2}, nil
		},
	}
	txRepo := &MockTransactionRepo{}
	svc := NewService(repo, txRepo)

	_, err := svc.GetRule(context.Background(), 1, 1)
	if err != ErrForbidden {
		t.Errorf("GetRule() error = %v, want ErrForbidden", err)
	}
}

func TestDeleteRule_Success(t *testing.T) {
	deleteCalled := false
	repo := &MockCousinRuleRepo{
		GetByIDFunc: func(ctx context.Context, id int64) (*CousinRule, error) {
			return &CousinRule{ID: id, UserID: 1}, nil
		},
		DeleteFunc: func(ctx context.Context, id int64) error {
			deleteCalled = true
			return nil
		},
	}
	txRepo := &MockTransactionRepo{}
	svc := NewService(repo, txRepo)

	err := svc.DeleteRule(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("DeleteRule() error: %v", err)
	}
	if !deleteCalled {
		t.Error("expected Delete to be called")
	}
}

func TestDeleteRule_NotFound(t *testing.T) {
	repo := &MockCousinRuleRepo{
		GetByIDFunc: func(ctx context.Context, id int64) (*CousinRule, error) {
			return nil, nil
		},
	}
	txRepo := &MockTransactionRepo{}
	svc := NewService(repo, txRepo)

	err := svc.DeleteRule(context.Background(), 1, 1)
	if err != ErrRuleNotFound {
		t.Errorf("DeleteRule() error = %v, want ErrRuleNotFound", err)
	}
}

func TestDeleteRule_Forbidden(t *testing.T) {
	repo := &MockCousinRuleRepo{
		GetByIDFunc: func(ctx context.Context, id int64) (*CousinRule, error) {
			return &CousinRule{ID: id, UserID: 2}, nil
		},
	}
	txRepo := &MockTransactionRepo{}
	svc := NewService(repo, txRepo)

	err := svc.DeleteRule(context.Background(), 1, 1)
	if err != ErrForbidden {
		t.Errorf("DeleteRule() error = %v, want ErrForbidden", err)
	}
}

func TestGetRuleForCousin_TypeSpecificRule(t *testing.T) {
	debit := "DEBIT"
	repo := &MockCousinRuleRepo{
		GetByCousinAndTypeFunc: func(ctx context.Context, userID, cousinID int64, txType *string) (*CousinRule, error) {
			if txType != nil && *txType == "DEBIT" {
				return &CousinRule{ID: 1, UserID: userID, Type: &debit}, nil
			}
			return nil, nil
		},
		GetRuleTagsFunc: func(ctx context.Context, ruleID int64) ([]string, error) {
			return []string{}, nil
		},
	}
	txRepo := &MockTransactionRepo{}
	svc := NewService(repo, txRepo)

	rule, err := svc.GetRuleForCousin(context.Background(), 1, 42, "DEBIT")
	if err != nil {
		t.Fatalf("GetRuleForCousin() error: %v", err)
	}
	if rule == nil {
		t.Fatal("GetRuleForCousin() returned nil")
	}
	if rule.ID != 1 {
		t.Errorf("rule.ID = %d, want 1", rule.ID)
	}
}

func TestGetRuleForCousin_FallbackToAllTypes(t *testing.T) {
	callCount := 0
	repo := &MockCousinRuleRepo{
		GetByCousinAndTypeFunc: func(ctx context.Context, userID, cousinID int64, txType *string) (*CousinRule, error) {
			callCount++
			if txType != nil {
				return nil, nil // No type-specific rule
			}
			return &CousinRule{ID: 2, UserID: userID}, nil // fallback rule
		},
		GetRuleTagsFunc: func(ctx context.Context, ruleID int64) ([]string, error) {
			return []string{}, nil
		},
	}
	txRepo := &MockTransactionRepo{}
	svc := NewService(repo, txRepo)

	rule, err := svc.GetRuleForCousin(context.Background(), 1, 42, "DEBIT")
	if err != nil {
		t.Fatalf("GetRuleForCousin() error: %v", err)
	}
	if rule == nil {
		t.Fatal("GetRuleForCousin() returned nil")
	}
	if callCount != 2 {
		t.Errorf("GetByCousinAndType called %d times, want 2 (type-specific + fallback)", callCount)
	}
	if rule.ID != 2 {
		t.Errorf("rule.ID = %d, want 2 (fallback rule)", rule.ID)
	}
}

func TestGetRuleForCousin_NoRule(t *testing.T) {
	repo := &MockCousinRuleRepo{
		GetByCousinAndTypeFunc: func(ctx context.Context, userID, cousinID int64, txType *string) (*CousinRule, error) {
			return nil, nil
		},
	}
	txRepo := &MockTransactionRepo{}
	svc := NewService(repo, txRepo)

	rule, err := svc.GetRuleForCousin(context.Background(), 1, 42, "DEBIT")
	if err != nil {
		t.Fatalf("GetRuleForCousin() error: %v", err)
	}
	if rule != nil {
		t.Error("GetRuleForCousin() expected nil rule")
	}
}

func TestListRulesByUser(t *testing.T) {
	repo := &MockCousinRuleRepo{
		ListByUserIDFunc: func(ctx context.Context, userID int64) ([]*CousinRule, error) {
			return []*CousinRule{
				{ID: 1, UserID: userID, CreatedAt: time.Now()},
				{ID: 2, UserID: userID, CreatedAt: time.Now()},
			}, nil
		},
		GetRuleTagsFunc: func(ctx context.Context, ruleID int64) ([]string, error) {
			return []string{}, nil
		},
	}
	txRepo := &MockTransactionRepo{}
	svc := NewService(repo, txRepo)

	rules, err := svc.ListRulesByUser(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListRulesByUser() error: %v", err)
	}
	if len(rules) != 2 {
		t.Errorf("ListRulesByUser() returned %d rules, want 2", len(rules))
	}
}

func TestListRulesByUser_RepoError(t *testing.T) {
	repo := &MockCousinRuleRepo{
		ListByUserIDFunc: func(ctx context.Context, userID int64) ([]*CousinRule, error) {
			return nil, errors.New("db error")
		},
	}
	txRepo := &MockTransactionRepo{}
	svc := NewService(repo, txRepo)

	_, err := svc.ListRulesByUser(context.Background(), 1)
	if err == nil {
		t.Error("ListRulesByUser() expected error, got nil")
	}
}

func TestCheckDontAskAgain(t *testing.T) {
	repo := &MockCousinRuleRepo{
		CheckDontAskAgainFunc: func(ctx context.Context, userID, cousinID int64, txType string) (bool, error) {
			return true, nil
		},
	}
	txRepo := &MockTransactionRepo{}
	svc := NewService(repo, txRepo)

	result, err := svc.CheckDontAskAgain(context.Background(), 1, 42, "DEBIT")
	if err != nil {
		t.Fatalf("CheckDontAskAgain() error: %v", err)
	}
	if !result {
		t.Error("CheckDontAskAgain() = false, want true")
	}
}

func strPtr(s string) *string  { return &s }
func boolPtr(b bool) *bool     { return &b }
