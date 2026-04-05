package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	db, ok := openDB()
	if !ok {
		log.Fatal("PGHOST/PGUSER/PGDATABASE are required")
	}
	defer db.Close()

	if err := applyMigrations(db); err != nil {
		log.Fatalf("apply migrations: %v", err)
	}
	log.Println("migrations applied")
}

func openDB() (*sql.DB, bool) {
	host := os.Getenv("PGHOST")
	user := os.Getenv("PGUSER")
	password := os.Getenv("PGPASSWORD")
	database := os.Getenv("PGDATABASE")
	port := os.Getenv("PGPORT")
	sslmode := os.Getenv("PGSSLMODE")

	if host == "" || user == "" || database == "" {
		return nil, false
	}
	if port == "" {
		port = "5432"
	}
	if sslmode == "" {
		sslmode = "disable"
	}

	dsn := "host=" + host + " user=" + user + " dbname=" + database + " port=" + port + " sslmode=" + sslmode
	if password != "" {
		dsn += " password=" + password
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, false
	}
	db.SetMaxOpenConns(2)
	return db, true
}

func applyMigrations(db *sql.DB) error {
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

	for _, file := range files {
		body, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if strings.TrimSpace(string(body)) == "" {
			continue
		}
		if _, err := db.Exec(string(body)); err != nil {
			return err
		}
	}
	return nil
}

func repoRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	dir := filepath.Dir(filename)
	return filepath.Clean(filepath.Join(dir, "..", ".."))
}
