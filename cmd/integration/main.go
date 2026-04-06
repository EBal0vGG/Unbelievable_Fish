package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strconv"
	"time"

	catalogapp "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
	catalogpg "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/postgres"
	dealspg "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/postgres"
	integration "github.com/EBal0vGG/Unbelievable_Fish/internal/integration/runtime"
	tradingpg "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	db, ok := openDB()
	if !ok {
		log.Fatal("PGHOST/PGUSER/PGDATABASE are required")
	}
	defer db.Close()

	catalogService := catalogapp.NewCatalogService(
		catalogpg.NewFishRepository(db),
		catalogpg.NewUnitRepository(db),
		catalogpg.NewProcessingTypeRepository(db),
		catalogpg.NewProductRepository(db),
		catalogpg.NewLotRepository(db),
		catalogpg.NewOutboxRepository(db),
		catalogapp.NewRandomIDGenerator(),
		catalogpg.NewTransactionManager(db, nil),
	)

	tradingUOW := tradingpg.NewUnitOfWork(db)
	dealsUOW := dealspg.NewUnitOfWork(db)
	projectionRepo := dealspg.NewProjectionRepository(db)
	auctionLister := tradingpg.NewAuctionLister(db)

	runtime, err := integration.New(db, integration.Dependencies{
		Catalog:        catalogService,
		TradingUOW:     tradingUOW,
		DealsUOW:       dealsUOW,
		ProjectionRepo: projectionRepo,
		AuctionLister:  auctionLister,
	})
	if err != nil {
		log.Fatalf("init integration runtime: %v", err)
	}

	ctx := context.Background()
	closeInterval := envDurationSeconds("AUCTION_CLOSE_INTERVAL_SEC", 10)
	closeLimit := envInt("AUCTION_CLOSE_BATCH", 100)
	go func() {
		ticker := time.NewTicker(closeInterval)
		defer ticker.Stop()
		for range ticker.C {
			if err := runtime.RunCloseExpired(ctx, time.Now().UTC(), closeLimit); err != nil {
				log.Printf("close expired auctions error: %v", err)
			}
		}
	}()
	for {
		if err := runtime.Relay.RunOnce(ctx, runtime.Bus, 100); err != nil {
			log.Printf("relay error: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
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

func envDurationSeconds(key string, def int) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return time.Duration(def) * time.Second
	}
	parsed, err := time.ParseDuration(value + "s")
	if err != nil {
		return time.Duration(def) * time.Second
	}
	return parsed
}

func envInt(key string, def int) int {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return def
	}
	return parsed
}
