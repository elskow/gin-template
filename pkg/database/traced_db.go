package database

import (
	"context"
	"database/sql"
	"strings"

	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("database")

const (
	maxQueryLogLength   = 1000
	queryTruncatedLabel = "..."
)

var dbSystemAttr = attribute.String("db.system", "postgresql")

type TracedDB struct {
	*sqlx.DB
	logQueries bool
}

func NewTracedDB(db *sqlx.DB) *TracedDB {
	return &TracedDB{
		DB:         db,
		logQueries: true,
	}
}

func (db *TracedDB) startSpan(ctx context.Context, operation, query string) (context.Context, trace.Span) {
	ctx, span := tracer.Start(ctx, operation,
		trace.WithSpanKind(trace.SpanKindClient),
	)

	span.SetAttributes(dbSystemAttr)

	if db.logQueries {
		queryLen := len(query)
		if queryLen < maxQueryLogLength {
			span.SetAttributes(attribute.String("db.statement", query))
		} else {
			var builder strings.Builder
			builder.Grow(maxQueryLogLength + len(queryTruncatedLabel))
			builder.WriteString(query[:maxQueryLogLength])
			builder.WriteString(queryTruncatedLabel)
			span.SetAttributes(attribute.String("db.statement", builder.String()))
		}
	}

	return ctx, span
}

func (db *TracedDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ctx, span := db.startSpan(ctx, "db.exec", query)
	defer span.End()

	result, err := db.DB.ExecContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return result, err
}

func (db *TracedDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	ctx, span := db.startSpan(ctx, "db.query", query)
	defer span.End()

	rows, err := db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return rows, err
}

func (db *TracedDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	ctx, span := db.startSpan(ctx, "db.query_row", query)
	defer span.End()

	return db.DB.QueryRowContext(ctx, query, args...)
}

func (db *TracedDB) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	ctx, span := db.startSpan(ctx, "db.query_row", query)
	defer span.End()

	return db.DB.QueryRowxContext(ctx, query, args...)
}

func (db *TracedDB) QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	ctx, span := db.startSpan(ctx, "db.query", query)
	defer span.End()

	rows, err := db.DB.QueryxContext(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return rows, err
}

func (db *TracedDB) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	ctx, span := db.startSpan(ctx, "db.get", query)
	defer span.End()

	err := db.DB.GetContext(ctx, dest, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

func (db *TracedDB) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	ctx, span := db.startSpan(ctx, "db.select", query)
	defer span.End()

	err := db.DB.SelectContext(ctx, dest, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

func (db *TracedDB) NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	ctx, span := db.startSpan(ctx, "db.named_exec", query)
	defer span.End()

	result, err := db.DB.NamedExecContext(ctx, query, arg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return result, err
}
