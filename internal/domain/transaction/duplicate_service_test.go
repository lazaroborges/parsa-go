package transaction

import (
	"context"
	"errors"
	"testing"
	"time"
)

type MockTransactionRepo struct {
	CreateFunc                         func(ctx context.Context, params CreateTransactionParams) (*Transaction, error)
	GetByIDFunc                        func(ctx context.Context, id string) (*Transaction, error)
	ListByAccountIDFunc                func(ctx context.Context, accountID string, limit, offset int) ([]*Transaction, error)
	ListByUserIDFunc                   func(ctx context.Context, userID int64, limit, offset int) ([]*Transaction, error)
	CountByUserIDFunc                  func(ctx context.Context, userID int64) (int64, error)
	UpdateFunc                         func(ctx context.Context, id string, params UpdateTransactionParams) (*Transaction, error)
	DeleteFunc                         func(ctx context.Context, id string) error
	UpsertFunc                         func(ctx context.Context, params UpsertTransactionParams) (*Transaction, error)
	FindPotentialDuplicatesFunc        func(ctx context.Context, criteria DuplicateCriteria) ([]*Transaction, error)
	FindPotentialDuplicatesForBillFunc func(ctx context.Context, criteria DuplicateCriteria) ([]*Transaction, error)
	SetTransactionTagsFunc             func(ctx context.Context, transactionID string, tagIDs []string) error
	GetTransactionTagsFunc             func(ctx context.Context, transactionID string) ([]string, error)
}

func (m *MockTransactionRepo) Create(ctx context.Context, params CreateTransactionParams) (*Transaction, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, params)
	}
	return nil, nil
}
func (m *MockTransactionRepo) GetByID(ctx context.Context, id string) (*Transaction, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *MockTransactionRepo) ListByAccountID(ctx context.Context, accountID string, limit, offset int) ([]*Transaction, error) {
	if m.ListByAccountIDFunc != nil {
		return m.ListByAccountIDFunc(ctx, accountID, limit, offset)
	}
	return nil, nil
}
func (m *MockTransactionRepo) ListByUserID(ctx context.Context, userID int64, limit, offset int) ([]*Transaction, error) {
	if m.ListByUserIDFunc != nil {
		return m.ListByUserIDFunc(ctx, userID, limit, offset)
	}
	return nil, nil
}
func (m *MockTransactionRepo) CountByUserID(ctx context.Context, userID int64) (int64, error) {
	if m.CountByUserIDFunc != nil {
		return m.CountByUserIDFunc(ctx, userID)
	}
	return 0, nil
}
func (m *MockTransactionRepo) Update(ctx context.Context, id string, params UpdateTransactionParams) (*Transaction, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, id, params)
	}
	return nil, nil
}
func (m *MockTransactionRepo) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}
func (m *MockTransactionRepo) Upsert(ctx context.Context, params UpsertTransactionParams) (*Transaction, error) {
	if m.UpsertFunc != nil {
		return m.UpsertFunc(ctx, params)
	}
	return nil, nil
}
func (m *MockTransactionRepo) FindPotentialDuplicates(ctx context.Context, criteria DuplicateCriteria) ([]*Transaction, error) {
	if m.FindPotentialDuplicatesFunc != nil {
		return m.FindPotentialDuplicatesFunc(ctx, criteria)
	}
	return nil, nil
}
func (m *MockTransactionRepo) FindPotentialDuplicatesForBill(ctx context.Context, criteria DuplicateCriteria) ([]*Transaction, error) {
	if m.FindPotentialDuplicatesForBillFunc != nil {
		return m.FindPotentialDuplicatesForBillFunc(ctx, criteria)
	}
	return nil, nil
}
func (m *MockTransactionRepo) SetTransactionTags(ctx context.Context, transactionID string, tagIDs []string) error {
	if m.SetTransactionTagsFunc != nil {
		return m.SetTransactionTagsFunc(ctx, transactionID, tagIDs)
	}
	return nil
}
func (m *MockTransactionRepo) GetTransactionTags(ctx context.Context, transactionID string) ([]string, error) {
	if m.GetTransactionTagsFunc != nil {
		return m.GetTransactionTagsFunc(ctx, transactionID)
	}
	return nil, nil
}

