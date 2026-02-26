package http

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"parsa/internal/domain/notification"
	"parsa/internal/shared/middleware"
)

type NotificationHandler struct {
	notificationService *notification.Service
}

func NewNotificationHandler(notificationService *notification.Service) *NotificationHandler {
	return &NotificationHandler{notificationService: notificationService}
}

// --- Request/Response types ---

type RegisterDeviceRequest struct {
	Token      string `json:"token"`
	DeviceType string `json:"device_type"`
}

type UpdatePreferencesRequest struct {
	BudgetsEnabled      *bool `json:"budgets_enabled"`
	GeneralEnabled      *bool `json:"general_enabled"`
	AccountsEnabled     *bool `json:"accounts_enabled"`
	TransactionsEnabled *bool `json:"transactions_enabled"`
}

type PreferencesResponse struct {
	Success bool                    `json:"success"`
	Data    *PreferencesDataResponse `json:"data"`
}

type PreferencesDataResponse struct {
	BudgetsEnabled      bool `json:"budgets_enabled"`
	GeneralEnabled      bool `json:"general_enabled"`
	AccountsEnabled     bool `json:"accounts_enabled"`
	TransactionsEnabled bool `json:"transactions_enabled"`
}

type NotificationResponse struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Message   string            `json:"message"`
	Category  string            `json:"category"`
	OpenedAt  *string           `json:"opened_at"`
	CreatedAt string            `json:"created_at"`
	Data      map[string]string `json:"data"`
}

type NotificationListResponse struct {
	Notifications []NotificationResponse `json:"notifications"`
	Pagination    PaginationResponse     `json:"pagination"`
}

type PaginationResponse struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
	Pages   int `json:"pages"`
}

type OpenNotificationRequest struct {
	NotificationID string `json:"notification_id"`
}

const maxNotificationBodySize = 1 << 20 // 1 MiB

// --- Handlers ---

// HandleNotifications handles GET /api/notifications/ (list)
func (h *NotificationHandler) HandleNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 {
		perPage = 20
	}

	notifications, total, err := h.notificationService.ListNotifications(r.Context(), userID, page, perPage)
	if err != nil {
		log.Printf("Error listing notifications for user %d: %v", userID, err)
		http.Error(w, "Failed to list notifications", http.StatusInternalServerError)
		return
	}

	items := make([]NotificationResponse, 0, len(notifications))
	for _, n := range notifications {
		items = append(items, toNotificationResponse(n))
	}

	pages := 0
	if total > 0 {
		pages = (total + perPage - 1) / perPage
	}

	resp := NotificationListResponse{
		Notifications: items,
		Pagination: PaginationResponse{
			Page:    page,
			PerPage: perPage,
			Total:   total,
			Pages:   pages,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleNotificationByID handles PUT/DELETE /api/notifications/{id}
func (h *NotificationHandler) HandleNotificationByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	notificationID := r.PathValue("id")
	if notificationID == "" {
		http.Error(w, "Notification ID is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		// Mark as opened
		if err := h.notificationService.MarkNotificationOpened(r.Context(), notificationID, userID); err != nil {
			if err == notification.ErrNotificationNotFound {
				http.Error(w, "Notification not found", http.StatusNotFound)
				return
			}
			log.Printf("Error marking notification %s as opened: %v", notificationID, err)
			http.Error(w, "Failed to update notification", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandlePreferences handles GET/POST /api/notifications/preferences/
func (h *NotificationHandler) HandlePreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGetPreferences(w, r, userID)
	case http.MethodPost:
		h.handleUpdatePreferences(w, r, userID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *NotificationHandler) handleGetPreferences(w http.ResponseWriter, r *http.Request, userID int64) {
	prefs, err := h.notificationService.GetPreferences(r.Context(), userID)
	if err != nil {
		log.Printf("Error getting preferences for user %d: %v", userID, err)
		http.Error(w, "Failed to get preferences", http.StatusInternalServerError)
		return
	}

	resp := PreferencesResponse{
		Success: true,
		Data: &PreferencesDataResponse{
			BudgetsEnabled:      prefs.BudgetsEnabled,
			GeneralEnabled:      prefs.GeneralEnabled,
			AccountsEnabled:     prefs.AccountsEnabled,
			TransactionsEnabled: prefs.TransactionsEnabled,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *NotificationHandler) handleUpdatePreferences(w http.ResponseWriter, r *http.Request, userID int64) {
	r.Body = http.MaxBytesReader(w, r.Body, maxNotificationBodySize)
	var req UpdatePreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	params := notification.UpdatePreferenceParams{
		BudgetsEnabled:      req.BudgetsEnabled,
		GeneralEnabled:      req.GeneralEnabled,
		AccountsEnabled:     req.AccountsEnabled,
		TransactionsEnabled: req.TransactionsEnabled,
	}

	prefs, err := h.notificationService.UpdatePreferences(r.Context(), userID, params)
	if err != nil {
		log.Printf("Error updating preferences for user %d: %v", userID, err)
		http.Error(w, "Failed to update preferences", http.StatusInternalServerError)
		return
	}

	resp := PreferencesResponse{
		Success: true,
		Data: &PreferencesDataResponse{
			BudgetsEnabled:      prefs.BudgetsEnabled,
			GeneralEnabled:      prefs.GeneralEnabled,
			AccountsEnabled:     prefs.AccountsEnabled,
			TransactionsEnabled: prefs.TransactionsEnabled,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleRegisterDevice handles POST /api/notifications/register-device/
func (h *NotificationHandler) HandleRegisterDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxNotificationBodySize)
	var req RegisterDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	params := notification.CreateDeviceTokenParams{
		UserID:     userID,
		Token:      req.Token,
		DeviceType: req.DeviceType,
	}

	token, err := h.notificationService.RegisterDevice(r.Context(), params)
	if err != nil {
		if err == notification.ErrInvalidToken || err == notification.ErrInvalidDeviceType {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("Error registering device for user %d: %v", userID, err)
		http.Error(w, "Failed to register device", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"token":   token.Token,
	})
}

// HandleOpen handles POST /api/notifications/open/
func (h *NotificationHandler) HandleOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxNotificationBodySize)
	var req OpenNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.NotificationID == "" {
		http.Error(w, "notification_id is required", http.StatusBadRequest)
		return
	}

	if err := h.notificationService.MarkNotificationOpened(r.Context(), req.NotificationID, userID); err != nil {
		if err == notification.ErrNotificationNotFound {
			http.Error(w, "Notification not found", http.StatusNotFound)
			return
		}
		log.Printf("Error marking notification %s as opened: %v", req.NotificationID, err)
		http.Error(w, "Failed to mark notification as opened", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// --- Helpers ---

func toNotificationResponse(n *notification.Notification) NotificationResponse {
	var openedAt *string
	if n.OpenedAt != nil {
		formatted := n.OpenedAt.Format("2006-01-02T15:04:05Z07:00")
		openedAt = &formatted
	}

	data := n.Data
	if data == nil {
		data = make(map[string]string)
	}

	return NotificationResponse{
		ID:        n.ID,
		Title:     n.Title,
		Message:   n.Message,
		Category:  n.Category,
		OpenedAt:  openedAt,
		CreatedAt: n.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Data:      data,
	}
}
