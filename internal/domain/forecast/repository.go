package forecast

import (
	"context"
	"time"
)

type Repository interface {
	GetByUUID(ctx context.Context, uuid string, userID int64) (*ForecastTransaction, error)
	ListByUserID(ctx context.Context, userID int64) ([]*ForecastTransaction, error)
	ListByMonth(ctx context.Context, userID int64, forecastMonth time.Time) ([]*ForecastTransaction, error)
}
