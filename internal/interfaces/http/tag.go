package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"parsa/internal/domain/tag"
	"parsa/internal/shared/middleware"
)

type TagHandler struct {
	tagRepo tag.Repository
}

func NewTagHandler(tagRepo tag.Repository) *TagHandler {
	return &TagHandler{tagRepo: tagRepo}
}

// Request/Response DTOs

type CreateTagRequest struct {
	Name         string  `json:"name"`
	Color        string  `json:"color"`
	DisplayOrder *int    `json:"displayOrder,omitempty"`
	Description  *string `json:"description,omitempty"`
}

type UpdateTagRequest struct {
	Name         *string `json:"name,omitempty"`
	Color        *string `json:"color,omitempty"`
	DisplayOrder *int    `json:"displayOrder,omitempty"`
	Description  *string `json:"description,omitempty"`
}

type TagResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Color        string `json:"color"`
	DisplayOrder int    `json:"displayOrder"`
	Description  string `json:"description"`
}

func toTagResponse(t *tag.Tag) TagResponse {
	return TagResponse{
		ID:           t.ID,
		Name:         t.Name,
		Color:        t.Color,
		DisplayOrder: t.DisplayOrder,
		Description:  t.Description,
	}
}

// HandleTags routes requests to the appropriate handler based on method
func (h *TagHandler) HandleTags(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleListTags(w, r)
	case http.MethodPost:
		h.handleCreateTag(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleTagByID routes requests for a specific tag
func (h *TagHandler) HandleTagByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		h.handleUpdateTag(w, r)
	case http.MethodDelete:
		h.handleDeleteTag(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListTags returns all tags for the authenticated user
func (h *TagHandler) handleListTags(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tags, err := h.tagRepo.ListByUserID(r.Context(), userID)
	if err != nil {
		log.Printf("Error listing tags for user %d: %v", userID, err)
		http.Error(w, "Failed to list tags", http.StatusInternalServerError)
		return
	}

	response := make([]TagResponse, 0, len(tags))
	for _, t := range tags {
		response = append(response, toTagResponse(t))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCreateTag creates a new tag
func (h *TagHandler) handleCreateTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding create tag request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	params := tag.CreateTagParams{
		Name:         req.Name,
		Color:        req.Color,
		DisplayOrder: req.DisplayOrder,
		Description:  req.Description,
	}

	if err := params.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	t, err := h.tagRepo.Create(r.Context(), userID, params)
	if err != nil {
		log.Printf("Error creating tag for user %d: %v", userID, err)
		http.Error(w, "Failed to create tag", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toTagResponse(t))
}

// handleUpdateTag updates an existing tag
func (h *TagHandler) handleUpdateTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tagID := r.PathValue("id")
	if tagID == "" {
		http.Error(w, "Tag ID is required", http.StatusBadRequest)
		return
	}

	// Verify tag exists and belongs to user
	existingTag, err := h.tagRepo.GetByID(r.Context(), tagID)
	if err != nil {
		log.Printf("Error getting tag %s: %v", tagID, err)
		http.Error(w, "Failed to get tag", http.StatusInternalServerError)
		return
	}
	if existingTag == nil {
		http.Error(w, "Tag not found", http.StatusNotFound)
		return
	}
	if existingTag.UserID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req UpdateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding update tag request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	params := tag.UpdateTagParams{
		Name:         req.Name,
		Color:        req.Color,
		DisplayOrder: req.DisplayOrder,
		Description:  req.Description,
	}

	if err := params.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	t, err := h.tagRepo.Update(r.Context(), tagID, params)
	if err != nil {
		if errors.Is(err, tag.ErrTagNotFound) {
			http.Error(w, "Tag not found", http.StatusNotFound)
			return
		}
		log.Printf("Error updating tag %s: %v", tagID, err)
		http.Error(w, "Failed to update tag", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toTagResponse(t))
}

// handleDeleteTag deletes a tag
func (h *TagHandler) handleDeleteTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tagID := r.PathValue("id")
	if tagID == "" {
		http.Error(w, "Tag ID is required", http.StatusBadRequest)
		return
	}

	// Verify tag exists and belongs to user
	existingTag, err := h.tagRepo.GetByID(r.Context(), tagID)
	if err != nil {
		log.Printf("Error getting tag %s for deletion: %v", tagID, err)
		http.Error(w, "Failed to get tag", http.StatusInternalServerError)
		return
	}
	if existingTag == nil {
		http.Error(w, "Tag not found", http.StatusNotFound)
		return
	}
	if existingTag.UserID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.tagRepo.Delete(r.Context(), tagID); err != nil {
		if errors.Is(err, tag.ErrTagNotFound) {
			http.Error(w, "Tag not found", http.StatusNotFound)
			return
		}
		log.Printf("Error deleting tag %s: %v", tagID, err)
		http.Error(w, "Failed to delete tag", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
