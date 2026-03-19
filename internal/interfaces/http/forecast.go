package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"parsa/internal/domain/forecast"
	"parsa/internal/shared/middleware"
)

type ForecastHandler struct {
	forecastRepo forecast.Repository
}

func NewForecastHandler(forecastRepo forecast.Repository) *ForecastHandler {
	return &ForecastHandler{forecastRepo: forecastRepo}
}

type ForecastListResponse struct {
	Count   int                              `json:"count"`
	Results []*forecast.ForecastTransaction  `json:"results"`
}

func (h *ForecastHandler) HandleForecasts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleListForecasts(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ForecastHandler) HandleForecastByUUID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetForecast(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ForecastHandler) handleListForecasts(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	monthStr := r.URL.Query().Get("forecast_month")
	if monthStr == "" {
		http.Error(w, "forecast_month query parameter is required (format: YYYY-MM)", http.StatusBadRequest)
		return
	}

	forecastMonth, err := time.Parse("2006-01", monthStr)
	if err != nil {
		http.Error(w, "Invalid forecast_month format (use YYYY-MM)", http.StatusBadRequest)
		return
	}

	forecasts, err := h.forecastRepo.ListByMonth(r.Context(), userID, forecastMonth)
	if err != nil {
		log.Printf("Error listing forecasts for user %d: %v", userID, err)
		http.Error(w, "Failed to list forecasts", http.StatusInternalServerError)
		return
	}

	results := forecasts
	if results == nil {
		results = make([]*forecast.ForecastTransaction, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ForecastListResponse{
		Count:   len(results),
		Results: results,
	})
}

func (h *ForecastHandler) handleGetForecast(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	uuid := r.PathValue("uuid")
	if uuid == "" {
		http.Error(w, "Forecast UUID is required", http.StatusBadRequest)
		return
	}

	f, err := h.forecastRepo.GetByUUID(r.Context(), uuid, userID)
	if err != nil {
		if errors.Is(err, forecast.ErrForecastNotFound) {
			http.Error(w, "Forecast not found", http.StatusNotFound)
			return
		}
		log.Printf("Error getting forecast %s for user %d: %v", uuid, userID, err)
		http.Error(w, "Failed to get forecast", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(f)
}
