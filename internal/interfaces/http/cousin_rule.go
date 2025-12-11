package http

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"parsa/internal/domain/cousinrule"
	"parsa/internal/shared/middleware"
)

type CousinRuleHandler struct {
	cousinRuleService *cousinrule.Service
}

func NewCousinRuleHandler(cousinRuleService *cousinrule.Service) *CousinRuleHandler {
	return &CousinRuleHandler{
		cousinRuleService: cousinRuleService,
	}
}

// ApplyRuleRequest is the request body for applying a cousin rule
type ApplyRuleRequest struct {
	CousinID     int64                  `json:"cousinId"`
	TriggeringID string                 `json:"triggeringId,omitempty"`
	CreateRule   bool                   `json:"createRule"`
	DontAskAgain bool                   `json:"dontAskAgain"`
	Changes      ApplyRuleChangeRequest `json:"changes"`
}

// ApplyRuleChangeRequest contains the changes to apply
type ApplyRuleChangeRequest struct {
	Category    *string   `json:"category,omitempty"`
	Description *string   `json:"description,omitempty"`
	Notes       *string   `json:"notes,omitempty"`
	Status      *string   `json:"status,omitempty"`     // "reconciled" or other - maps to considered
	Considered  *bool     `json:"considered,omitempty"` // Direct boolean, takes precedence over status
	Tags        *[]string `json:"tags,omitempty"`       // nil = don't change, empty = clear all
}

// ApplyRuleResponse is the response for applying a cousin rule
type ApplyRuleResponse struct {
	Message             string `json:"message"`
	TransactionsUpdated int    `json:"transactionsUpdated"`
	RuleCreated         bool   `json:"ruleCreated"`
	RuleUpdated         bool   `json:"ruleUpdated"`
}

// CousinRuleAPIResponse is the API response format for a cousin rule
type CousinRuleAPIResponse struct {
	ID           int64    `json:"id"`
	CousinID     int64    `json:"cousinId"`
	Type         *string  `json:"type,omitempty"`
	Category     *string  `json:"category,omitempty"`
	Description  *string  `json:"description,omitempty"`
	Notes        *string  `json:"notes,omitempty"`
	Considered   *bool    `json:"considered,omitempty"`
	DontAskAgain bool     `json:"dontAskAgain"`
	Tags         []string `json:"tags"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    string   `json:"updatedAt"`
}

// HandleCousinRules handles POST /api/cousin-rules/ for applying rules
func (h *CousinRuleHandler) HandleCousinRules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handleApplyRule(w, r)
	case http.MethodGet:
		h.handleListRules(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleCousinRuleByID handles GET/DELETE /api/cousin-rules/{id}
func (h *CousinRuleHandler) HandleCousinRuleByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract rule ID from path
	idStr := r.PathValue("id")
	if idStr == "" {
		// Fallback for older Go versions
		idStr = strings.TrimPrefix(r.URL.Path, "/api/cousin-rules/")
	}

	ruleID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid rule ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGetRule(w, r, ruleID, userID)
	case http.MethodDelete:
		h.handleDeleteRule(w, r, ruleID, userID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleApplyRule handles POST /api/cousin-rules/ - apply rule to transactions
func (h *CousinRuleHandler) handleApplyRule(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req ApplyRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding apply rule request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CousinID == 0 {
		http.Error(w, "cousinId is required", http.StatusBadRequest)
		return
	}

	// Convert status to considered boolean (direct considered takes precedence)
	var considered *bool
	if req.Changes.Considered != nil {
		considered = req.Changes.Considered
	} else if req.Changes.Status != nil {
		isConsidered := *req.Changes.Status == "reconciled"
		considered = &isConsidered
	}

	params := cousinrule.ApplyRuleParams{
		CousinID:     req.CousinID,
		TriggeringID: req.TriggeringID,
		CreateRule:   req.CreateRule,
		DontAskAgain: req.DontAskAgain,
		Changes: cousinrule.Changes{
			Category:    req.Changes.Category,
			Description: req.Changes.Description,
			Notes:       req.Changes.Notes,
			Considered:  considered,
			Tags:        req.Changes.Tags,
		},
	}

	result, err := h.cousinRuleService.ApplyRule(r.Context(), userID, params)
	if err != nil {
		if err == cousinrule.ErrNoChanges {
			http.Error(w, "No changes provided", http.StatusBadRequest)
			return
		}
		log.Printf("Error applying cousin rule for user %d: %v", userID, err)
		http.Error(w, "Failed to apply rule", http.StatusInternalServerError)
		return
	}

	response := ApplyRuleResponse{
		Message:             "Rules applied successfully",
		TransactionsUpdated: result.TransactionsUpdated,
		RuleCreated:         result.RuleCreated,
		RuleUpdated:         result.RuleUpdated,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleListRules handles GET /api/cousin-rules/ - list all rules for user
func (h *CousinRuleHandler) handleListRules(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rules, err := h.cousinRuleService.ListRulesByUser(r.Context(), userID)
	if err != nil {
		log.Printf("Error listing cousin rules for user %d: %v", userID, err)
		http.Error(w, "Failed to list rules", http.StatusInternalServerError)
		return
	}

	results := make([]CousinRuleAPIResponse, 0, len(rules))
	for _, rule := range rules {
		results = append(results, toCousinRuleAPIResponse(rule))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// handleGetRule handles GET /api/cousin-rules/{id}
func (h *CousinRuleHandler) handleGetRule(w http.ResponseWriter, r *http.Request, ruleID, userID int64) {
	rule, err := h.cousinRuleService.GetRule(r.Context(), ruleID, userID)
	if err != nil {
		if err == cousinrule.ErrRuleNotFound {
			http.Error(w, "Rule not found", http.StatusNotFound)
			return
		}
		if err == cousinrule.ErrForbidden {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		log.Printf("Error getting cousin rule %d for user %d: %v", ruleID, userID, err)
		http.Error(w, "Failed to get rule", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toCousinRuleAPIResponse(rule))
}

// handleDeleteRule handles DELETE /api/cousin-rules/{id}
func (h *CousinRuleHandler) handleDeleteRule(w http.ResponseWriter, r *http.Request, ruleID, userID int64) {
	err := h.cousinRuleService.DeleteRule(r.Context(), ruleID, userID)
	if err != nil {
		if err == cousinrule.ErrRuleNotFound {
			http.Error(w, "Rule not found", http.StatusNotFound)
			return
		}
		if err == cousinrule.ErrForbidden {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		log.Printf("Error deleting cousin rule %d for user %d: %v", ruleID, userID, err)
		http.Error(w, "Failed to delete rule", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toCousinRuleAPIResponse(rule *cousinrule.CousinRule) CousinRuleAPIResponse {
	tags := rule.Tags
	if tags == nil {
		tags = []string{}
	}

	return CousinRuleAPIResponse{
		ID:           rule.ID,
		CousinID:     rule.CousinID,
		Type:         rule.Type,
		Category:     rule.Category,
		Description:  rule.Description,
		Notes:        rule.Notes,
		Considered:   rule.Considered,
		DontAskAgain: rule.DontAskAgain,
		Tags:         tags,
		CreatedAt:    rule.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    rule.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
