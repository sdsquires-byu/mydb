package mydb

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

var _ Queryer = (*DB)(nil)

// DB wraps an existing sqlx database with this package's helpers.
//
// The application still owns opening, configuring, and closing the sqlx.DB.
type DB struct {
	db        *sqlx.DB
	txManager *TxManager
}

// NewDB wraps an existing sqlx database.
func NewDB(db *sqlx.DB) (*DB, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	txManager, err := NewTxManager(db)
	if err != nil {
		return nil, err
	}

	return &DB{
		db:        db,
		txManager: txManager,
	}, nil
}

// SQLX returns the wrapped sqlx database for application code that needs it.
func (db *DB) SQLX() *sqlx.DB {
	return db.db
}

// TxManager returns the transaction manager used by this wrapper.
func (db *DB) TxManager() *TxManager {
	return db.txManager
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

// Do runs fn inside a transaction using default transaction options.
func (db *DB) Do(ctx context.Context, fn func(Queryer) error) error {
	return db.txManager.Do(ctx, fn)
}

// DoTx runs fn inside a transaction using explicit transaction options.
func (db *DB) DoTx(ctx context.Context, opts *sql.TxOptions, fn func(Queryer) error) error {
	return db.txManager.DoTx(ctx, opts, fn)
}

// ExecContext lets DB satisfy Queryer.
func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.db.ExecContext(ctx, query, args...)
}

// QueryContext lets DB satisfy Queryer.
func (db *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.db.QueryContext(ctx, query, args...)
}

// QueryxContext lets DB satisfy Queryer.
func (db *DB) QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error) {
	return db.db.QueryxContext(ctx, query, args...)
}

// QueryRowxContext lets DB satisfy Queryer.
func (db *DB) QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row {
	return db.db.QueryRowxContext(ctx, query, args...)
}
