package account

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockRepository is a mock implementation of Repository interface
type MockRepository struct {
	CreateFunc                 func(ctx context.Context, params CreateParams) (*Account, error)
	GetByIDFunc                func(ctx context.Context, id string) (*Account, error)
	ListByUserIDFunc           func(ctx context.Context, userID int64) ([]*Account, error)
	ListByUserIDWithBankFunc   func(ctx context.Context, userID int64) ([]*AccountWithBank, error)
	DeleteFunc                 func(ctx context.Context, id string) error
	UpdateFunc                 func(ctx context.Context, id string, params UpdateParams) (*Account, error)
	UpsertFunc                 func(ctx context.Context, params UpsertParams) (*Account, error)
	ExistsFunc                 func(ctx context.Context, id string) (bool, error)
	FindByMatchFunc            func(ctx context.Context, userID int64, name, accountType, subtype string) (*Account, error)
	UpdateBankIDFunc           func(ctx context.Context, accountID string, bankID int64) error
	GetBalanceSumBySubtypeFunc func(ctx context.Context, userID int64, subtypes []string) (float64, error)
}

// GetBalanceSumBySubtype implements Repository.
func (m *MockRepository) GetBalanceSumBySubtype(ctx context.Context, userID int64, subtypes []string) (float64, error) {
	if m.GetBalanceSumBySubtypeFunc != nil {
		return m.GetBalanceSumBySubtypeFunc(ctx, userID, subtypes)
	}
	return 0, nil
}

func (m *MockRepository) Create(ctx context.Context, params CreateParams) (*Account, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, params)
	}
	return nil, nil
}

func (m *MockRepository) GetByID(ctx context.Context, id string) (*Account, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockRepository) ListByUserID(ctx context.Context, userID int64) ([]*Account, error) {
	if m.ListByUserIDFunc != nil {
		return m.ListByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockRepository) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockRepository) Update(ctx context.Context, id string, params UpdateParams) (*Account, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, id, params)
	}
	return nil, nil
}

func (m *MockRepository) Upsert(ctx context.Context, params UpsertParams) (*Account, error) {
	if m.UpsertFunc != nil {
		return m.UpsertFunc(ctx, params)
	}
	return nil, nil
}

func (m *MockRepository) Exists(ctx context.Context, id string) (bool, error) {
	if m.ExistsFunc != nil {
		return m.ExistsFunc(ctx, id)
	}
	return false, nil
}

func (m *MockRepository) FindByMatch(ctx context.Context, userID int64, name, accountType, subtype string) (*Account, error) {
	if m.FindByMatchFunc != nil {
		return m.FindByMatchFunc(ctx, userID, name, accountType, subtype)
	}
	return nil, nil
}

func (m *MockRepository) UpdateBankID(ctx context.Context, accountID string, bankID int64) error {
	if m.UpdateBankIDFunc != nil {
		return m.UpdateBankIDFunc(ctx, accountID, bankID)
	}
	return nil
}

func (m *MockRepository) ListByUserIDWithBank(ctx context.Context, userID int64) ([]*AccountWithBank, error) {
	if m.ListByUserIDWithBankFunc != nil {
		return m.ListByUserIDWithBankFunc(ctx, userID)
	}
	return nil, nil
}

