package mydb

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

// Tx is the SQL execution contract plus transaction lifecycle methods.
//
// A *sqlx.Tx satisfies this interface if it also satisfies Queryer.
type Tx interface {
	Queryer
	Commit() error
	Rollback() error
}

type txStarter interface {
	beginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error)
}

// TxManager owns BEGIN / COMMIT / ROLLBACK behavior for a unit of work.
type TxManager struct {
	starter txStarter
}

// NewTxManager creates a transaction manager around a sqlx database.
func NewTxManager(db *sqlx.DB) (*TxManager, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &TxManager{starter: sqlxStarter{db: db}}, nil
}

// Do runs fn inside a transaction using default transaction options.
func (m *TxManager) Do(ctx context.Context, fn func(Queryer) error) error {
	return m.DoTx(ctx, nil, fn)
}

// DoTx runs fn inside a transaction.
//
// If fn returns nil, the transaction commits. If fn returns an error, the
// transaction rolls back and that error is returned.
func (m *TxManager) DoTx(ctx context.Context, opts *sql.TxOptions, fn func(Queryer) error) (err error) {
	if fn == nil {
		return ErrNilTxFunc
	}

	tx, err := m.starter.beginTx(ctx, opts)
	if err != nil {
		return err
	}
	if tx == nil {
		return ErrNilTx
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return errors.Join(err, rollbackErr)
		}
		return err
	}

	return tx.Commit()
}

type sqlxStarter struct {
	db *sqlx.DB
}

func (s sqlxStarter) beginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	return s.db.BeginTxx(ctx, opts)
}

type txStarterFunc func(ctx context.Context, opts *sql.TxOptions) (Tx, error)

func (f txStarterFunc) beginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	if f == nil {
		return nil, ErrNilBeginner
	}

	return f(ctx, opts)
}

func newTxManagerForTest(starter txStarter) (*TxManager, error) {
	if starter == nil {
		return nil, ErrNilBeginner
	}

	return &TxManager{starter: starter}, nil
}
