package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"parsa/internal/domain/account"
	"parsa/internal/domain/cousinrule"
	"parsa/internal/domain/transaction"
	"parsa/internal/shared/middleware"
)

// MockTransactionRepo implements transaction.Repository for testing
type MockTransactionRepo struct {
	CreateFunc                         func(ctx context.Context, params transaction.CreateTransactionParams) (*transaction.Transaction, error)
	GetByIDFunc                        func(ctx context.Context, id string) (*transaction.Transaction, error)
	ListByAccountIDFunc                func(ctx context.Context, accountID string, limit, offset int) ([]*transaction.Transaction, error)
	ListByUserIDFunc                   func(ctx context.Context, userID int64, limit, offset int) ([]*transaction.Transaction, error)
	CountByUserIDFunc                  func(ctx context.Context, userID int64) (int64, error)
	UpdateFunc                         func(ctx context.Context, id string, params transaction.UpdateTransactionParams) (*transaction.Transaction, error)
	DeleteFunc                         func(ctx context.Context, id string) error
	UpsertFunc                         func(ctx context.Context, params transaction.UpsertTransactionParams) (*transaction.Transaction, error)
	FindPotentialDuplicatesFunc        func(ctx context.Context, criteria transaction.DuplicateCriteria) ([]*transaction.Transaction, error)
	FindPotentialDuplicatesForBillFunc func(ctx context.Context, criteria transaction.DuplicateCriteria) ([]*transaction.Transaction, error)
	SetTransactionTagsFunc             func(ctx context.Context, transactionID string, tagIDs []string) error
	GetTransactionTagsFunc             func(ctx context.Context, transactionID string) ([]string, error)
}

func (m *MockTransactionRepo) Create(ctx context.Context, params transaction.CreateTransactionParams) (*transaction.Transaction, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, params)
	}
	return nil, nil
}

func (m *MockTransactionRepo) GetByID(ctx context.Context, id string) (*transaction.Transaction, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockTransactionRepo) ListByAccountID(ctx context.Context, accountID string, limit, offset int) ([]*transaction.Transaction, error) {
	if m.ListByAccountIDFunc != nil {
		return m.ListByAccountIDFunc(ctx, accountID, limit, offset)
	}
	return nil, nil
}

func (m *MockTransactionRepo) ListByUserID(ctx context.Context, userID int64, limit, offset int) ([]*transaction.Transaction, error) {
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

func (m *MockTransactionRepo) Update(ctx context.Context, id string, params transaction.UpdateTransactionParams) (*transaction.Transaction, error) {
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

func (m *MockTransactionRepo) Upsert(ctx context.Context, params transaction.UpsertTransactionParams) (*transaction.Transaction, error) {
	if m.UpsertFunc != nil {
		return m.UpsertFunc(ctx, params)
	}
	return nil, nil
}

func (m *MockTransactionRepo) FindPotentialDuplicates(ctx context.Context, criteria transaction.DuplicateCriteria) ([]*transaction.Transaction, error) {
	if m.FindPotentialDuplicatesFunc != nil {
		return m.FindPotentialDuplicatesFunc(ctx, criteria)
	}
	return nil, nil
}

func (m *MockTransactionRepo) FindPotentialDuplicatesForBill(ctx context.Context, criteria transaction.DuplicateCriteria) ([]*transaction.Transaction, error) {
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

// MockCousinRuleRepo implements cousinrule.Repository for testing
type MockCousinRuleRepo struct {
	CreateFunc                 func(ctx context.Context, params cousinrule.CreateCousinRuleParams) (*cousinrule.CousinRule, error)
	GetByIDFunc                func(ctx context.Context, id int64) (*cousinrule.CousinRule, error)
	GetByCousinAndTypeFunc     func(ctx context.Context, userID, cousinID int64, txType *string) (*cousinrule.CousinRule, error)
	ListByUserIDFunc           func(ctx context.Context, userID int64) ([]*cousinrule.CousinRule, error)
	ListByCousinIDFunc         func(ctx context.Context, userID, cousinID int64) ([]*cousinrule.CousinRule, error)
	UpdateFunc                 func(ctx context.Context, id int64, params cousinrule.UpdateCousinRuleParams) (*cousinrule.CousinRule, error)
	UpsertFunc                 func(ctx context.Context, params cousinrule.CreateCousinRuleParams) (*cousinrule.CousinRule, bool, error)
	DeleteFunc                 func(ctx context.Context, id int64) error
	SetRuleTagsFunc            func(ctx context.Context, ruleID int64, tagIDs []string) error
	GetRuleTagsFunc            func(ctx context.Context, ruleID int64) ([]string, error)
	ApplyRuleToTransactionsFunc func(ctx context.Context, userID, cousinID int64, txType *string, changes cousinrule.Changes) (int, error)
	CheckDontAskAgainFunc      func(ctx context.Context, userID, cousinID int64, txType string) (bool, error)
}

func (m *MockCousinRuleRepo) Create(ctx context.Context, params cousinrule.CreateCousinRuleParams) (*cousinrule.CousinRule, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, params)
	}
	return nil, nil
}

func (m *MockCousinRuleRepo) GetByID(ctx context.Context, id int64) (*cousinrule.CousinRule, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockCousinRuleRepo) GetByCousinAndType(ctx context.Context, userID, cousinID int64, txType *string) (*cousinrule.CousinRule, error) {
	if m.GetByCousinAndTypeFunc != nil {
		return m.GetByCousinAndTypeFunc(ctx, userID, cousinID, txType)
	}
	return nil, nil
}

func (m *MockCousinRuleRepo) ListByUserID(ctx context.Context, userID int64) ([]*cousinrule.CousinRule, error) {
	if m.ListByUserIDFunc != nil {
		return m.ListByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockCousinRuleRepo) ListByCousinID(ctx context.Context, userID, cousinID int64) ([]*cousinrule.CousinRule, error) {
	if m.ListByCousinIDFunc != nil {
		return m.ListByCousinIDFunc(ctx, userID, cousinID)
	}
	return nil, nil
}

func (m *MockCousinRuleRepo) Update(ctx context.Context, id int64, params cousinrule.UpdateCousinRuleParams) (*cousinrule.CousinRule, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, id, params)
	}
	return nil, nil
}

