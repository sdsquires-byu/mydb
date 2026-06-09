package mydb

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestNewDBRejectsNilSQLXDB(t *testing.T) {
	db, err := NewDB(nil)
	if err != ErrNilDB {
		t.Fatalf("expected ErrNilDB, got %v", err)
	}
	if db != nil {
		t.Fatalf("expected nil DB, got %#v", db)
	}
}

func TestDBSatisfiesQueryer(t *testing.T) {
	var _ Queryer = (*DB)(nil)
}

func TestNewDBWrapsSQLXDB(t *testing.T) {
	db, _ := newWrappedTestDB(t, nil)

	if db.SQLX() == nil {
		t.Fatal("expected wrapped sqlx db")
	}
	if db.TxManager() == nil {
		t.Fatal("expected transaction manager")
	}
}

func TestDBQueryLoggingIsDisabledByDefault(t *testing.T) {
	var logs bytes.Buffer
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() {
		slog.SetDefault(oldLogger)
	})

	ctx := context.Background()
	db, _ := newWrappedTestDB(t, nil)

	_, err := db.Exec(ctx, Stmt("insert into widgets(id) values (?)", 10))
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if logs.Len() != 0 {
		t.Fatalf("expected no logs by default, got %s", logs.String())
	}
}

func TestDBQueryLoggingCanBeEnabledAndDisabled(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))

	ctx := context.Background()
	db, _ := newWrappedTestDB(t, nil, WithQueryLogging(logger))
	if db.QueryLogger() != logger {
		t.Fatal("expected db query logger to be set")
	}
	if db.TxManager().QueryLogger() != logger {
		t.Fatal("expected tx manager query logger to be set")
	}

	_, err := db.Exec(ctx, Stmt("insert into widgets(id) values (?)", 10))
	if err != nil {
		t.Fatalf("exec: %v", err)
	}

	output := logs.String()
	for _, want := range []string{
		`"msg":"database query"`,
		`"operation":"exec"`,
		`"query":"insert into widgets(id) values ($1)"`,
		`"args":[10]`,
		`"duration"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected log output to contain %s, got %s", want, output)
		}
	}

	logs.Reset()
	db.SetQueryLogger(nil)
	if db.QueryLogger() != nil {
		t.Fatal("expected db query logger to be disabled")
	}
	if db.TxManager().QueryLogger() != nil {
		t.Fatal("expected tx manager query logger to be disabled")
	}

	_, err = db.Exec(ctx, Stmt("insert into widgets(id) values (?)", 11))
	if err != nil {
		t.Fatalf("exec after disabling logs: %v", err)
	}
	if logs.Len() != 0 {
		t.Fatalf("expected no logs after disabling, got %s", logs.String())
	}
}

func TestDBQueryLoggingAppliesInsideTransactions(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))

	ctx := context.Background()
	db, _ := newWrappedTestDB(t, nil, WithQueryLogging(logger))

	err := db.Do(ctx, func(q Queryer) error {
		_, err := q.ExecContext(ctx, "insert into widgets(id) values (?)", 10)
		return err
	})
	if err != nil {
		t.Fatalf("do: %v", err)
	}

	output := logs.String()
	for _, want := range []string{
		`"msg":"database query"`,
		`"operation":"exec"`,
		`"query":"insert into widgets(id) values (?)"`,
		`"args":[10]`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected log output to contain %s, got %s", want, output)
		}
	}
}

func TestDBExecRebindsAndExecutesStatement(t *testing.T) {
	ctx := context.Background()
	db, state := newWrappedTestDB(t, nil)

	result, err := db.Exec(ctx, Stmt("insert into widgets(id) values (?)", 10))
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
	if state.execQuery != "insert into widgets(id) values ($1)" {
		t.Fatalf("unexpected exec query: %q", state.execQuery)
	}
	if len(state.execArgs) != 1 || state.execArgs[0] != int64(10) {
		t.Fatalf("unexpected exec args: %#v", state.execArgs)
	}
}

func TestDBGetRebindsAndScansOneRow(t *testing.T) {
	ctx := context.Background()
	db, state := newWrappedTestDB(t, [][]driver.Value{{
		int64(10),
		"Alice",
		"alice@example.com",
	}})

	var user testUser
	err := db.Get(ctx, &user, Stmt("select id, name, email from users where id = ?", 10))
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if user.ID != 10 || user.Name != "Alice" || user.Email != "alice@example.com" {
		t.Fatalf("unexpected user: %#v", user)
	}
	if state.queryQuery != "select id, name, email from users where id = $1" {
		t.Fatalf("unexpected query: %q", state.queryQuery)
	}
}

func TestDBSelectRebindsAndScansRows(t *testing.T) {
	ctx := context.Background()
	db, state := newWrappedTestDB(t, [][]driver.Value{
		{int64(10), "Alice", "alice@example.com"},
		{int64(11), "Bob", "bob@example.com"},
	})

	var users []testUser
	err := db.Select(ctx, &users, Stmt("select id, name, email from users where id > ?", 9))
	if err != nil {
		t.Fatalf("select: %v", err)
	}

	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %#v", users)
	}
	if state.queryQuery != "select id, name, email from users where id > $1" {
		t.Fatalf("unexpected query: %q", state.queryQuery)
	}
}

func TestDBQueryContextForwardsToWrappedSQLXDB(t *testing.T) {
	ctx := context.Background()
	db, state := newWrappedTestDB(t, [][]driver.Value{{int64(10), "Alice", "alice@example.com"}})

	rows, err := db.QueryContext(ctx, "select id, name, email from users")
	if err != nil {
		t.Fatalf("query context: %v", err)
	}
	defer rows.Close()

	if state.queryQuery != "select id, name, email from users" {
		t.Fatalf("unexpected query: %q", state.queryQuery)
	}
}

func TestDBDoCommitsTransaction(t *testing.T) {
	ctx := context.Background()
	db, state := newWrappedTestDB(t, nil)

	err := db.Do(ctx, func(q Queryer) error {
		if q == nil {
			t.Fatal("expected transaction-backed queryer")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("do: %v", err)
	}

	if state.commits != 1 {
		t.Fatalf("expected 1 commit, got %d", state.commits)
	}
	if state.rollbacks != 0 {
		t.Fatalf("expected 0 rollbacks, got %d", state.rollbacks)
	}
}

func TestDBDoTxRollsBackTransaction(t *testing.T) {
	ctx := context.Background()
	db, state := newWrappedTestDB(t, nil)
	callbackErr := errors.New("callback failed")

	err := db.DoTx(ctx, nil, func(Queryer) error {
		return callbackErr
	})
	if !errors.Is(err, callbackErr) {
		t.Fatalf("expected callback error, got %v", err)
	}

	if state.commits != 0 {
		t.Fatalf("expected 0 commits, got %d", state.commits)
	}
	if state.rollbacks != 1 {
		t.Fatalf("expected 1 rollback, got %d", state.rollbacks)
	}
}

func newWrappedTestDB(t *testing.T, rows [][]driver.Value, opts ...DBOption) (*DB, *testDBState) {
	t.Helper()

	state := &testDBState{rows: rows}
	raw := sql.OpenDB(testConnector{state: state})
	t.Cleanup(func() {
		if err := raw.Close(); err != nil {
			t.Fatalf("close raw db: %v", err)
		}
	})

	sqlxDB := sqlx.NewDb(raw, "postgres")
	db, err := NewDB(sqlxDB, opts...)
	if err != nil {
		t.Fatalf("new db: %v", err)
	}

	return db, state
}

type testUser struct {
	ID    int64  `db:"id"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

type testDBState struct {
	execQuery  string
	execArgs   []driver.Value
	queryQuery string
	queryArgs  []driver.Value
	rows       [][]driver.Value
	commits    int
	rollbacks  int
}

type testConnector struct {
	state *testDBState
}

func (c testConnector) Connect(context.Context) (driver.Conn, error) {
	return &testConn{state: c.state}, nil
}

func (c testConnector) Driver() driver.Driver {
	return testDriver{state: c.state}
}

type testDriver struct {
	state *testDBState
}

func (d testDriver) Open(string) (driver.Conn, error) {
	return &testConn{state: d.state}, nil
}

type testConn struct {
	state *testDBState
}

func (c *testConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not implemented")
}

func (c *testConn) Close() error {
	return nil
}

func (c *testConn) Begin() (driver.Tx, error) {
	return &testTx{state: c.state}, nil
}

func (c *testConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return &testTx{state: c.state}, nil
}

func (c *testConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.state.execQuery = query
	c.state.execArgs = namedValues(args)
	return driver.RowsAffected(1), nil
}

func (c *testConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.state.queryQuery = query
	c.state.queryArgs = namedValues(args)
	return &testRows{
		columns: []string{"id", "name", "email"},
		values:  c.state.rows,
	}, nil
}

type testTx struct {
	state *testDBState
}

func (tx *testTx) Commit() error {
	tx.state.commits++
	return nil
}

func (tx *testTx) Rollback() error {
	tx.state.rollbacks++
	return nil
}

type testRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (r *testRows) Columns() []string {
	return r.columns
}

func (r *testRows) Close() error {
	return nil
}

func (r *testRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}

	copy(dest, r.values[r.index])
	r.index++
	return nil
}

func namedValues(args []driver.NamedValue) []driver.Value {
	values := make([]driver.Value, 0, len(args))
	for _, arg := range args {
		values = append(values, arg.Value)
	}

	return values
}
