package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var dbTracer = otel.Tracer("parsa.db")

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

// QueryContext wraps sql.DB.QueryContext with tracing.
func (db *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	ctx, span := dbTracer.Start(ctx, "db.Query", trace.WithAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.statement", truncateQuery(query)),
	))
	defer span.End()

	rows, err := db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return rows, err
}

// QueryRowContext wraps sql.DB.QueryRowContext with tracing.
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	ctx, span := dbTracer.Start(ctx, "db.QueryRow", trace.WithAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.statement", truncateQuery(query)),
	))
	defer span.End()

	return db.DB.QueryRowContext(ctx, query, args...)
}

// ExecContext wraps sql.DB.ExecContext with tracing.
func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	ctx, span := dbTracer.Start(ctx, "db.Exec", trace.WithAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.statement", truncateQuery(query)),
	))
	defer span.End()

	result, err := db.DB.ExecContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return result, err
}

func truncateQuery(q string) string {
	if len(q) > 256 {
		return q[:256] + "..."
	}
	return q
}