func TestNewDuplicateCheckService(t *testing.T) {
	repo := &MockTransactionRepo{}
	svc := NewDuplicateCheckService(repo)
	if svc == nil {
		t.Fatal("NewDuplicateCheckService() returned nil")
	}
	if svc.workerCount != DefaultWorkerCount {
		t.Errorf("workerCount = %d, want %d", svc.workerCount, DefaultWorkerCount)
	}
}

func TestNewDuplicateCheckServiceWithWorkers(t *testing.T) {
	repo := &MockTransactionRepo{}

	svc := NewDuplicateCheckServiceWithWorkers(repo, 8)
	if svc.workerCount != 8 {
		t.Errorf("workerCount = %d, want 8", svc.workerCount)
	}

	// Zero should default
	svc = NewDuplicateCheckServiceWithWorkers(repo, 0)
	if svc.workerCount != DefaultWorkerCount {
		t.Errorf("workerCount = %d, want %d for zero input", svc.workerCount, DefaultWorkerCount)
	}

	// Negative should default
	svc = NewDuplicateCheckServiceWithWorkers(repo, -5)
	if svc.workerCount != DefaultWorkerCount {
		t.Errorf("workerCount = %d, want %d for negative input", svc.workerCount, DefaultWorkerCount)
	}
}

func TestCheckBatchForDuplicates_EmptyBatch(t *testing.T) {
	repo := &MockTransactionRepo{}
	svc := NewDuplicateCheckService(repo)

	result := svc.CheckBatchForDuplicates(context.Background(), []*Transaction{}, 1)

	if result.TransactionsChecked != 0 {
		t.Errorf("TransactionsChecked = %d, want 0", result.TransactionsChecked)
	}
	if result.DuplicatesFound != 0 {
		t.Errorf("DuplicatesFound = %d, want 0", result.DuplicatesFound)
	}
}

