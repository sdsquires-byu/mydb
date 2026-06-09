package mydb

import "testing"

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
