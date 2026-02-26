package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"parsa/internal/domain/tag"
	"parsa/internal/shared/middleware"
)

// MockTagRepo implements tag.Repository for testing
type MockTagRepo struct {
	CreateFunc     func(ctx context.Context, userID int64, params tag.CreateTagParams) (*tag.Tag, error)
	GetByIDFunc    func(ctx context.Context, id string) (*tag.Tag, error)
	ListByUserIDFunc func(ctx context.Context, userID int64) ([]*tag.Tag, error)
	UpdateFunc     func(ctx context.Context, id string, params tag.UpdateTagParams) (*tag.Tag, error)
	DeleteFunc     func(ctx context.Context, id string) error
}

func (m *MockTagRepo) Create(ctx context.Context, userID int64, params tag.CreateTagParams) (*tag.Tag, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, userID, params)
	}
	return nil, nil
}

func (m *MockTagRepo) GetByID(ctx context.Context, id string) (*tag.Tag, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockTagRepo) ListByUserID(ctx context.Context, userID int64) ([]*tag.Tag, error) {
	if m.ListByUserIDFunc != nil {
		return m.ListByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockTagRepo) Update(ctx context.Context, id string, params tag.UpdateTagParams) (*tag.Tag, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, id, params)
	}
	return nil, nil
}

