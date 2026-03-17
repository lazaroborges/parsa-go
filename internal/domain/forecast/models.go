package forecast

import (
	"errors"
	"time"
)

var ErrForecastNotFound = errors.New("forecast not found")

type ForecastTransaction struct {
	UUID                string     `json:"id"`
	UserID              int64      `json:"-"`
	RecurrencyPatternID *int64     `json:"recurrencyPatternId,omitempty"`
	Type                string     `json:"type"`
	RecurrencyType      string     `json:"recurrencyType"`
	ForecastAmount      float64    `json:"forecastAmount"`
	ForecastLow         *float64   `json:"forecastLow,omitempty"`
	ForecastHigh        *float64   `json:"forecastHigh,omitempty"`
	ForecastDate        *time.Time `json:"forecastDate,omitempty"`
	ForecastMonth       time.Time  `json:"forecastMonth"`
	CousinID            *int64     `json:"cousin,omitempty"`
	CousinName          *string    `json:"cousinName,omitempty"`
	Category            *string    `json:"category,omitempty"`
	Description         *string    `json:"description,omitempty"`
	AccountID           string     `json:"accountId"`
}
