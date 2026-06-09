package mydb

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
)

type queryLogSuppressedKey struct{}

func suppressQueryLog(ctx context.Context) context.Context {
	return context.WithValue(ctx, queryLogSuppressedKey{}, true)
}

func queryLogSuppressed(ctx context.Context) bool {
	suppressed, _ := ctx.Value(queryLogSuppressedKey{}).(bool)
	return suppressed
}

func queryLogger(q Queryer) *slog.Logger {
	provider, ok := q.(interface {
		queryLogger() *slog.Logger
	})
	if !ok {
		return nil
	}

	return provider.queryLogger()
}

func logDatabaseQuery(ctx context.Context, logger *slog.Logger, operation string, stmt Statement, duration time.Duration, err error) {
	if logger == nil {
		return
	}

	attrs := []slog.Attr{
		slog.String("operation", operation),
		slog.String("query", stmt.Text),
		slog.Any("args", stmt.Args),
		slog.Duration("duration", duration),
	}

	level := slog.LevelInfo
	if err != nil {
		level = slog.LevelError
		attrs = append(attrs, slog.Any("error", err))
	}

	logger.LogAttrs(ctx, level, "database query", attrs...)
}

func logQuery(ctx context.Context, logger *slog.Logger, operation string, stmt Statement, run func(context.Context) error) error {
	start := time.Now()
	err := run(suppressQueryLog(ctx))
	logDatabaseQuery(ctx, logger, operation, stmt, time.Since(start), err)
	return err
}

type loggedQueryer struct {
	q      Queryer
	logger *slog.Logger
}

func (q loggedQueryer) ExecContext(ctx context.Context, query string, args ...any) (result sql.Result, err error) {
	stmt := Stmt(query, args...)
	if q.logger == nil || queryLogSuppressed(ctx) {
		return q.q.ExecContext(ctx, stmt.Text, stmt.Args...)
	}

	err = logQuery(ctx, q.logger, "exec", stmt, func(ctx context.Context) error {
		result, err = q.q.ExecContext(ctx, stmt.Text, stmt.Args...)
		return err
	})
	return result, err
}

func (q loggedQueryer) QueryContext(ctx context.Context, query string, args ...any) (rows *sql.Rows, err error) {
	stmt := Stmt(query, args...)
	if q.logger == nil || queryLogSuppressed(ctx) {
		return q.q.QueryContext(ctx, stmt.Text, stmt.Args...)
	}

	err = logQuery(ctx, q.logger, "query", stmt, func(ctx context.Context) error {
		rows, err = q.q.QueryContext(ctx, stmt.Text, stmt.Args...)
		return err
	})
	return rows, err
}

func (q loggedQueryer) QueryxContext(ctx context.Context, query string, args ...any) (rows *sqlx.Rows, err error) {
	stmt := Stmt(query, args...)
	if q.logger == nil || queryLogSuppressed(ctx) {
		return q.q.QueryxContext(ctx, stmt.Text, stmt.Args...)
	}

	err = logQuery(ctx, q.logger, "queryx", stmt, func(ctx context.Context) error {
		rows, err = q.q.QueryxContext(ctx, stmt.Text, stmt.Args...)
		return err
	})
	return rows, err
}

func (q loggedQueryer) QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row {
	stmt := Stmt(query, args...)
	if q.logger == nil || queryLogSuppressed(ctx) {
		return q.q.QueryRowxContext(ctx, stmt.Text, stmt.Args...)
	}

	start := time.Now()
	row := q.q.QueryRowxContext(ctx, stmt.Text, stmt.Args...)
	logDatabaseQuery(ctx, q.logger, "query_rowx", stmt, time.Since(start), nil)
	return row
}

func (q loggedQueryer) Rebind(query string) string {
	r, ok := q.q.(rebindable)
	if !ok {
		return query
	}

	return r.Rebind(query)
}

func (q loggedQueryer) queryLogger() *slog.Logger {
	return q.logger
}
