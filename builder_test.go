package mydb

import (
	"errors"
	"testing"
)

func TestJoinBuildsStatementWithIdentifierAndValue(t *testing.T) {
	stmt, err := Join(
		"SELECT * FROM",
		Ident("users"),
		"WHERE",
		Ident("users.name"),
		"=",
		Value("Alice"),
	)
	if err != nil {
		t.Fatalf("join: %v", err)
	}

	if stmt.Text != "SELECT * FROM users WHERE users.name = ?" {
		t.Fatalf("unexpected SQL: %q", stmt.Text)
	}
	if len(stmt.Args) != 1 || stmt.Args[0] != "Alice" {
		t.Fatalf("unexpected args: %#v", stmt.Args)
	}
}

func TestJoinBuildStatementWithIdentifierAndStatement(t *testing.T) {
	stmt, err := Join(
		"INSERT INTO",
		Ident("users"),
		"VALUES",
		Statement{
			Text: "('?', ?, '?')",
			Args: []any{"Seth", 1234, "seth@example.com"},
		},
	)
	if err != nil {
		t.Fatalf("join: %v", err)
	}

	if stmt.Text != "INSERT INTO users VALUES ('?', ?, '?')" {
		t.Fatalf("unexpected SQL: %q", stmt.Text)
	}
	if len(stmt.Args) != 3 || stmt.Args[0] != "Seth" || stmt.Args[1] != 1234 || stmt.Args[2] != "seth@example.com" {
		t.Fatalf("unexpected args: %#v", stmt.Args)
	}
}

func TestJoinRejectsUnsafeIdentifier(t *testing.T) {
	_, err := Join("SELECT * FROM", Ident("users; DROP TABLE users"))
	if !errors.Is(err, ErrInvalidIdentifier) {
		t.Fatalf("expected ErrInvalidIdentifier, got %v", err)
	}
}

func TestJoinRejectsUnsupportedPart(t *testing.T) {
	_, err := Join("SELECT", 42)
	if !errors.Is(err, ErrUnsupportedSQLPart) {
		t.Fatalf("expected ErrUnsupportedSQLPart, got %v", err)
	}
}