func TestCheckTransactionForDuplicates_NoDuplicates(t *testing.T) {
	repo := &MockTransactionRepo{
		FindPotentialDuplicatesFunc: func(ctx context.Context, criteria DuplicateCriteria) ([]*Transaction, error) {
			return []*Transaction{}, nil
		},
	}
	svc := NewDuplicateCheckService(repo)

	txn := &Transaction{
		ID:              "tx-1",
		Amount:          100.0,
		Type:            "DEBIT",
		TransactionDate: time.Now(),
	}

	found, marked, err := svc.CheckTransactionForDuplicates(context.Background(), txn, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != 0 {
		t.Errorf("found = %d, want 0", found)
	}
	if marked != 0 {
		t.Errorf("marked = %d, want 0", marked)
	}
}

func TestCheckTransactionForDuplicates_FindsDuplicate(t *testing.T) {
	updateCalled := false
	repo := &MockTransactionRepo{
		FindPotentialDuplicatesFunc: func(ctx context.Context, criteria DuplicateCriteria) ([]*Transaction, error) {
			if criteria.OppositeType != "CREDIT" {
				t.Errorf("OppositeType = %q, want CREDIT for DEBIT transaction", criteria.OppositeType)
			}
			return []*Transaction{
				{ID: "tx-dup", Amount: 100.0, Type: "CREDIT"},
			}, nil
		},
		UpdateFunc: func(ctx context.Context, id string, params UpdateTransactionParams) (*Transaction, error) {
			updateCalled = true
			if id != "tx-dup" {
				t.Errorf("Update called with id %q, want tx-dup", id)
			}
			if params.Considered == nil || *params.Considered != false {
				t.Error("expected Considered=false")
			}
			return &Transaction{ID: id}, nil
		},
	}
	svc := NewDuplicateCheckService(repo)

	txn := &Transaction{
		ID:              "tx-1",
		Amount:          100.0,
		Type:            "DEBIT",
		TransactionDate: time.Now(),
	}

	found, marked, err := svc.CheckTransactionForDuplicates(context.Background(), txn, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != 1 {
		t.Errorf("found = %d, want 1", found)
	}
	if marked != 1 {
		t.Errorf("marked = %d, want 1", marked)
	}
	if !updateCalled {
		t.Error("expected Update to be called")
	}
}

func TestCheckTransactionForDuplicates_AlreadyMarkedSkipped(t *testing.T) {
	existingNote := "Esta transação não será considerada no cálculo"
	repo := &MockTransactionRepo{
		FindPotentialDuplicatesFunc: func(ctx context.Context, criteria DuplicateCriteria) ([]*Transaction, error) {
			return []*Transaction{
				{ID: "tx-dup", Amount: 100.0, Type: "CREDIT", Notes: &existingNote},
			}, nil
		},
	}
	svc := NewDuplicateCheckService(repo)

	txn := &Transaction{
		ID:              "tx-1",
		Amount:          100.0,
		Type:            "DEBIT",
		TransactionDate: time.Now(),
	}

	found, marked, err := svc.CheckTransactionForDuplicates(context.Background(), txn, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != 1 {
		t.Errorf("found = %d, want 1", found)
	}
	if marked != 0 {
		t.Errorf("marked = %d, want 0 (already marked)", marked)
	}
}

func TestCheckTransactionForDuplicates_RepoError(t *testing.T) {
	repo := &MockTransactionRepo{
		FindPotentialDuplicatesFunc: func(ctx context.Context, criteria DuplicateCriteria) ([]*Transaction, error) {
			return nil, errors.New("db error")
		},
	}
	svc := NewDuplicateCheckService(repo)

	txn := &Transaction{
		ID:              "tx-1",
		Amount:          100.0,
		Type:            "DEBIT",
		TransactionDate: time.Now(),
	}

	_, _, err := svc.CheckTransactionForDuplicates(context.Background(), txn, 1)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestCheckTransactionForDuplicates_CreditTransaction(t *testing.T) {
	repo := &MockTransactionRepo{
		FindPotentialDuplicatesFunc: func(ctx context.Context, criteria DuplicateCriteria) ([]*Transaction, error) {
			if criteria.OppositeType != "DEBIT" {
				t.Errorf("OppositeType = %q, want DEBIT for CREDIT transaction", criteria.OppositeType)
			}
			return []*Transaction{}, nil
		},
	}
	svc := NewDuplicateCheckService(repo)

	txn := &Transaction{
		ID:              "tx-1",
		Amount:          50.0,
		Type:            "CREDIT",
		TransactionDate: time.Now(),
	}

	_, _, err := svc.CheckTransactionForDuplicates(context.Background(), txn, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckBatchForDuplicates_WithTransactions(t *testing.T) {
	repo := &MockTransactionRepo{
		FindPotentialDuplicatesFunc: func(ctx context.Context, criteria DuplicateCriteria) ([]*Transaction, error) {
			return []*Transaction{}, nil
		},
	}
	svc := NewDuplicateCheckServiceWithWorkers(repo, 2)

	transactions := []*Transaction{
		{ID: "tx-1", Amount: 100.0, Type: "DEBIT", TransactionDate: time.Now()},
		{ID: "tx-2", Amount: 200.0, Type: "CREDIT", TransactionDate: time.Now()},
		{ID: "tx-3", Amount: 300.0, Type: "DEBIT", TransactionDate: time.Now()},
	}

	result := svc.CheckBatchForDuplicates(context.Background(), transactions, 1)

	if result.TransactionsChecked != 3 {
		t.Errorf("TransactionsChecked = %d, want 3", result.TransactionsChecked)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors = %v, want empty", result.Errors)
	}
}
