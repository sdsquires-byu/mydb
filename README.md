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
sqlx struct scanning.

`Join`, `Ident`, and `Value` build query text and args without executing
anything.

`TxManager` owns transaction lifecycle. You can use it directly through
`database.TxManager()` or use the wrapper shortcut `database.Do(...)`.

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

## Next Steps

1. Build a small caller-owned `UserStore` that accepts `mydb.Queryer`.
2. Put user-specific SQL in that store, not in `TxManager`.
3. Use `txManager.Do` only when several store calls must succeed or fail
   together.
4. Keep expanding the query builder only when a real store method needs it.
