package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/jmoiron/sqlx"
	"github.com/sdsquires-byu/mydb"
	_ "modernc.org/sqlite"
)

var errRollbackCheck = errors.New("rollback check")

type User struct {
	ID    int64  `db:"id"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

type UserStore struct {
	q mydb.Queryer
}

func NewUserStore(q mydb.Queryer) UserStore {
	return UserStore{q: q}
}

func (s UserStore) GetByID(ctx context.Context, id int64) (*User, error) {
	stmt, err := mydb.Join(
		"SELECT id, name, email FROM",
		mydb.Ident("users"),
		"WHERE",
		mydb.Ident("id"),
		"=",
		mydb.Value(id),
	)
	if err != nil {
		return nil, err
	}

	var user User
	if err := mydb.Get(ctx, s.q, &user, stmt); err != nil {
		return nil, err
	}

	return &user, nil
}

func (s UserStore) Create(ctx context.Context, user User) error {
	stmt, err := mydb.Join(
		"INSERT INTO",
		mydb.Ident("users"),
		"(id, name, email) VALUES (",
		mydb.Value(user.ID),
		",",
		mydb.Value(user.Name),
		",",
		mydb.Value(user.Email),
		")",
	)
	if err != nil {
		return err
	}

	_, err = mydb.Exec(ctx, s.q, stmt)
	return err
}

func (s UserStore) Exists(ctx context.Context, id int64) (bool, error) {
	_, err := s.GetByID(ctx, id)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	return false, err
}

func run(ctx context.Context, db *mydb.DB) error {
	stmt, err :=
		mydb.Join(`
		CREATE TABLE `,
			mydb.Ident("users"),
			`(
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT
		)`)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, stmt)
	if err != nil {

		return err
	}

	if err := db.Do(ctx, func(q mydb.Queryer) error {
		txUsers := NewUserStore(q)

		return txUsers.Create(ctx, User{
			ID:    1246,
			Name:  "Seth Squires",
			Email: "seth_squires@example.com",
		})
	}); err != nil {
		return err
	}

	return verifyRollback(ctx, db)
}

func verifyRollback(ctx context.Context, db *mydb.DB) error {
	err := db.Do(ctx, func(q mydb.Queryer) error {
		txUsers := NewUserStore(q)
		if err := txUsers.Create(ctx, User{
			ID:    1247,
			Name:  "Rollback Test",
			Email: "rollback@example.com",
		}); err != nil {
			return err
		}

		return errRollbackCheck
	})
	if !errors.Is(err, errRollbackCheck) {
		return err
	}

	users := NewUserStore(db)
	exists, err := users.Exists(ctx, 1247)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("rollback verification failed: user 1247 still exists")
	}

	return nil
}

func main() {
	sqlxDB, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		slog.Error("oh no", "oh no", err)
		return
	}
	sqlxDB.SetMaxOpenConns(1)

	db, err := mydb.NewDB(sqlxDB, mydb.WithQueryLogging(slog.Default()))
	if err != nil {
		slog.Error("database wrapping failed", "error:", err)
		return
	}
	err = run(context.Background(), db)
	if err != nil {
		slog.Error("running failed", "error:", err)
		return
	}

	users := NewUserStore(db)
	user, err := users.GetByID(context.Background(), 1246)
	if err != nil {
		slog.Error("database could not find user", "whoops", err)
		return
	}
	attr := slog.Any("user", user)
	slog.Info("User found!", attr)
}
