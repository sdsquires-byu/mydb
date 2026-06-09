package mydb

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
)

var _ Queryer = (*DB)(nil)

// DB wraps an existing sqlx database with this package's helpers.
//
// The application still owns opening, configuring, and closing the sqlx.DB.
type DB struct {
	db        *sqlx.DB
	txManager *TxManager
	logger    *slog.Logger
}

// DBOption configures a DB wrapper.
type DBOption func(*DB)

// WithQueryLogging enables query logging for the DB wrapper.
//
// If logger is nil, slog.Default() is used.
func WithQueryLogging(logger *slog.Logger) DBOption {
	return func(db *DB) {
		if logger == nil {
			logger = slog.Default()
		}
		db.SetQueryLogger(logger)
	}
}

// NewDB wraps an existing sqlx database.
func NewDB(db *sqlx.DB, opts ...DBOption) (*DB, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	wrapped := &DB{db: db}
	for _, opt := range opts {
		opt(wrapped)
	}

	txManager, err := NewTxManager(db, WithTxQueryLogging(wrapped.logger))
	if err != nil {
		return nil, err
	}

	wrapped.txManager = txManager
	return wrapped, nil
}

// SQLX returns the wrapped sqlx database for application code that needs it.
func (db *DB) SQLX() *sqlx.DB {
	return db.db
}

// TxManager returns the transaction manager used by this wrapper.
func (db *DB) TxManager() *TxManager {
	return db.txManager
}

// SetQueryLogger toggles query logging. A nil logger disables query logging.
func (db *DB) SetQueryLogger(logger *slog.Logger) {
	db.logger = logger
	if db.txManager != nil {
		db.txManager.SetQueryLogger(logger)
	}
}

// QueryLogger returns the logger used for query logging, or nil when disabled.
func (db *DB) QueryLogger() *slog.Logger {
	return db.logger
}

// Exec runs a statement against the wrapped database.
func (db *DB) Exec(ctx context.Context, stmt Statement) (sql.Result, error) {
	return Exec(ctx, db, stmt)
}

// Get runs a statement expected to scan exactly one row into dest.
func (db *DB) Get(ctx context.Context, dest any, stmt Statement) error {
	return Get(ctx, db, dest, stmt)
}

// Select runs a statement expected to scan many rows into dest.
func (db *DB) Select(ctx context.Context, dest any, stmt Statement) error {
	return Select(ctx, db, dest, stmt)
}

// Rebind changes ? placeholders to the wrapped database driver's bind style.
func (db *DB) Rebind(query string) string {
	return db.db.Rebind(query)
}

// Do runs fn inside a transaction using default transaction options.
func (db *DB) Do(ctx context.Context, fn func(Queryer) error) error {
	return db.txManager.Do(ctx, fn)
}

// DoTx runs fn inside a transaction using explicit transaction options.
func (db *DB) DoTx(ctx context.Context, opts *sql.TxOptions, fn func(Queryer) error) error {
	return db.txManager.DoTx(ctx, opts, fn)
}

// ExecContext lets DB satisfy Queryer.
func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (result sql.Result, err error) {
	stmt := Stmt(query, args...)
	if db.logger == nil || queryLogSuppressed(ctx) {
		return db.db.ExecContext(ctx, stmt.Text, stmt.Args...)
	}

	err = logQuery(ctx, db.logger, "exec", stmt, func(ctx context.Context) error {
		result, err = db.db.ExecContext(ctx, stmt.Text, stmt.Args...)
		return err
	})
	return result, err
}

// QueryContext lets DB satisfy Queryer.
func (db *DB) QueryContext(ctx context.Context, query string, args ...any) (rows *sql.Rows, err error) {
	stmt := Stmt(query, args...)
	if db.logger == nil || queryLogSuppressed(ctx) {
		return db.db.QueryContext(ctx, stmt.Text, stmt.Args...)
	}

	err = logQuery(ctx, db.logger, "query", stmt, func(ctx context.Context) error {
		rows, err = db.db.QueryContext(ctx, stmt.Text, stmt.Args...)
		return err
	})
	return rows, err
}

// QueryxContext lets DB satisfy Queryer.
func (db *DB) QueryxContext(ctx context.Context, query string, args ...any) (rows *sqlx.Rows, err error) {
	stmt := Stmt(query, args...)
	if db.logger == nil || queryLogSuppressed(ctx) {
		return db.db.QueryxContext(ctx, stmt.Text, stmt.Args...)
	}

	err = logQuery(ctx, db.logger, "queryx", stmt, func(ctx context.Context) error {
		rows, err = db.db.QueryxContext(ctx, stmt.Text, stmt.Args...)
		return err
	})
	return rows, err
}

// QueryRowxContext lets DB satisfy Queryer.
func (db *DB) QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row {
	stmt := Stmt(query, args...)
	if db.logger == nil || queryLogSuppressed(ctx) {
		return db.db.QueryRowxContext(ctx, stmt.Text, stmt.Args...)
	}

	start := time.Now()
	row := db.db.QueryRowxContext(ctx, stmt.Text, stmt.Args...)
	logDatabaseQuery(ctx, db.logger, "query_rowx", stmt, time.Since(start), nil)
	return row
}

func (db *DB) queryLogger() *slog.Logger {
	return db.logger
}
