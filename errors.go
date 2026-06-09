package mydb

import "errors"

var (
	ErrInvalidIdentifier  = errors.New("mydb: invalid SQL identifier")
	ErrNilBeginner        = errors.New("mydb: transaction beginner is nil")
	ErrNilDB              = errors.New("mydb: sqlx database is nil")
	ErrNilQueryer         = errors.New("mydb: queryer is nil")
	ErrNilTx              = errors.New("mydb: transaction is nil")
	ErrNilTxFunc          = errors.New("mydb: transaction callback is nil")
	ErrUnsupportedSQLPart = errors.New("mydb: unsupported SQL part")
)
