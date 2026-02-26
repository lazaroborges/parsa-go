package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type DB struct {
	*sql.DB
}

func New(connStr string) (*DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{db}, nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}

// RegisterMetrics registers observable gauges for the database connection pool.
// Safe to call even when telemetry is disabled (instruments become no-ops).
func (db *DB) RegisterMetrics() {
	m := otel.Meter("parsa/db")

	openConns, _ := m.Int64ObservableGauge("db.pool.open_connections",
		metric.WithDescription("Number of open database connections"),
	)
	inUse, _ := m.Int64ObservableGauge("db.pool.in_use",
		metric.WithDescription("Number of in-use database connections"),
	)
	idle, _ := m.Int64ObservableGauge("db.pool.idle",
		metric.WithDescription("Number of idle database connections"),
	)
	waitCount, _ := m.Int64ObservableCounter("db.pool.wait_count",
		metric.WithDescription("Total wait count for database connections"),
	)

	m.RegisterCallback(
		func(_ context.Context, o metric.Observer) error {
			stats := db.Stats()
			o.ObserveInt64(openConns, int64(stats.OpenConnections))
			o.ObserveInt64(inUse, int64(stats.InUse))
			o.ObserveInt64(idle, int64(stats.Idle))
			o.ObserveInt64(waitCount, int64(stats.WaitCount))
			return nil
		},
		openConns, inUse, idle, waitCount,
	)
}