func TestCreateAccount(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		params  CreateParams
		mock    func() *MockRepository
		wantErr bool
		errType error
	}{
		{
			name: "Success",
			params: CreateParams{
				ID:          "acc-123",
				UserID:      1,
				Name:        "Test Account",
				AccountType: "BANK",
				Currency:    "USD",
				Balance:     100.0,
			},
			mock: func() *MockRepository {
				return &MockRepository{
					CreateFunc: func(ctx context.Context, params CreateParams) (*Account, error) {
						return &Account{
							ID:          params.ID,
							UserID:      params.UserID,
							Name:        params.Name,
							AccountType: params.AccountType,
							Currency:    params.Currency,
							Balance:     params.Balance,
							CreatedAt:   time.Now(),
							UpdatedAt:   time.Now(),
						}, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name: "Invalid Currency",
			params: CreateParams{
				ID:          "acc-123",
				UserID:      1,
				Name:        "Test Account",
				AccountType: "BANK",
				Currency:    "INVALID",
			},
			mock: func() *MockRepository {
				return &MockRepository{}
			},
			wantErr: true,
			errType: ErrInvalidCurrency,
		},
		{
			name: "Invalid Account Type",
			params: CreateParams{
				ID:          "acc-123",
				UserID:      1,
				Name:        "Test Account",
				AccountType: "UNKNOWN",
				Currency:    "USD",
			},
			mock: func() *MockRepository {
				return &MockRepository{}
			},
			wantErr: true,
			errType: ErrInvalidAccountType,
		},
		{
			name: "Repository Error",
			params: CreateParams{
				ID:          "acc-123",
				UserID:      1,
				Name:        "Test Account",
				AccountType: "BANK",
				Currency:    "USD",
			},
			mock: func() *MockRepository {
				return &MockRepository{
					CreateFunc: func(ctx context.Context, params CreateParams) (*Account, error) {
						return nil, errors.New("db error")
					},
				}
			},
			wantErr: true,
		},
		{
			name: "Success with Balance Sum",
			params: CreateParams{
				ID:          "acc-123",
				UserID:      1,
				Name:        "Test Account",
				AccountType: "BANK",
				Currency:    "USD",
				Balance:     100.0,
			},
			mock: func() *MockRepository {
				return &MockRepository{
					CreateFunc: func(ctx context.Context, params CreateParams) (*Account, error) {
						return &Account{ID: params.ID, UserID: params.UserID, Name: params.Name, AccountType: params.AccountType, Currency: params.Currency, Balance: params.Balance, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
					},
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mock()
			service := NewService(repo)

			acc, err := service.CreateAccount(ctx, tt.params)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateAccount() expected error, got nil")
				}
				if tt.errType != nil && err != tt.errType {
					t.Errorf("CreateAccount() expected error type %v, got %v", tt.errType, err)
				}
			} else {
				if err != nil {
					t.Errorf("CreateAccount() unexpected error: %v", err)
				}
				if acc == nil {
					t.Errorf("CreateAccount() expected account, got nil")
				} else if acc.ID != tt.params.ID {
					t.Errorf("CreateAccount() expected ID %s, got %s", tt.params.ID, acc.ID)
				}
			}
		})
	}
}

func TestGetAccount(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		accountID string
		userID    int64
		mock      func() *MockRepository
		wantErr   bool
		errType   error
	}{
		{
			name:      "Success",
			accountID: "acc-123",
			userID:    1,
			mock: func() *MockRepository {
				return &MockRepository{
					GetByIDFunc: func(ctx context.Context, id string) (*Account, error) {
						return &Account{ID: id, UserID: 1}, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:      "Not Found",
			accountID: "acc-999",
			userID:    1,
			mock: func() *MockRepository {
				return &MockRepository{
					GetByIDFunc: func(ctx context.Context, id string) (*Account, error) {
						return nil, ErrAccountNotFound
					},
				}
			},
			wantErr: true,
			errType: ErrAccountNotFound,
		},
		{
			name:      "Forbidden",
			accountID: "acc-123",
			userID:    2, // Different user
			mock: func() *MockRepository {
				return &MockRepository{
					GetByIDFunc: func(ctx context.Context, id string) (*Account, error) {
						return &Account{ID: id, UserID: 1}, nil // Owned by user 1
					},
				}
			},
			wantErr: true,
			errType: ErrForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mock()
			service := NewService(repo)

			acc, err := service.GetAccount(ctx, tt.accountID, tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetAccount() expected error, got nil")
				}
				if tt.errType != nil && err != tt.errType {
					t.Errorf("GetAccount() expected error type %v, got %v", tt.errType, err)
				}
			} else {
				if err != nil {
					t.Errorf("GetAccount() unexpected error: %v", err)
				}
				if acc == nil {
					t.Errorf("GetAccount() expected account, got nil")
				}
			}
		})
	}
}

func TestDeleteAccount(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		accountID string
		userID    int64
		mock      func() *MockRepository
		wantErr   bool
		errType   error
	}{
		{
			name:      "Success",
			accountID: "acc-123",
			userID:    1,
			mock: func() *MockRepository {
				return &MockRepository{
					GetByIDFunc: func(ctx context.Context, id string) (*Account, error) {
						return &Account{ID: id, UserID: 1}, nil
					},
					DeleteFunc: func(ctx context.Context, id string) error {
						return nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:      "Not Found",
			accountID: "acc-999",
			userID:    1,
			mock: func() *MockRepository {
				return &MockRepository{
					GetByIDFunc: func(ctx context.Context, id string) (*Account, error) {
						return nil, ErrAccountNotFound
					},
				}
			},
			wantErr: true,
			errType: ErrAccountNotFound,
		},
		{
			name:      "Forbidden",
			accountID: "acc-123",
			userID:    2,
			mock: func() *MockRepository {
				return &MockRepository{
					GetByIDFunc: func(ctx context.Context, id string) (*Account, error) {
						return &Account{ID: id, UserID: 1}, nil
					},
				}
			},
			wantErr: true,
			errType: ErrForbidden,
		},
		{
			name:      "Delete Error",
			accountID: "acc-123",
			userID:    1,
			mock: func() *MockRepository {
				return &MockRepository{
					GetByIDFunc: func(ctx context.Context, id string) (*Account, error) {
						return &Account{ID: id, UserID: 1}, nil
					},
					DeleteFunc: func(ctx context.Context, id string) error {
						return errors.New("db error")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mock()
			service := NewService(repo)

			err := service.DeleteAccount(ctx, tt.accountID, tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteAccount() expected error, got nil")
				}
				if tt.errType != nil && err != tt.errType {
					t.Errorf("DeleteAccount() expected error type %v, got %v", tt.errType, err)
				}
			} else {
				if err != nil {
					t.Errorf("DeleteAccount() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestListAccountsByUserID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		userID  int64
		mock    func() *MockRepository
		wantErr bool
		wantLen int
	}{
		{
			name:   "Success",
			userID: 1,
			mock: func() *MockRepository {
				return &MockRepository{
					ListByUserIDFunc: func(ctx context.Context, userID int64) ([]*Account, error) {
						return []*Account{{ID: "acc-1"}, {ID: "acc-2"}}, nil
					},
				}
			},
			wantErr: false,
			wantLen: 2,
		},
		{
			name:   "Empty List",
			userID: 1,
			mock: func() *MockRepository {
				return &MockRepository{
					ListByUserIDFunc: func(ctx context.Context, userID int64) ([]*Account, error) {
						return []*Account{}, nil
					},
				}
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name:   "Invalid User ID",
			userID: 0,
			mock: func() *MockRepository {
				return &MockRepository{}
			},
			wantErr: true,
		},
		{
			name:   "Repository Error",
			userID: 1,
			mock: func() *MockRepository {
				return &MockRepository{
					ListByUserIDFunc: func(ctx context.Context, userID int64) ([]*Account, error) {
						return nil, errors.New("db error")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mock()
			service := NewService(repo)

			accounts, err := service.ListAccountsByUserID(ctx, tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ListAccountsByUserID() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ListAccountsByUserID() unexpected error: %v", err)
				}
				if len(accounts) != tt.wantLen {
					t.Errorf("ListAccountsByUserID() got %d accounts, want %d", len(accounts), tt.wantLen)
				}
			}
		})
	}
}

func TestUpsertAccount(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		params  UpsertParams
		mock    func() *MockRepository
		wantErr bool
		errType error
	}{
		{
			name: "Success",
			params: UpsertParams{
				ID:          "acc-123",
				UserID:      1,
				Name:        "Test Account",
				AccountType: "BANK",
				Currency:    "USD",
				Balance:     100.0,
			},
			mock: func() *MockRepository {
				return &MockRepository{
					UpsertFunc: func(ctx context.Context, params UpsertParams) (*Account, error) {
						return &Account{ID: params.ID}, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name: "Default Currency Applied",
			params: UpsertParams{
				ID:          "acc-123",
				UserID:      1,
				Name:        "Test Account",
				AccountType: "BANK",
				Currency:    "",
			},
			mock: func() *MockRepository {
				return &MockRepository{
					UpsertFunc: func(ctx context.Context, params UpsertParams) (*Account, error) {
						if params.Currency != "BRL" {
							t.Errorf("Expected default currency BRL, got %s", params.Currency)
						}
						return &Account{ID: params.ID}, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name: "Invalid Account Type",
			params: UpsertParams{
				ID:          "acc-123",
				UserID:      1,
				Name:        "Test Account",
				AccountType: "INVALID",
				Currency:    "USD",
			},
			mock: func() *MockRepository {
				return &MockRepository{}
			},
			wantErr: true,
			errType: ErrInvalidAccountType,
		},
		{
			name: "Invalid Subtype",
			params: UpsertParams{
				ID:          "acc-123",
				UserID:      1,
				Name:        "Test Account",
				AccountType: "BANK",
				Currency:    "USD",
				Subtype:     strPtr("INVALID"),
			},
			mock: func() *MockRepository {
				return &MockRepository{}
			},
			wantErr: true,
			errType: ErrInvalidAccountSubtype,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mock()
			service := NewService(repo)

			acc, err := service.UpsertAccount(ctx, tt.params)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpsertAccount() expected error, got nil")
				}
				if tt.errType != nil && err != tt.errType {
					t.Errorf("UpsertAccount() expected error type %v, got %v", tt.errType, err)
				}
			} else {
				if err != nil {
					t.Errorf("UpsertAccount() unexpected error: %v", err)
				}
				if acc == nil {
					t.Errorf("UpsertAccount() expected account, got nil")
				}
			}
		})
	}
}

func strPtr(s string) *string { return &s }