func (m *MockCousinRuleRepo) Upsert(ctx context.Context, params cousinrule.CreateCousinRuleParams) (*cousinrule.CousinRule, bool, error) {
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
	return nil, nil
}

func (m *MockCousinRuleRepo) ApplyRuleToTransactions(ctx context.Context, userID, cousinID int64, txType *string, changes cousinrule.Changes) (int, error) {
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

func TestHandleListTransactions(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		mockTxRepo     func() *MockTransactionRepo
		mockAccRepo    func() *MockAccountRepo
		expectedStatus int
	}{
		{
			name:   "Success",
			userID: 1,
			mockTxRepo: func() *MockTransactionRepo {
				return &MockTransactionRepo{
					CountByUserIDFunc: func(ctx context.Context, userID int64) (int64, error) {
						return 1, nil
					},
					ListByUserIDFunc: func(ctx context.Context, userID int64, limit, offset int) ([]*transaction.Transaction, error) {
						return []*transaction.Transaction{{ID: "tx-1", AccountID: "acc-1", Type: "DEBIT", Status: "POSTED"}}, nil
					},
					GetTransactionTagsFunc: func(ctx context.Context, transactionID string) ([]string, error) {
						return []string{}, nil
					},
				}
			},
			mockAccRepo: func() *MockAccountRepo {
				return &MockAccountRepo{}
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txRepo := tt.mockTxRepo()
			accRepo := tt.mockAccRepo()
			cousinRepo := &MockCousinRuleRepo{}
			handler := NewTransactionHandler(txRepo, accRepo, cousinRepo)

			req, _ := http.NewRequest(http.MethodGet, "/api/transactions", nil)
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, tt.userID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.HandleListTransactions(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandleCreateTransaction(t *testing.T) {
	tests := []struct {
		name           string
		body           map[string]interface{}
		userID         int64
		mockTxRepo     func() *MockTransactionRepo
		mockAccRepo    func() *MockAccountRepo
		expectedStatus int
	}{
		{
			name: "Success",
			body: map[string]interface{}{
				"accountId":       "acc-1",
				"amount":          100.0,
				"description":     "Test Tx",
				"transactionDate": "2023-01-01",
			},
			userID: 1,
			mockTxRepo: func() *MockTransactionRepo {
				return &MockTransactionRepo{
					CreateFunc: func(ctx context.Context, params transaction.CreateTransactionParams) (*transaction.Transaction, error) {
						return &transaction.Transaction{ID: params.ID, AccountID: params.AccountID, Type: "DEBIT", Status: "POSTED"}, nil
					},
				}
			},
			mockAccRepo: func() *MockAccountRepo {
				return &MockAccountRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*account.Account, error) {
						return &account.Account{ID: "acc-1", UserID: 1}, nil
					},
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Forbidden",
			body: map[string]interface{}{
				"accountId":       "acc-1",
				"amount":          100.0,
				"description":     "Test Tx",
				"transactionDate": "2023-01-01",
			},
			userID: 2,
			mockTxRepo: func() *MockTransactionRepo {
				return &MockTransactionRepo{}
			},
			mockAccRepo: func() *MockAccountRepo {
				return &MockAccountRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*account.Account, error) {
						return &account.Account{ID: "acc-1", UserID: 1}, nil
					},
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "Invalid Date",
			body: map[string]interface{}{
				"accountId":       "acc-1",
				"amount":          100.0,
				"description":     "Test Tx",
				"transactionDate": "invalid-date",
			},
			userID: 1,
			mockTxRepo: func() *MockTransactionRepo {
				return &MockTransactionRepo{}
			},
			mockAccRepo: func() *MockAccountRepo {
				return &MockAccountRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*account.Account, error) {
						return &account.Account{ID: "acc-1", UserID: 1}, nil
					},
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txRepo := tt.mockTxRepo()
			accRepo := tt.mockAccRepo()
			cousinRepo := &MockCousinRuleRepo{}
			handler := NewTransactionHandler(txRepo, accRepo, cousinRepo)

			bodyBytes, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPost, "/api/transactions", bytes.NewBuffer(bodyBytes))
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, tt.userID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.HandleCreateTransaction(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandleGetTransaction(t *testing.T) {
	tests := []struct {
		name           string
		transactionID  string
		userID         int64
		mockTxRepo     func() *MockTransactionRepo
		mockAccRepo    func() *MockAccountRepo
		expectedStatus int
	}{
		{
			name:          "Success",
			transactionID: "tx-1",
			userID:        1,
			mockTxRepo: func() *MockTransactionRepo {
				return &MockTransactionRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*transaction.Transaction, error) {
						return &transaction.Transaction{ID: "tx-1", AccountID: "acc-1", Type: "DEBIT", Status: "POSTED"}, nil
					},
					GetTransactionTagsFunc: func(ctx context.Context, transactionID string) ([]string, error) {
						return []string{}, nil
					},
				}
			},
			mockAccRepo: func() *MockAccountRepo {
				return &MockAccountRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*account.Account, error) {
						return &account.Account{ID: "acc-1", UserID: 1}, nil
					},
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:          "Not Found",
			transactionID: "tx-999",
			userID:        1,
			mockTxRepo: func() *MockTransactionRepo {
				return &MockTransactionRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*transaction.Transaction, error) {
						return nil, nil
					},
				}
			},
			mockAccRepo: func() *MockAccountRepo {
				return &MockAccountRepo{}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:          "Forbidden",
			transactionID: "tx-1",
			userID:        2,
			mockTxRepo: func() *MockTransactionRepo {
				return &MockTransactionRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*transaction.Transaction, error) {
						return &transaction.Transaction{ID: "tx-1", AccountID: "acc-1", Type: "DEBIT", Status: "POSTED"}, nil
					},
				}
			},
			mockAccRepo: func() *MockAccountRepo {
				return &MockAccountRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*account.Account, error) {
						return &account.Account{ID: "acc-1", UserID: 1}, nil
					},
				}
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txRepo := tt.mockTxRepo()
			accRepo := tt.mockAccRepo()
			cousinRepo := &MockCousinRuleRepo{}
			handler := NewTransactionHandler(txRepo, accRepo, cousinRepo)

			// Use ServeMux so r.PathValue("id") is populated
			mux := http.NewServeMux()
			mux.HandleFunc("GET /api/transactions/{id}", handler.HandleGetTransaction)

			req, _ := http.NewRequest(http.MethodGet, "/api/transactions/"+tt.transactionID, nil)
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, tt.userID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatus)
			}
		})
	}
}
