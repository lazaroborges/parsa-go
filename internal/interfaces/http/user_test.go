package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"parsa/internal/domain/user"
	"parsa/internal/shared/middleware"
)

// MockUserRepo implements user.Repository for testing
type MockUserRepo struct {
	CreateFunc                   func(ctx context.Context, params user.CreateUserParams) (*user.User, error)
	GetByIDFunc                  func(ctx context.Context, id int64) (*user.User, error)
	GetByEmailFunc               func(ctx context.Context, email string) (*user.User, error)
	GetByOAuthFunc               func(ctx context.Context, provider, oauthID string) (*user.User, error)
	ListFunc                     func(ctx context.Context) ([]*user.User, error)
	UpdateFunc                   func(ctx context.Context, userID int64, params user.UpdateUserParams) (*user.User, error)
	ListUsersWithProviderKeyFunc func(ctx context.Context) ([]*user.User, error)
}

func (m *MockUserRepo) Create(ctx context.Context, params user.CreateUserParams) (*user.User, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, params)
	}
	return nil, nil
}

func (m *MockUserRepo) GetByID(ctx context.Context, id int64) (*user.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockUserRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	if m.GetByEmailFunc != nil {
		return m.GetByEmailFunc(ctx, email)
	}
	return nil, nil
}

func (m *MockUserRepo) GetByOAuth(ctx context.Context, provider, oauthID string) (*user.User, error) {
	if m.GetByOAuthFunc != nil {
		return m.GetByOAuthFunc(ctx, provider, oauthID)
	}
	return nil, nil
}

func (m *MockUserRepo) List(ctx context.Context) ([]*user.User, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx)
	}
	return nil, nil
}

func (m *MockUserRepo) Update(ctx context.Context, userID int64, params user.UpdateUserParams) (*user.User, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, userID, params)
	}
	return nil, nil
}

func (m *MockUserRepo) ListUsersWithProviderKey(ctx context.Context) ([]*user.User, error) {
	if m.ListUsersWithProviderKeyFunc != nil {
		return m.ListUsersWithProviderKeyFunc(ctx)
	}
	return nil, nil
}

func TestHandleMe_Get(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		mockRepo       func() *MockUserRepo
		expectedStatus int
	}{
		{
			name:   "Success",
			userID: 1,
			mockRepo: func() *MockUserRepo {
				return &MockUserRepo{
					GetByIDFunc: func(ctx context.Context, id int64) (*user.User, error) {
						return &user.User{ID: id, Email: "test@example.com"}, nil
					},
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "User Not Found",
			userID: 999,
			mockRepo: func() *MockUserRepo {
				return &MockUserRepo{
					GetByIDFunc: func(ctx context.Context, id int64) (*user.User, error) {
						return nil, errors.New("user not found")
					},
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mockRepo()
			handler := NewUserHandler(repo)

			req, _ := http.NewRequest(http.MethodGet, "/api/users/me", nil)
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, tt.userID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.HandleMe(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var u user.User
				json.NewDecoder(rr.Body).Decode(&u)
				if u.ID != tt.userID {
					t.Errorf("handler returned wrong user ID: got %v want %v", u.ID, tt.userID)
				}
			}
		})
	}
}

func TestHandleMe_Patch(t *testing.T) {
	newName := "New Name"
	tests := []struct {
		name           string
		userID         int64
		body           map[string]interface{}
		mockRepo       func() *MockUserRepo
		expectedStatus int
	}{
		{
			name:   "Success",
			userID: 1,
			body: map[string]interface{}{
				"name": newName,
			},
			mockRepo: func() *MockUserRepo {
				return &MockUserRepo{
					UpdateFunc: func(ctx context.Context, userID int64, params user.UpdateUserParams) (*user.User, error) {
						if userID != 1 {
							return nil, errors.New("wrong user id")
						}
						if params.Name == nil || *params.Name != newName {
							return nil, errors.New("wrong params")
						}
						return &user.User{ID: userID, Name: *params.Name, UpdatedAt: time.Now()}, nil
					},
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Invalid Body",
			userID: 1,
			body: map[string]interface{}{
				"name": 123, // Invalid type
			},
			mockRepo: func() *MockUserRepo {
				return &MockUserRepo{}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "Update Error",
			userID: 1,
			body: map[string]interface{}{
				"name": "Error Name",
			},
			mockRepo: func() *MockUserRepo {
				return &MockUserRepo{
					UpdateFunc: func(ctx context.Context, userID int64, params user.UpdateUserParams) (*user.User, error) {
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
			handler := NewUserHandler(repo)

			bodyBytes, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPatch, "/api/users/me", bytes.NewBuffer(bodyBytes))
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, tt.userID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.HandleMe(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandleMe_MethodNotAllowed(t *testing.T) {
	repo := &MockUserRepo{}
	handler := NewUserHandler(repo)

	req, _ := http.NewRequest(http.MethodDelete, "/api/users/me", nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.HandleMe(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusMethodNotAllowed)
	}
}
