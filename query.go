package mydb

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// Queryer is the small SQL execution contract this package builds around.
//
// Both *sqlx.DB and *sqlx.Tx satisfy this interface.
type Queryer interface {
	sqlx.QueryerContext
	sqlx.ExecerContext
}

// Statement keeps SQL text and bind arguments together.
type Statement struct {
	Text string
	Args []any
}

// Stmt creates a database statement value.
func Stmt(text string, args ...any) Statement {
	return Statement{
		Text: text,
		Args: append([]any(nil), args...),
	}
}

// Exec runs a statement that changes data or schema.
func Exec(ctx context.Context, q Queryer, stmt Statement) (sql.Result, error) {
	if q == nil {
		return nil, ErrNilQueryer
	}

	return q.ExecContext(ctx, stmt.Text, stmt.Args...)
}

// Get runs a statement expected to scan exactly one row into dest.
func Get(ctx context.Context, q Queryer, dest any, stmt Statement) error {
	if q == nil {
		return ErrNilQueryer
	}

	return sqlx.GetContext(ctx, q, dest, stmt.Text, stmt.Args...)
}

// Select runs a statement expected to scan many rows into dest.
func Select(ctx context.Context, q Queryer, dest any, stmt Statement) error {
	if q == nil {
		return ErrNilQueryer
	}

	return sqlx.SelectContext(ctx, q, dest, stmt.Text, stmt.Args...)
}
