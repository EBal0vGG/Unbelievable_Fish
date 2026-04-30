package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/dbconfig"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/logging"
)

func main() {
	logger := logging.New("migrate")
	db, ok := dbconfig.OpenPostgresFromEnv(2)
	if !ok {
		logging.Fatal(logger, "database_config_missing", "required", "PGHOST,PGUSER,PGDATABASE")
	}
	defer db.Close()

	if err := applyMigrations(db); err != nil {
		logging.Fatal(logger, "migrations_apply_failed", "error", err)
	}
	logger.Info("migrations_applied")
}

func applyMigrations(db *sql.DB) error {
	if err := ensureSchemaMigrationsTable(db); err != nil {
		return err
	}
	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		root := repoRoot()
		migrationsDir = filepath.Join(root, "migrations")
	}
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		files = append(files, filepath.Join(migrationsDir, entry.Name()))
	}
	sort.Strings(files)

	applied, err := loadAppliedMigrations(db)
	if err != nil {
		return err
	}

	for _, file := range files {
		body, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if strings.TrimSpace(string(body)) == "" {
			continue
		}
		filename := filepath.Base(file)
		checksum := migrationChecksum(body)
		if appliedChecksum, ok := applied[filename]; ok {
			if appliedChecksum != checksum {
				return fmt.Errorf("migration %s was already applied with different checksum", filename)
			}
			continue
		}
		if err := applyMigration(db, filename, checksum, string(body)); err != nil {
			return err
		}
	}
	return nil
}

func ensureSchemaMigrationsTable(db *sql.DB) error {
	const query = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    filename TEXT PRIMARY KEY,
    checksum TEXT NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)
`
	_, err := db.Exec(query)
	return err
}

func loadAppliedMigrations(db *sql.DB) (map[string]string, error) {
	rows, err := db.Query(`SELECT filename, checksum FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]string)
	for rows.Next() {
		var filename string
		var checksum string
		if err := rows.Scan(&filename, &checksum); err != nil {
			return nil, err
		}
		applied[filename] = checksum
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return applied, nil
}

func applyMigration(db *sql.DB, filename, checksum, body string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(body); err != nil {
		return fmt.Errorf("apply migration %s: %w", filename, err)
	}
	if _, err = tx.Exec(
		`INSERT INTO schema_migrations (filename, checksum) VALUES ($1, $2)`,
		filename,
		checksum,
	); err != nil {
		return fmt.Errorf("record migration %s: %w", filename, err)
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func migrationChecksum(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func repoRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	dir := filepath.Dir(filename)
	return filepath.Clean(filepath.Join(dir, "..", ".."))
}
