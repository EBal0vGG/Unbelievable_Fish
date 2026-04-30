package dbconfig

import "testing"

func TestOpenPostgresFromEnvMissingRequired(t *testing.T) {
	t.Setenv("PGHOST", "")
	t.Setenv("PGUSER", "u")
	t.Setenv("PGDATABASE", "d")
	db, ok := OpenPostgresFromEnv(0)
	if ok || db != nil {
		t.Fatalf("expected no db when PGHOST empty, got ok=%v db=%v", ok, db)
	}
}