func (m *MockTagRepo) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func TestHandleTags_ListTags(t *testing.T) {
	tests := []struct {
		name           string
		mockRepo       func() *MockTagRepo
		expectedStatus int
		expectedLen    int
	}{
		{
			name: "Success",
			mockRepo: func() *MockTagRepo {
				return &MockTagRepo{
					ListByUserIDFunc: func(ctx context.Context, userID int64) ([]*tag.Tag, error) {
						return []*tag.Tag{
							{ID: "tag-1", Name: "Work", Color: "#FF0000"},
							{ID: "tag-2", Name: "Personal", Color: "#00FF00"},
						}, nil
					},
				}
			},
			expectedStatus: http.StatusOK,
			expectedLen:    2,
		},
		{
			name: "Empty List",
			mockRepo: func() *MockTagRepo {
				return &MockTagRepo{
					ListByUserIDFunc: func(ctx context.Context, userID int64) ([]*tag.Tag, error) {
						return []*tag.Tag{}, nil
					},
				}
			},
			expectedStatus: http.StatusOK,
			expectedLen:    0,
		},
		{
			name: "Repository Error",
			mockRepo: func() *MockTagRepo {
				return &MockTagRepo{
					ListByUserIDFunc: func(ctx context.Context, userID int64) ([]*tag.Tag, error) {
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
			handler := NewTagHandler(repo)

			req, _ := http.NewRequest(http.MethodGet, "/api/tags/", nil)
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.HandleTags(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var tags []TagResponse
				json.NewDecoder(rr.Body).Decode(&tags)
				if len(tags) != tt.expectedLen {
					t.Errorf("response length = %d, want %d", len(tags), tt.expectedLen)
				}
			}
		})
	}
}

func TestHandleTags_CreateTag(t *testing.T) {
	tests := []struct {
		name           string
		body           map[string]interface{}
		mockRepo       func() *MockTagRepo
		expectedStatus int
	}{
		{
			name: "Success",
			body: map[string]interface{}{
				"name":  "Work",
				"color": "#FF0000",
			},
			mockRepo: func() *MockTagRepo {
				return &MockTagRepo{
					CreateFunc: func(ctx context.Context, userID int64, params tag.CreateTagParams) (*tag.Tag, error) {
						return &tag.Tag{ID: "tag-1", Name: params.Name, Color: params.Color, UserID: userID}, nil
					},
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Missing Name",
			body: map[string]interface{}{
				"color": "#FF0000",
			},
			mockRepo: func() *MockTagRepo {
				return &MockTagRepo{}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing Color",
			body: map[string]interface{}{
				"name": "Work",
			},
			mockRepo: func() *MockTagRepo {
				return &MockTagRepo{}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid JSON",
			body: nil,
			mockRepo: func() *MockTagRepo {
				return &MockTagRepo{}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Repository Error",
			body: map[string]interface{}{
				"name":  "Work",
				"color": "#FF0000",
			},
			mockRepo: func() *MockTagRepo {
				return &MockTagRepo{
					CreateFunc: func(ctx context.Context, userID int64, params tag.CreateTagParams) (*tag.Tag, error) {
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
			handler := NewTagHandler(repo)

			var body *bytes.Buffer
			if tt.body != nil {
				bodyBytes, _ := json.Marshal(tt.body)
				body = bytes.NewBuffer(bodyBytes)
			} else {
				body = bytes.NewBuffer([]byte("invalid json{"))
			}

			req, _ := http.NewRequest(http.MethodPost, "/api/tags/", body)
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.HandleTags(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandleTags_MethodNotAllowed(t *testing.T) {
	handler := NewTagHandler(&MockTagRepo{})

	req, _ := http.NewRequest(http.MethodDelete, "/api/tags/", nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.HandleTags(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleTagByID_UpdateTag(t *testing.T) {
	tests := []struct {
		name           string
		tagID          string
		body           map[string]interface{}
		mockRepo       func() *MockTagRepo
		expectedStatus int
	}{
		{
			name:  "Success",
			tagID: "tag-1",
			body: map[string]interface{}{
				"name": "Updated",
			},
			mockRepo: func() *MockTagRepo {
				return &MockTagRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*tag.Tag, error) {
						return &tag.Tag{ID: id, UserID: 1}, nil
					},
					UpdateFunc: func(ctx context.Context, id string, params tag.UpdateTagParams) (*tag.Tag, error) {
						return &tag.Tag{ID: id, Name: *params.Name, Color: "#FF0000", UserID: 1}, nil
					},
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:  "Tag Not Found",
			tagID: "tag-999",
			body: map[string]interface{}{
				"name": "Updated",
			},
			mockRepo: func() *MockTagRepo {
				return &MockTagRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*tag.Tag, error) {
						return nil, nil
					},
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:  "Forbidden",
			tagID: "tag-1",
			body: map[string]interface{}{
				"name": "Updated",
			},
			mockRepo: func() *MockTagRepo {
				return &MockTagRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*tag.Tag, error) {
						return &tag.Tag{ID: id, UserID: 2}, nil // different user
					},
				}
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mockRepo()
			handler := NewTagHandler(repo)

			bodyBytes, _ := json.Marshal(tt.body)

			mux := http.NewServeMux()
			mux.HandleFunc("PUT /api/tags/{id}", handler.HandleTagByID)

			req, _ := http.NewRequest(http.MethodPut, "/api/tags/"+tt.tagID, bytes.NewBuffer(bodyBytes))
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandleTagByID_DeleteTag(t *testing.T) {
	tests := []struct {
		name           string
		tagID          string
		mockRepo       func() *MockTagRepo
		expectedStatus int
	}{
		{
			name:  "Success",
			tagID: "tag-1",
			mockRepo: func() *MockTagRepo {
				return &MockTagRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*tag.Tag, error) {
						return &tag.Tag{ID: id, UserID: 1}, nil
					},
					DeleteFunc: func(ctx context.Context, id string) error {
						return nil
					},
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:  "Tag Not Found",
			tagID: "tag-999",
			mockRepo: func() *MockTagRepo {
				return &MockTagRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*tag.Tag, error) {
						return nil, nil
					},
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:  "Forbidden",
			tagID: "tag-1",
			mockRepo: func() *MockTagRepo {
				return &MockTagRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*tag.Tag, error) {
						return &tag.Tag{ID: id, UserID: 2}, nil
					},
				}
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mockRepo()
			handler := NewTagHandler(repo)

			mux := http.NewServeMux()
			mux.HandleFunc("DELETE /api/tags/{id}", handler.HandleTagByID)

			req, _ := http.NewRequest(http.MethodDelete, "/api/tags/"+tt.tagID, nil)
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandleTagByID_MethodNotAllowed(t *testing.T) {
	handler := NewTagHandler(&MockTagRepo{})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/tags/{id}", handler.HandleTagByID)
	mux.HandleFunc("PUT /api/tags/{id}", handler.HandleTagByID)
	mux.HandleFunc("DELETE /api/tags/{id}", handler.HandleTagByID)
	mux.HandleFunc("PATCH /api/tags/{id}", handler.HandleTagByID)

	req, _ := http.NewRequest(http.MethodPatch, "/api/tags/tag-1", nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, int64(1))
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}
