package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"parsa/internal/domain/account"
	"parsa/internal/domain/transaction"
	"parsa/internal/shared/middleware"
)

// MockTransactionRepo implements transaction.Repository for testing
type MockTransactionRepo struct {
	CreateFunc          func(ctx context.Context, params transaction.CreateTransactionParams) (*transaction.Transaction, error)
	GetByIDFunc         func(ctx context.Context, id string) (*transaction.Transaction, error)
	ListByAccountIDFunc func(ctx context.Context, accountID string, limit, offset int) ([]*transaction.Transaction, error)
	UpdateFunc          func(ctx context.Context, id string, params transaction.UpdateTransactionParams) (*transaction.Transaction, error)
	DeleteFunc          func(ctx context.Context, id string) error
	UpsertFunc          func(ctx context.Context, params transaction.UpsertTransactionParams) (*transaction.Transaction, error)
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

func TestHandleListTransactions(t *testing.T) {
	tests := []struct {
		name           string
		accountID      string
		userID         int64
		mockTxRepo     func() *MockTransactionRepo
		mockAccRepo    func() *MockAccountRepo
		expectedStatus int
	}{
		{
			name:      "Success",
			accountID: "acc-1",
			userID:    1,
			mockTxRepo: func() *MockTransactionRepo {
				return &MockTransactionRepo{
					ListByAccountIDFunc: func(ctx context.Context, accountID string, limit, offset int) ([]*transaction.Transaction, error) {
						return []*transaction.Transaction{{ID: "tx-1"}}, nil
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
			name:      "Forbidden",
			accountID: "acc-1",
			userID:    2, // Different user
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txRepo := tt.mockTxRepo()
			accRepo := tt.mockAccRepo()
			handler := NewTransactionHandler(txRepo, accRepo)

			req, _ := http.NewRequest(http.MethodGet, "/api/transactions?accountId="+tt.accountID, nil)
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
						return &transaction.Transaction{ID: params.ID}, nil
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
			handler := NewTransactionHandler(txRepo, accRepo)

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
						return &transaction.Transaction{ID: "tx-1", AccountID: "acc-1"}, nil
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
						return nil, nil // Repo returns nil for not found in this implementation
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
						return &transaction.Transaction{ID: "tx-1", AccountID: "acc-1"}, nil
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
			handler := NewTransactionHandler(txRepo, accRepo)

			req, _ := http.NewRequest(http.MethodGet, "/api/transactions/"+tt.transactionID, nil)
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, tt.userID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.HandleGetTransaction(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatus)
			}
		})
	}
}
