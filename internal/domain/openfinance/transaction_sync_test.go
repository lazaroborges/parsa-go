package openfinance

import (
	"context"
	"testing"

	"parsa/internal/domain/account"
	"parsa/internal/domain/transaction"
	"parsa/internal/domain/user"
	ofclient "parsa/internal/infrastructure/openfinance"
	"parsa/internal/models"
)

// Additional mocks for Transaction Sync

type MockTransactionRepo struct {
	CreateFunc          func(ctx context.Context, params transaction.CreateTransactionParams) (*transaction.Transaction, error)
	GetByIDFunc         func(ctx context.Context, id string) (*transaction.Transaction, error)
	ListByAccountIDFunc func(ctx context.Context, accountID string, limit, offset int) ([]*transaction.Transaction, error)
	ListByUserIDFunc    func(ctx context.Context, userID int64, limit, offset int) ([]*transaction.Transaction, error)
	CountByUserIDFunc   func(ctx context.Context, userID int64) (int64, error)
	UpdateFunc          func(ctx context.Context, id string, params transaction.UpdateTransactionParams) (*transaction.Transaction, error)
	DeleteFunc          func(ctx context.Context, id string) error
	UpsertFunc          func(ctx context.Context, params transaction.UpsertTransactionParams) (*transaction.Transaction, error)
	FindPotentialDuplicatesFunc func(ctx context.Context, criteria transaction.DuplicateCriteria) ([]*transaction.Transaction, error)
	FindPotentialDuplicatesForBillFunc func(ctx context.Context, criteria transaction.DuplicateCriteria) ([]*transaction.Transaction, error)
	SetTransactionTagsFunc func(ctx context.Context, transactionID string, tagIDs []string) error
	GetTransactionTagsFunc func(ctx context.Context, transactionID string) ([]string, error)
}

func (m *MockTransactionRepo) Create(ctx context.Context, params transaction.CreateTransactionParams) (*transaction.Transaction, error) {
	return nil, nil
}
func (m *MockTransactionRepo) GetByID(ctx context.Context, id string) (*transaction.Transaction, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *MockTransactionRepo) ListByAccountID(ctx context.Context, accountID string, limit, offset int) ([]*transaction.Transaction, error) {
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
	return nil, nil
}
func (m *MockTransactionRepo) Delete(ctx context.Context, id string) error { return nil }
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

type MockCreditCardDataRepo struct {
	UpsertFunc func(ctx context.Context, transactionID string, params models.CreateCreditCardDataParams) (*models.CreditCardData, error)
}

func (m *MockCreditCardDataRepo) Upsert(ctx context.Context, transactionID string, params models.CreateCreditCardDataParams) (*models.CreditCardData, error) {
	if m.UpsertFunc != nil {
		return m.UpsertFunc(ctx, transactionID, params)
	}
	return nil, nil
}

type MockBankRepo struct {
	FindOrCreateByNameFunc func(ctx context.Context, name string) (*models.Bank, error)
}

func (m *MockBankRepo) FindOrCreateByName(ctx context.Context, name string) (*models.Bank, error) {
	if m.FindOrCreateByNameFunc != nil {
		return m.FindOrCreateByNameFunc(ctx, name)
	}
	return nil, nil
}

