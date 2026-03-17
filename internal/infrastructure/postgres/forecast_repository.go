package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"parsa/internal/domain/forecast"
)

type ForecastRepository struct {
	db *DB
}

func NewForecastRepository(db *DB) *ForecastRepository {
	return &ForecastRepository{db: db}
}

type scanner interface {
	Scan(dest ...any) error
}

func scanForecastTransaction(s scanner) (*forecast.ForecastTransaction, error) {
	var f forecast.ForecastTransaction
	var recurrencyPatternID, cousinID sql.NullInt64
	var forecastLow, forecastHigh sql.NullFloat64
	var forecastDate sql.NullTime
	var cousinName, category, description sql.NullString

	err := s.Scan(
		&f.UUID, &f.UserID, &recurrencyPatternID,
		&f.Type, &f.RecurrencyType,
		&f.ForecastAmount, &forecastLow, &forecastHigh,
		&forecastDate, &f.ForecastMonth,
		&cousinID, &cousinName, &category, &description,
		&f.AccountID,
	)
	if err != nil {
		return nil, err
	}

	if recurrencyPatternID.Valid {
		f.RecurrencyPatternID = &recurrencyPatternID.Int64
	}
	if forecastLow.Valid {
		f.ForecastLow = &forecastLow.Float64
	}
	if forecastHigh.Valid {
		f.ForecastHigh = &forecastHigh.Float64
	}
	if forecastDate.Valid {
		f.ForecastDate = &forecastDate.Time
	}
	if cousinID.Valid {
		f.CousinID = &cousinID.Int64
	}
	if cousinName.Valid {
		f.CousinName = &cousinName.String
	}
	if category.Valid {
		f.Category = &category.String
	}
	if description.Valid {
		f.Description = &description.String
	}

	return &f, nil
}

const forecastColumns = `uuid, user_id, recurrency_pattern_id, type, recurrency_type,
	forecast_amount, forecast_low, forecast_high, forecast_date, forecast_month,
	cousin_id, cousin_name, category, description, account_id`

func (r *ForecastRepository) GetByUUID(ctx context.Context, uuid string, userID int64) (*forecast.ForecastTransaction, error) {
	query := fmt.Sprintf(`SELECT %s FROM forecast_transactions WHERE uuid = $1 AND user_id = $2`, forecastColumns)

	f, err := scanForecastTransaction(r.db.QueryRowContext(ctx, query, uuid, userID))
	if err == sql.ErrNoRows {
		return nil, forecast.ErrForecastNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get forecast: %w", err)
	}

	return f, nil
}

func (r *ForecastRepository) ListByMonth(ctx context.Context, userID int64, forecastMonth time.Time) ([]*forecast.ForecastTransaction, error) {
	query := fmt.Sprintf(`SELECT %s FROM forecast_transactions
		WHERE user_id = $1 AND forecast_month = $2
		ORDER BY forecast_amount DESC`, forecastColumns)

	rows, err := r.db.QueryContext(ctx, query, userID, forecastMonth)
	if err != nil {
		return nil, fmt.Errorf("failed to list forecasts: %w", err)
	}
	defer rows.Close()

	var forecasts []*forecast.ForecastTransaction
	for rows.Next() {
		f, err := scanForecastTransaction(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan forecast: %w", err)
		}
		forecasts = append(forecasts, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating forecasts: %w", err)
	}

	return forecasts, nil
}
