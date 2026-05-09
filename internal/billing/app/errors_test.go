package app

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsPostgresUniqueViolation(t *testing.T) {
	if IsPostgresUniqueViolation(nil) {
		t.Fatal("nil")
	}
	if IsPostgresUniqueViolation(errors.New("other")) {
		t.Fatal("generic")
	}
	pg := &pgconn.PgError{Code: "23505"}
	if !IsPostgresUniqueViolation(pg) {
		t.Fatal("want true for 23505")
	}
	if IsPostgresUniqueViolation(&pgconn.PgError{Code: "23503"}) {
		t.Fatal("foreign key")
	}
	if !IsPostgresUniqueViolation(fmt.Errorf("exec: %w", pg)) {
		t.Fatal("wrapped")
	}
}
