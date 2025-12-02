package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"parsa/internal/domain/account"
	"parsa/internal/shared/middleware"
)

// MockAccountRepo implements account.Repository for testing
type MockAccountRepo struct {
	CreateFunc               func(ctx context.Context, params account.CreateParams) (*account.Account, error)
	GetByIDFunc              func(ctx context.Context, id string) (*account.Account, error)
	ListByUserIDFunc         func(ctx context.Context, userID int64) ([]*account.Account, error)
	ListByUserIDWithBankFunc func(ctx context.Context, userID int64) ([]*account.AccountWithBank, error)
	DeleteFunc               func(ctx context.Context, id string) error
	UpsertFunc               func(ctx context.Context, params account.UpsertParams) (*account.Account, error)
	ExistsFunc               func(ctx context.Context, id string) (bool, error)
	FindByMatchFunc          func(ctx context.Context, userID int64, name, accountType, subtype string) (*account.Account, error)
	UpdateBankIDFunc         func(ctx context.Context, accountID string, bankID int64) error
}

func (m *MockAccountRepo) Create(ctx context.Context, params account.CreateParams) (*account.Account, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, params)
	}
	return nil, nil
}

func (m *MockAccountRepo) GetByID(ctx context.Context, id string) (*account.Account, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockAccountRepo) ListByUserID(ctx context.Context, userID int64) ([]*account.Account, error) {
	if m.ListByUserIDFunc != nil {
		return m.ListByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockAccountRepo) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockAccountRepo) Upsert(ctx context.Context, params account.UpsertParams) (*account.Account, error) {
	if m.UpsertFunc != nil {
		return m.UpsertFunc(ctx, params)
	}
	return nil, nil
}

func (m *MockAccountRepo) Exists(ctx context.Context, id string) (bool, error) {
	if m.ExistsFunc != nil {
		return m.ExistsFunc(ctx, id)
	}
	return false, nil
}

func (m *MockAccountRepo) FindByMatch(ctx context.Context, userID int64, name, accountType, subtype string) (*account.Account, error) {
	if m.FindByMatchFunc != nil {
		return m.FindByMatchFunc(ctx, userID, name, accountType, subtype)
	}
	return nil, nil
}

func (m *MockAccountRepo) UpdateBankID(ctx context.Context, accountID string, bankID int64) error {
	if m.UpdateBankIDFunc != nil {
		return m.UpdateBankIDFunc(ctx, accountID, bankID)
	}
	return nil
}

func (m *MockAccountRepo) ListByUserIDWithBank(ctx context.Context, userID int64) ([]*account.AccountWithBank, error) {
	if m.ListByUserIDWithBankFunc != nil {
		return m.ListByUserIDWithBankFunc(ctx, userID)
	}
	return nil, nil
}

func TestHandleListAccounts(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		mockRepo       func() *MockAccountRepo
		expectedStatus int
	}{
		{
			name:   "Success",
			userID: 1,
			mockRepo: func() *MockAccountRepo {
				return &MockAccountRepo{
					ListByUserIDWithBankFunc: func(ctx context.Context, userID int64) ([]*account.AccountWithBank, error) {
						return []*account.AccountWithBank{
							{Account: account.Account{ID: "acc-1", UserID: 1, Name: "Test Account"}},
						}, nil
					},
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Empty List",
			userID: 1,
			mockRepo: func() *MockAccountRepo {
				return &MockAccountRepo{
					ListByUserIDWithBankFunc: func(ctx context.Context, userID int64) ([]*account.AccountWithBank, error) {
						return []*account.AccountWithBank{}, nil
					},
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Service Error",
			userID: 1,
			mockRepo: func() *MockAccountRepo {
				return &MockAccountRepo{
					ListByUserIDWithBankFunc: func(ctx context.Context, userID int64) ([]*account.AccountWithBank, error) {
						return nil, errors.New("db error")
					},
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mockRepo()
			service := account.NewService(repo)
			handler := NewAccountHandler(service)

			req, _ := http.NewRequest(http.MethodGet, "/api/accounts/", nil)
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, tt.userID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.HandleAccounts(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandleGetAccount(t *testing.T) {
	tests := []struct {
		name           string
		accountID      string
		userID         int64
		mockRepo       func() *MockAccountRepo
		expectedStatus int
	}{
		{
			name:      "Success",
			accountID: "acc-1",
			userID:    1,
			mockRepo: func() *MockAccountRepo {
				return &MockAccountRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*account.Account, error) {
						return &account.Account{ID: id, UserID: 1}, nil
					},
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "Not Found",
			accountID: "acc-999",
			userID:    1,
			mockRepo: func() *MockAccountRepo {
				return &MockAccountRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*account.Account, error) {
						return nil, account.ErrAccountNotFound
					},
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:      "Forbidden",
			accountID: "acc-2",
			userID:    1,
			mockRepo: func() *MockAccountRepo {
				return &MockAccountRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*account.Account, error) {
						// Account belongs to user 2
						return &account.Account{ID: id, UserID: 2}, nil
					},
				}
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mockRepo()
			service := account.NewService(repo)
			handler := NewAccountHandler(service)

			req, _ := http.NewRequest(http.MethodGet, "/api/accounts/"+tt.accountID, nil)
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, tt.userID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.HandleAccounts(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatus)
			}
		})
	}
}
