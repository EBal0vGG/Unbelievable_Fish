package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	tradingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/http"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/http/handler"
	tradingpg "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	db, ok := openDB()
	if !ok {
		log.Fatal("PGHOST/PGUSER/PGDATABASE are required")
	}
	defer db.Close()

	uow := tradingpg.NewUnitOfWork(db)
	placeBidUC, err := tradingapp.NewPlaceBid(uow)
	if err != nil {
		log.Fatal(err)
	}

	router := httpapi.NewRouter(
		handler.NewPlaceBidHandler(placeBidUC),
	)

	port := envOrDefault("TRADING_PORT", "8082")
	log.Printf("trading http listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
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
	db.SetMaxOpenConns(5)
	return db, true
}

func envOrDefault(key, def string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return def
}
