package openfinance

import (
	"context"
	"testing"

	"parsa/internal/domain/account"
	"parsa/internal/domain/user"
	ofclient "parsa/internal/infrastructure/openfinance"
	"parsa/internal/models"
)

// MockClient implements ofclient.ClientInterface
type MockClient struct {
	GetAccountsFunc     func(ctx context.Context, apiKey string) (*ofclient.AccountResponse, error)
	GetTransactionsFunc func(ctx context.Context, apiKey string) (*ofclient.TransactionResponse, error)
}

func (m *MockClient) GetAccounts(ctx context.Context, apiKey string) (*ofclient.AccountResponse, error) {
	if m.GetAccountsFunc != nil {
		return m.GetAccountsFunc(ctx, apiKey)
	}
	return &ofclient.AccountResponse{Success: true, Data: []ofclient.Account{}}, nil
}

func (m *MockClient) GetTransactions(ctx context.Context, apiKey string) (*ofclient.TransactionResponse, error) {
	if m.GetTransactionsFunc != nil {
		return m.GetTransactionsFunc(ctx, apiKey)
	}
	return &ofclient.TransactionResponse{Success: true, Data: []ofclient.Transaction{}}, nil
}

// MockItemRepo implements models.ItemRepository
type MockItemRepo struct {
	FindOrCreateFunc func(ctx context.Context, id string, userID int64) (*models.Item, error)
	ListByUserIDFunc func(ctx context.Context, userID int64) ([]*models.Item, error)
	DeleteFunc       func(ctx context.Context, id string) error
}

func (m *MockItemRepo) FindOrCreate(ctx context.Context, id string, userID int64) (*models.Item, error) {
	if m.FindOrCreateFunc != nil {
		return m.FindOrCreateFunc(ctx, id, userID)
	}
	return nil, nil
}

func (m *MockItemRepo) ListByUserID(ctx context.Context, userID int64) ([]*models.Item, error) {
	if m.ListByUserIDFunc != nil {
		return m.ListByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockItemRepo) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

// MockUserRepo implements user.Repository (Minimal implementation for sync)
type MockUserRepo struct {
	GetByIDFunc func(ctx context.Context, id int64) (*user.User, error)
	// Other methods can be nil for this test file
}

func (m *MockUserRepo) GetByID(ctx context.Context, id int64) (*user.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *MockUserRepo) Create(ctx context.Context, params user.CreateUserParams) (*user.User, error) {
	return nil, nil
}
func (m *MockUserRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	return nil, nil
}
func (m *MockUserRepo) GetByOAuth(ctx context.Context, provider, oauthID string) (*user.User, error) {
	return nil, nil
}
func (m *MockUserRepo) List(ctx context.Context) ([]*user.User, error) { return nil, nil }
func (m *MockUserRepo) Update(ctx context.Context, userID int64, params user.UpdateUserParams) (*user.User, error) {
	return nil, nil
}
func (m *MockUserRepo) ListUsersWithProviderKey(ctx context.Context) ([]*user.User, error) {
	return nil, nil
}

// MockAccountRepo for Service
type MockAccountRepo struct {
	CreateFunc       func(ctx context.Context, params account.CreateParams) (*account.Account, error)
	GetByIDFunc      func(ctx context.Context, id string) (*account.Account, error)
	ListByUserIDFunc func(ctx context.Context, userID int64) ([]*account.Account, error)
	DeleteFunc       func(ctx context.Context, id string) error
	UpsertFunc       func(ctx context.Context, params account.UpsertParams) (*account.Account, error)
	ExistsFunc       func(ctx context.Context, id string) (bool, error)
	FindByMatchFunc  func(ctx context.Context, userID int64, name, accountType, subtype string) (*account.Account, error)
	UpdateBankIDFunc func(ctx context.Context, accountID string, bankID int64) error
}

func (m *MockAccountRepo) Create(ctx context.Context, params account.CreateParams) (*account.Account, error) {
	return nil, nil
}
func (m *MockAccountRepo) GetByID(ctx context.Context, id string) (*account.Account, error) { return nil, nil }
func (m *MockAccountRepo) ListByUserID(ctx context.Context, userID int64) ([]*account.Account, error) {
	if m.ListByUserIDFunc != nil {
		return m.ListByUserIDFunc(ctx, userID)
	}
	return nil, nil
}
func (m *MockAccountRepo) Delete(ctx context.Context, id string) error { return nil }
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
	return nil, nil
}
func (m *MockAccountRepo) UpdateBankID(ctx context.Context, accountID string, bankID int64) error {
	return nil
}

