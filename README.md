# Go Database Package Starter

This repo is a Go package for database integration. It wraps `sqlx` with a
transaction manager and a small SQL query builder.

## Shape

```text
mydb/
  go.mod          module metadata and the sqlx dependency
  db.go           database wrapper around an existing *sqlx.DB
  query.go        sqlx-backed query contract and statement helpers
  tx.go           sqlx transaction manager
  builder.go      query builder: Join, Ident, Value
  errors.go       shared package errors
  *_test.go       starter tests for the package contract
```

## How The Pieces Fit

`Queryer` is the contract. It describes the small part of `sqlx` this package
needs. Both `*sqlx.DB` and `*sqlx.Tx` match it, which lets caller-owned stores
work outside or inside a transaction.

`DB` is the concrete wrapper. It accepts an existing `*sqlx.DB`, exposes
statement helpers, implements `Queryer`, and owns a `TxManager`.

`Statement` keeps SQL text and bind arguments together:

```go
stmt := mydb.Stmt("insert into users(name) values (?)", "Alice")
```

`Exec`, `Get`, and `Select` run a `Statement` through a `Queryer`. `Get` and
`Select` use `sqlx.GetContext` and `sqlx.SelectContext`, so callers still get
sqlx struct scanning. If the query runner supports `Rebind`, statements are
automatically rebound to the driver's placeholder style before execution.

`Join`, `Ident`, and `Value` build query text and args without executing
anything.

`TxManager` owns transaction lifecycle. You can use it directly through
`database.TxManager()` or use the wrapper shortcut `database.Do(...)`.

Query logging is opt-in. When enabled, the package logs the operation, rebound
query text, args, duration, and error if one occurs:

```go
database, err := mydb.NewDB(sqlxDB, mydb.WithQueryLogging(slog.Default()))
if err != nil {
	return err
}

database.SetQueryLogger(nil) // disable query logging
```

## Using It

The importing application owns connection setup and shutdown. This package only
uses the `*sqlx.DB` it is given:

```go
type UserStore struct {
	q mydb.Queryer
}

func NewUserStore(q mydb.Queryer) UserStore {
	return UserStore{q: q}
}

database, err := mydb.NewDB(sqlxDB)
if err != nil {
	return err
}

users := NewUserStore(database)
_, err = users.GetByID(ctx, 1245)
if err != nil {
	return err
}

err = database.Do(ctx, func(q mydb.Queryer) error {
	txUsers := NewUserStore(q)
	return txUsers.Create(ctx, user)
})
```
