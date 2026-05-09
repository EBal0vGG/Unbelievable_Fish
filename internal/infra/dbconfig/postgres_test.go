package dbconfig

import (
	"context"
	"testing"
	"time"
)

func TestOpenPostgresFromEnvMissingRequired(t *testing.T) {
	t.Setenv("PGHOST", "")
	t.Setenv("PGUSER", "u")
	t.Setenv("PGDATABASE", "d")
	db, ok := OpenPostgresFromEnv(0)
	if ok || db != nil {
		t.Fatalf("expected no db when PGHOST empty, got ok=%v db=%v", ok, db)
	}
}

func TestOpenPostgresDockerComposeDefaultsPing(t *testing.T) {
	db, err := OpenPostgresDockerComposeDefaults(1)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Skipf("postgres not reachable on docker-compose defaults (%v); try: docker compose up -d postgres", err)
	}
}
