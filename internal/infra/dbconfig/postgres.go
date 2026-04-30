package dbconfig

import (
	"database/sql"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	defaultPGPort   = "5432"
	defaultSSLMode  = "disable"
	defaultMaxConns = 5
)

// OpenPostgresFromEnv opens a *sql.DB from PGHOST, PGUSER, PGPASSWORD, PGDATABASE, PGPORT, PGSSLMODE.
// Returns (nil, false) if required vars are missing or sql.Open fails.
// maxOpenConns <= 0 defaults to 5; migrate uses 2 explicitly.
func OpenPostgresFromEnv(maxOpenConns int) (*sql.DB, bool) {
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
		port = defaultPGPort
	}
	if sslmode == "" {
		sslmode = defaultSSLMode
	}

	dsn := "host=" + host + " user=" + user + " dbname=" + database + " port=" + port + " sslmode=" + sslmode
	if password != "" {
		dsn += " password=" + password
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, false
	}
	if maxOpenConns <= 0 {
		maxOpenConns = defaultMaxConns
	}
	db.SetMaxOpenConns(maxOpenConns)
	return db, true
}