func TestSyncUserAccounts(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		userID      int64
		mockUser    func() *MockUserRepo
		mockClient  func() *MockClient
		mockItem    func() *MockItemRepo
		mockAccount func() *MockAccountRepo
		wantErr     bool
		wantCreated int
		wantUpdated int
	}{
		{
			name:   "Success - Create New Account",
			userID: 1,
			mockUser: func() *MockUserRepo {
				key := "valid-key"
				return &MockUserRepo{
					GetByIDFunc: func(ctx context.Context, id int64) (*user.User, error) {
						return &user.User{ID: 1, ProviderKey: &key}, nil
					},
				}
			},
			mockClient: func() *MockClient {
				return &MockClient{
					GetAccountsFunc: func(ctx context.Context, apiKey string) (*ofclient.AccountResponse, error) {
						return &ofclient.AccountResponse{
							Success: true,
							Data: []ofclient.Account{
								{
									AccountID:           "acc-1",
									ItemID:              "item-1",
									AccountName:         "My Bank",
									AccountType:         "BANK",
									AccountCurrencyCode: "USD",
									BalanceString:       "100.50",
								},
							},
						}, nil
					},
				}
			},
			mockItem: func() *MockItemRepo {
				return &MockItemRepo{
					FindOrCreateFunc: func(ctx context.Context, id string, userID int64) (*models.Item, error) {
						return &models.Item{ID: id, UserID: userID}, nil
					},
				}
			},
			mockAccount: func() *MockAccountRepo {
				return &MockAccountRepo{
					ExistsFunc: func(ctx context.Context, id string) (bool, error) {
						return false, nil // Not exists
					},
					UpsertFunc: func(ctx context.Context, params account.UpsertParams) (*account.Account, error) {
						if params.ID != "acc-1" {
							t.Errorf("Upsert ID = %s, want acc-1", params.ID)
						}
						if params.Balance != 100.50 {
							t.Errorf("Upsert Balance = %f, want 100.50", params.Balance)
						}
						return &account.Account{ID: params.ID}, nil
					},
				}
			},
			wantErr:     false,
			wantCreated: 1,
			wantUpdated: 0,
		},
		{
			name:   "Success - Update Existing Account",
			userID: 1,
			mockUser: func() *MockUserRepo {
				key := "valid-key"
				return &MockUserRepo{
					GetByIDFunc: func(ctx context.Context, id int64) (*user.User, error) {
						return &user.User{ID: 1, ProviderKey: &key}, nil
					},
				}
			},
			mockClient: func() *MockClient {
				return &MockClient{
					GetAccountsFunc: func(ctx context.Context, apiKey string) (*ofclient.AccountResponse, error) {
						return &ofclient.AccountResponse{
							Success: true,
							Data: []ofclient.Account{
								{AccountID: "acc-1", AccountName: "My Bank", AccountType: "BANK", AccountCurrencyCode: "USD"},
							},
						}, nil
					},
				}
			},
			mockItem: func() *MockItemRepo {
				return &MockItemRepo{
					FindOrCreateFunc: func(ctx context.Context, id string, userID int64) (*models.Item, error) {
						return &models.Item{ID: "default", UserID: userID}, nil
					},
				}
			},
			mockAccount: func() *MockAccountRepo {
				return &MockAccountRepo{
					ExistsFunc: func(ctx context.Context, id string) (bool, error) {
						return true, nil // Exists
					},
					UpsertFunc: func(ctx context.Context, params account.UpsertParams) (*account.Account, error) {
						return &account.Account{ID: params.ID}, nil
					},
				}
			},
			wantErr:     false,
			wantCreated: 0,
			wantUpdated: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accRepo := tt.mockAccount()
			accService := account.NewService(accRepo)
			
			svc := NewAccountSyncService(
				tt.mockClient(),
				tt.mockUser(),
				accService,
				tt.mockItem(),
			)

			got, err := svc.SyncUserAccounts(ctx, tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SyncUserAccounts() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("SyncUserAccounts() unexpected error: %v", err)
				}
				if got.Created != tt.wantCreated {
					t.Errorf("SyncUserAccounts() created = %d, want %d", got.Created, tt.wantCreated)
				}
				if got.Updated != tt.wantUpdated {
					t.Errorf("SyncUserAccounts() updated = %d, want %d", got.Updated, tt.wantUpdated)
				}
			}
		})
	}
}