func TestSyncUserTransactions(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		userID      int64
		mockUser    func() *MockUserRepo
		mockClient  func() *MockClient
		mockTxRepo  func() *MockTransactionRepo
		mockAccRepo func() *MockAccountRepo
		mockBank    func() *MockBankRepo
		mockCC      func() *MockCreditCardDataRepo
		wantErr     bool
		wantCreated int
		wantUpdated int
		wantSkipped int
	}{
		{
			name:   "Success - Create New Transaction",
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
					GetTransactionsFunc: func(ctx context.Context, apiKey string, startDate string) (*ofclient.TransactionResponse, error) {
						return &ofclient.TransactionResponse{
							Success: true,
							Data: []ofclient.Transaction{
								{
									ID:             "tx-1",
									Description:    "Uber Trip",
									AmountString:   "25.50",
									DateString:     "2023-10-27 10:00:00",
									AccountName:    "Checking",
									AccountType:    "BANK",
									AccountSubtype: "CHECKING_ACCOUNT",
									Type:           "DEBIT",
									Status:         "POSTED",
								},
							},
						}, nil
					},
				}
			},
			mockAccRepo: func() *MockAccountRepo {
				return &MockAccountRepo{
					ListByUserIDFunc: func(ctx context.Context, userID int64) ([]*account.Account, error) {
						return []*account.Account{
							{
								ID:          "acc-1",
								Name:        "Checking",
								AccountType: "BANK",
								Subtype:     "CHECKING_ACCOUNT",
							},
						}, nil
					},
				}
			},
			mockTxRepo: func() *MockTransactionRepo {
				return &MockTransactionRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*transaction.Transaction, error) {
						return nil, nil // Not exists
					},
					UpsertFunc: func(ctx context.Context, params transaction.UpsertTransactionParams) (*transaction.Transaction, error) {
						return &transaction.Transaction{ID: params.ID}, nil
					},
				}
			},
			mockBank: func() *MockBankRepo { return &MockBankRepo{} },
			mockCC:   func() *MockCreditCardDataRepo { return &MockCreditCardDataRepo{} },
			wantErr:     false,
			wantCreated: 1,
			wantUpdated: 0,
			wantSkipped: 0,
		},
		{
			name:   "Success - Skip Unmatched Transaction",
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
					GetTransactionsFunc: func(ctx context.Context, apiKey string, startDate string) (*ofclient.TransactionResponse, error) {
						return &ofclient.TransactionResponse{
							Success: true,
							Data: []ofclient.Transaction{
								{
									ID:          "tx-2",
									Description: "Unknown",
									AccountName: "Unknown Account", // No match
								},
							},
						}, nil
					},
				}
			},
			mockAccRepo: func() *MockAccountRepo {
				return &MockAccountRepo{
					ListByUserIDFunc: func(ctx context.Context, userID int64) ([]*account.Account, error) {
						return []*account.Account{}, nil // No accounts
					},
					FindByMatchFunc: func(ctx context.Context, userID int64, name, accountType, subtype string) (*account.Account, error) {
						return nil, nil // Not found in DB either
					},
				}
			},
			mockTxRepo: func() *MockTransactionRepo { return &MockTransactionRepo{} },
			mockBank:   func() *MockBankRepo { return &MockBankRepo{} },
			mockCC:     func() *MockCreditCardDataRepo { return &MockCreditCardDataRepo{} },
			wantErr:     false,
			wantCreated: 0,
			wantUpdated: 0,
			wantSkipped: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accRepo := tt.mockAccRepo()
			accService := account.NewService(accRepo)

			svc := NewTransactionSyncService(
				tt.mockClient(),
				tt.mockUser(),
				accService,
				accRepo,
				tt.mockTxRepo(),
				tt.mockCC(),
				tt.mockBank(),
				"2023-01-01", // Default test start date
			)

			got, err := svc.SyncUserTransactions(ctx, tt.userID, false)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SyncUserTransactions() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("SyncUserTransactions() unexpected error: %v", err)
				}
				if got == nil {
					t.Fatalf("SyncUserTransactions() returned nil result")
				}
				if got.Created != tt.wantCreated {
					t.Errorf("SyncUserTransactions() created = %d, want %d", got.Created, tt.wantCreated)
				}
				if got.Updated != tt.wantUpdated {
					t.Errorf("SyncUserTransactions() updated = %d, want %d", got.Updated, tt.wantUpdated)
				}
				if got.Skipped != tt.wantSkipped {
					t.Errorf("SyncUserTransactions() skipped = %d, want %d", got.Skipped, tt.wantSkipped)
				}
			}
		})
	}
}
