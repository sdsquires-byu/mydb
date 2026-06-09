package mydb

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestNewTxManagerRejectsNilBeginner(t *testing.T) {
	manager, err := newTxManagerForTest(nil)
	if err != ErrNilBeginner {
		t.Fatalf("expected ErrNilBeginner, got %v", err)
	}
	if manager != nil {
		t.Fatalf("expected nil manager, got %#v", manager)
	}
}

func TestNewTxManagerRejectsNilDB(t *testing.T) {
	manager, err := NewTxManager(nil)
	if err != ErrNilDB {
		t.Fatalf("expected ErrNilDB, got %v", err)
	}
	if manager != nil {
		t.Fatalf("expected nil manager, got %#v", manager)
	}
}

func TestDoRejectsNilCallback(t *testing.T) {
	manager := newTestTxManager(t, &fakeTx{})

	err := manager.Do(context.Background(), nil)
	if err != ErrNilTxFunc {
		t.Fatalf("expected ErrNilTxFunc, got %v", err)
	}
}

func TestDoRejectsNilBeginFunc(t *testing.T) {
	manager, err := newTxManagerForTest(txStarterFunc(nil))
	if err != nil {
		t.Fatalf("new tx manager: %v", err)
	}

	err = manager.Do(context.Background(), func(Queryer) error {
		t.Fatal("callback should not run")
		return nil
	})
	if err != ErrNilBeginner {
		t.Fatalf("expected ErrNilBeginner, got %v", err)
	}
}

func TestDoReturnsBeginError(t *testing.T) {
	beginErr := errors.New("begin failed")
	manager, err := newTxManagerForTest(txStarterFunc(func(context.Context, *sql.TxOptions) (Tx, error) {
		return nil, beginErr
	}))
	if err != nil {
		t.Fatalf("new tx manager: %v", err)
	}

	err = manager.Do(context.Background(), func(Queryer) error {
		t.Fatal("callback should not run")
		return nil
	})
	if !errors.Is(err, beginErr) {
		t.Fatalf("expected begin error, got %v", err)
	}
}

func TestDoRejectsNilTransaction(t *testing.T) {
	manager, err := newTxManagerForTest(txStarterFunc(func(context.Context, *sql.TxOptions) (Tx, error) {
		return nil, nil
	}))
	if err != nil {
		t.Fatalf("new tx manager: %v", err)
	}

	err = manager.Do(context.Background(), func(Queryer) error {
		t.Fatal("callback should not run")
		return nil
	})
	if err != ErrNilTx {
		t.Fatalf("expected ErrNilTx, got %v", err)
	}
}

func TestDoCommitsOnSuccess(t *testing.T) {
	tx := &fakeTx{}
	manager := newTestTxManager(t, tx)

	err := manager.Do(context.Background(), func(q Queryer) error {
		if q != tx {
			t.Fatalf("expected callback queryer to be tx")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	if !tx.committed {
		t.Fatal("expected transaction to commit")
	}
	if tx.rolledBack {
		t.Fatal("did not expect transaction to rollback")
	}
}

func TestDoRollsBackOnCallbackError(t *testing.T) {
	callbackErr := errors.New("callback failed")
	tx := &fakeTx{}
	manager := newTestTxManager(t, tx)

	err := manager.Do(context.Background(), func(Queryer) error {
		return callbackErr
	})
	if !errors.Is(err, callbackErr) {
		t.Fatalf("expected callback error, got %v", err)
	}
	if !tx.rolledBack {
		t.Fatal("expected transaction to rollback")
	}
	if tx.committed {
		t.Fatal("did not expect transaction to commit")
	}
}

func TestDoReturnsCommitError(t *testing.T) {
	commitErr := errors.New("commit failed")
	tx := &fakeTx{commitErr: commitErr}
	manager := newTestTxManager(t, tx)

	err := manager.Do(context.Background(), func(Queryer) error {
		return nil
	})
	if !errors.Is(err, commitErr) {
		t.Fatalf("expected commit error, got %v", err)
	}
}

func TestDoJoinsCallbackAndRollbackErrors(t *testing.T) {
	callbackErr := errors.New("callback failed")
	rollbackErr := errors.New("rollback failed")
	tx := &fakeTx{rollbackErr: rollbackErr}
	manager := newTestTxManager(t, tx)

	err := manager.Do(context.Background(), func(Queryer) error {
		return callbackErr
	})
	if !errors.Is(err, callbackErr) {
		t.Fatalf("expected callback error in joined error, got %v", err)
	}
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("expected rollback error in joined error, got %v", err)
	}
}

func newTestTxManager(t *testing.T, tx Tx) *TxManager {
	t.Helper()

	manager, err := newTxManagerForTest(txStarterFunc(func(context.Context, *sql.TxOptions) (Tx, error) {
		return tx, nil
	}))
	if err != nil {
		t.Fatalf("new tx manager: %v", err)
	}

	return manager
}

type fakeTx struct {
	recordingQueryer
	committed   bool
	rolledBack  bool
	commitErr   error
	rollbackErr error
}

func (tx *fakeTx) Commit() error {
	tx.committed = true
	return tx.commitErr
}

func (tx *fakeTx) Rollback() error {
	tx.rolledBack = true
	return tx.rollbackErr
}
