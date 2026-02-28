package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unicode"

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
		attribute.String("db.operation", extractSQLVerb(query)),
		attribute.String("db.statement", sanitizeQuery(query)),
	))
	defer span.End()

	rows, err := db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return rows, err
}

// tracedRow wraps *sql.Row so the tracing span stays open until Scan() is
// called, which is where sql.Row surfaces all errors (including sql.ErrNoRows).
type tracedRow struct {
	row  *sql.Row
	span trace.Span
}

func (r *tracedRow) Scan(dest ...any) error {
	err := r.row.Scan(dest...)
	if r.span != nil {
		if err != nil {
			r.span.RecordError(err)
			r.span.SetStatus(codes.Error, err.Error())
		}
		r.span.End()
		r.span = nil
	}
	return err
}

// QueryRowContext wraps sql.DB.QueryRowContext with tracing.
// The returned tracedRow ends the span in Scan(), not here, because
// sql.Row defers all errors to Scan().
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...any) *tracedRow {
	ctx, span := dbTracer.Start(ctx, "db.QueryRow", trace.WithAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", extractSQLVerb(query)),
		attribute.String("db.statement", sanitizeQuery(query)),
	))

	return &tracedRow{
		row:  db.DB.QueryRowContext(ctx, query, args...),
		span: span,
	}
}

// ExecContext wraps sql.DB.ExecContext with tracing.
func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	ctx, span := dbTracer.Start(ctx, "db.Exec", trace.WithAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", extractSQLVerb(query)),
		attribute.String("db.statement", sanitizeQuery(query)),
	))
	defer span.End()

	result, err := db.DB.ExecContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return result, err
}

// sanitizeQuery replaces string literals and bare numeric literals with '?'
// so that sensitive values (PII, tokens, etc.) are never stored in traces.
// Parameterized queries using $1, $2, ... are left as-is since they carry no data.
func sanitizeQuery(q string) string {
	var b strings.Builder
	b.Grow(len(q))

	i := 0
	for i < len(q) {
		ch := q[i]

		// Replace quoted string literals: 'value' → '?'
		if ch == '\'' {
			b.WriteString("'?'")
			i++
			for i < len(q) {
				if q[i] == '\'' {
					if i+1 < len(q) && q[i+1] == '\'' {
						i += 2 // escaped quote ''
						continue
					}
					i++ // closing quote
					break
				}
				i++
			}
			continue
		}

		// Replace bare numeric literals that aren't $N parameters
		if unicode.IsDigit(rune(ch)) && (i == 0 || !isIdentChar(q[i-1])) {
			// Check it's not a $N placeholder
			if i > 0 && q[i-1] == '$' {
				b.WriteByte(ch)
				i++
				continue
			}
			b.WriteByte('?')
			for i < len(q) && (unicode.IsDigit(rune(q[i])) || q[i] == '.') {
				i++
			}
			continue
		}

		b.WriteByte(ch)
		i++
	}

	s := b.String()
	if len(s) > 256 {
		return s[:256] + "..."
	}
	return s
}

func isIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c == '$'
}

func extractSQLVerb(q string) string {
	q = strings.TrimSpace(q)
	if idx := strings.IndexByte(q, ' '); idx > 0 {
		return strings.ToUpper(q[:idx])
	}
	return strings.ToUpper(q)
}
