package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	dealsapp "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	dealspg "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/postgres"
	tradingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	tradingpg "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	command := os.Args[1]
	auctionID := os.Getenv("AUCTION_ID")
	if auctionID == "" {
		log.Fatal("AUCTION_ID is required")
	}

	db, ok := openDB()
	if !ok {
		log.Fatal("PGHOST/PGUSER/PGDATABASE are required")
	}
	defer db.Close()

	switch command {
	case "close-auction":
		log.Printf("admin_command command=%s auction_id=%s correlation_id=%s", command, auctionID, os.Getenv("CORRELATION_ID"))
		if err := closeAuction(db, auctionID); err != nil {
			log.Fatalf("close auction: %v", err)
		}
		log.Printf("auction closed: %s", auctionID)
	case "decline-deal":
		log.Printf("admin_command command=%s auction_id=%s correlation_id=%s", command, auctionID, os.Getenv("CORRELATION_ID"))
		if err := declineDeal(db, auctionID); err != nil {
			log.Fatalf("deal decline: %v", err)
		}
		log.Printf("deal declined handled for auction: %s", auctionID)
	default:
		usage()
		os.Exit(2)
	}
}

func closeAuction(db *sql.DB, auctionID string) error {
	uow := tradingpg.NewUnitOfWork(db)
	uc, err := tradingapp.NewCloseAuction(uow)
	if err != nil {
		return err
	}
	meta := tradingapp.CommandMeta{
		CompanyID:     envOrDefault("COMPANY_ID", "system"),
		UserID:        envOrDefault("USER_ID", "system"),
		CorrelationID: envOrDefault("CORRELATION_ID", "admin-close"),
		CausationID:   envOrDefault("CAUSATION_ID", "admin-close"),
	}
	return uc.Execute(context.Background(), meta, tradingapp.AuctionID(auctionID))
}

func declineDeal(db *sql.DB, auctionID string) error {
	uow := dealspg.NewUnitOfWork(db)
	uc, err := dealsapp.NewHandleDealDeclined(uow)
	if err != nil {
		return err
	}
	meta := dealsapp.CommandMeta{
		CompanyID:     envOrDefault("COMPANY_ID", "system"),
		UserID:        envOrDefault("USER_ID", "system"),
		CorrelationID: envOrDefault("CORRELATION_ID", "admin-decline"),
		CausationID:   envOrDefault("CAUSATION_ID", "admin-decline"),
	}
	dealID := os.Getenv("DEAL_ID")
	return uc.Execute(context.Background(), meta, auctionID, dealID)
}

func usage() {
	fmt.Println("Usage: admin <close-auction|decline-deal>")
	fmt.Println("Required env: AUCTION_ID, PGHOST, PGUSER, PGDATABASE")
	fmt.Println("Optional env: DEAL_ID (for idempotent decline)")
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
