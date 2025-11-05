package apm

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

type DBMetricsWrapper struct {
	db        *sqlx.DB
	collector *MetricsCollector
}

func (w *DBMetricsWrapper) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := w.db.QueryContext(ctx, query, args...)
	duration := time.Since(start)

	w.collector.RecordDatabaseQuery(ctx, query, duration, err == nil)
	return rows, err
}

func (w *DBMetricsWrapper) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	row := w.db.QueryRowContext(ctx, query, args...)
	duration := time.Since(start)

	w.collector.RecordDatabaseQuery(ctx, query, duration, true)
	return row
}

func (w *DBMetricsWrapper) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := w.db.ExecContext(ctx, query, args...)
	duration := time.Since(start)

	w.collector.RecordDatabaseQuery(ctx, query, duration, err == nil)
	return result, err
}

func (w *DBMetricsWrapper) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	start := time.Now()
	err := w.db.SelectContext(ctx, dest, query, args...)
	duration := time.Since(start)

	w.collector.RecordDatabaseQuery(ctx, query, duration, err == nil)
	return err
}

func (w *DBMetricsWrapper) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	start := time.Now()
	err := w.db.GetContext(ctx, dest, query, args...)
	duration := time.Since(start)

	w.collector.RecordDatabaseQuery(ctx, query, duration, err == nil)
	return err
}

func (w *DBMetricsWrapper) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return w.db.BeginTx(ctx, opts)
}

func (w *DBMetricsWrapper) Stats() sql.DBStats {
	return w.db.Stats()
}

func (w *DBMetricsWrapper) Close() error {
	return w.db.Close()
}

func (w *DBMetricsWrapper) GetDB() *sqlx.DB {
	return w.db
}
