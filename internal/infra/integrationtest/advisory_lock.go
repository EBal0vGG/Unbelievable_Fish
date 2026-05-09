package integrationtest

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

// sharedPostgresTestLockKey serializes RealPG tests across packages that reuse one Docker Postgres.
const sharedPostgresTestLockKey int64 = 0x4669_7368_5F69_6E74 // "fish_int" as hex-ish distinct key

// AcquireSharedPostgresAdvisoryLock blocks until no other session holds this test lock.
//
// Call after registering t.Cleanup(db.Close) so the unlock cleanup runs before the pool closes
// (last registered cleanup runs first).
func AcquireSharedPostgresAdvisoryLock(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("integrationtest: db.Conn: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, sharedPostgresTestLockKey); err != nil {
		_ = conn.Close()
		t.Fatalf("integrationtest: pg_advisory_lock: %v", err)
	}
	t.Cleanup(func() {
		ctx2, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if _, err := conn.ExecContext(ctx2, `SELECT pg_advisory_unlock($1)`, sharedPostgresTestLockKey); err != nil {
			t.Errorf("integrationtest: pg_advisory_unlock: %v", err)
		}
		if err := conn.Close(); err != nil {
			t.Errorf("integrationtest: release conn: %v", err)
		}
	})
}
