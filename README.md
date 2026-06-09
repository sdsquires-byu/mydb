# Go Database Package Starter

This repo is a Go package for database integration. It wraps `sqlx` with a
transaction manager and a small SQL query builder.

## Shape

```text
mydb/
  go.mod          module metadata and the sqlx dependency
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

`Statement` keeps SQL text and bind arguments together:

```go
stmt := mydb.Stmt("insert into users(name) values (?)", "Alice")
```

`Exec`, `Get`, and `Select` run a `Statement` through a `Queryer`. `Get` and
`Select` use `sqlx.GetContext` and `sqlx.SelectContext`, so callers still get
sqlx struct scanning.

`Join`, `Ident`, and `Value` build query text and args without executing
anything.

`TxManager` owns transaction lifecycle. It starts a `*sqlx.Tx` with
`db.BeginTxx`, gives the transaction to your callback as a `Queryer`, rolls back
on callback error, and commits on success.

## Using It

The importing application owns connection setup and shutdown. This package only
uses the `*sqlx.DB` it is given:

```go
db, err := sqlx.Connect("postgres", dsn)
if err != nil {
	return err
}
defer db.Close()

txManager, err := mydb.NewTxManager(db)
if err != nil {
	return err
}

stmt, err := mydb.Join(
	"INSERT INTO",
	mydb.Ident("users"),
	"(name) VALUES",
	mydb.Value("Alice"),
)
if err != nil {
	return err
}

err = txManager.Do(ctx, func(q mydb.Queryer) error {
	_, err := mydb.Exec(ctx, q, stmt)
	return err
})
```

## Next Steps

1. Build a small caller-owned `UserStore` that accepts `mydb.Queryer`.
2. Put user-specific SQL in that store, not in `TxManager`.
3. Use `txManager.Do` only when several store calls must succeed or fail
   together.
4. Keep expanding the query builder only when a real store method needs it.
