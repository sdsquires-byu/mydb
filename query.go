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

	stmt = rebind(q, stmt)
	logger := queryLogger(q)
	if logger == nil {
		return q.ExecContext(ctx, stmt.Text, stmt.Args...)
	}

	var result sql.Result
	var err error
	err = logQuery(ctx, logger, "exec", stmt, func(ctx context.Context) error {
		result, err = q.ExecContext(ctx, stmt.Text, stmt.Args...)
		return err
	})
	return result, err
}

// Get runs a statement expected to scan exactly one row into dest.
func Get(ctx context.Context, q Queryer, dest any, stmt Statement) error {
	if q == nil {
		return ErrNilQueryer
	}

	stmt = rebind(q, stmt)
	logger := queryLogger(q)
	if logger == nil {
		return sqlx.GetContext(ctx, q, dest, stmt.Text, stmt.Args...)
	}

	return logQuery(ctx, logger, "get", stmt, func(ctx context.Context) error {
		return sqlx.GetContext(ctx, q, dest, stmt.Text, stmt.Args...)
	})
}

// Select runs a statement expected to scan many rows into dest.
func Select(ctx context.Context, q Queryer, dest any, stmt Statement) error {
	if q == nil {
		return ErrNilQueryer
	}

	stmt = rebind(q, stmt)
	logger := queryLogger(q)
	if logger == nil {
		return sqlx.SelectContext(ctx, q, dest, stmt.Text, stmt.Args...)
	}

	return logQuery(ctx, logger, "select", stmt, func(ctx context.Context) error {
		return sqlx.SelectContext(ctx, q, dest, stmt.Text, stmt.Args...)
	})
}

type rebindable interface {
	Rebind(query string) string
}

func rebind(q Queryer, stmt Statement) Statement {
	r, ok := q.(rebindable)
	if !ok {
		return stmt
	}

	return Stmt(r.Rebind(stmt.Text), stmt.Args...)
}
