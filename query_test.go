package mydb

import (
	"context"
	"database/sql"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestStmtCopiesArgs(t *testing.T) {
	args := []any{"alpha"}
	stmt := Stmt("select ?", args...)

	args[0] = "bravo"

	if stmt.Args[0] != "alpha" {
		t.Fatalf("expected statement args to be copied, got %#v", stmt.Args)
	}
}

func TestExecUsesStatementTextAndArgs(t *testing.T) {
	queryer := &recordingQueryer{}
	result, err := Exec(context.Background(), queryer, Stmt("insert into widgets(name) values (?)", "alpha"))
	if err != nil {
		t.Fatalf("exec: %v", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("rows affected: %v", err)
	}
	if affected != 1 {
		t.Fatalf("expected 1 affected row, got %d", affected)
	}
	if queryer.text != "insert into widgets(name) values (?)" {
		t.Fatalf("unexpected query text: %q", queryer.text)
	}
	if len(queryer.args) != 1 || queryer.args[0] != "alpha" {
		t.Fatalf("unexpected args: %#v", queryer.args)
	}
}

func TestExecRejectsNilQueryer(t *testing.T) {
	_, err := Exec(context.Background(), nil, Stmt("select 1"))
	if err != ErrNilQueryer {
		t.Fatalf("expected ErrNilQueryer, got %v", err)
	}
}

type recordingQueryer struct {
	text string
	args []any
}

func (q *recordingQueryer) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, nil
}

func (q *recordingQueryer) QueryxContext(context.Context, string, ...any) (*sqlx.Rows, error) {
	return nil, nil
}

func (q *recordingQueryer) QueryRowxContext(context.Context, string, ...any) *sqlx.Row {
	return nil
}

func (q *recordingQueryer) ExecContext(_ context.Context, text string, args ...any) (sql.Result, error) {
	q.text = text
	q.args = args
	return fakeResult(1), nil
}

type fakeResult int64

func (r fakeResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (r fakeResult) RowsAffected() (int64, error) {
	return int64(r), nil
}
