package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	dealsapp "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	dealspg "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/postgres"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/dbconfig"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/logging"
	tradingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	tradingpg "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/postgres"
)

func main() {
	logger := logging.New("admin")
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	command := os.Args[1]
	auctionID := os.Getenv("AUCTION_ID")
	if auctionID == "" {
		logging.Fatal(logger, "auction_id_missing", "required", "AUCTION_ID")
	}

	db, ok := dbconfig.OpenPostgresFromEnv(0)
	if !ok {
		logging.Fatal(logger, "database_config_missing", "required", "PGHOST,PGUSER,PGDATABASE")
	}
	defer db.Close()

	switch command {
	case "close-auction":
		logger.Info("admin_command_started", "command", command, "auction_id", auctionID, "correlation_id", os.Getenv("CORRELATION_ID"))
		if err := closeAuction(db, auctionID); err != nil {
			logging.Fatal(logger, "admin_command_failed", "command", command, "auction_id", auctionID, "error", err)
		}
		logger.Info("admin_command_completed", "command", command, "auction_id", auctionID)
	case "decline-deal":
		logger.Info("admin_command_started", "command", command, "auction_id", auctionID, "correlation_id", os.Getenv("CORRELATION_ID"))
		if err := declineDeal(db, auctionID); err != nil {
			logging.Fatal(logger, "admin_command_failed", "command", command, "auction_id", auctionID, "deal_id", os.Getenv("DEAL_ID"), "error", err)
		}
		logger.Info("admin_command_completed", "command", command, "auction_id", auctionID, "deal_id", os.Getenv("DEAL_ID"))
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
		CompanyID:     dbconfig.EnvOrDefault("COMPANY_ID", "system"),
		UserID:        dbconfig.EnvOrDefault("USER_ID", "system"),
		CorrelationID: dbconfig.EnvOrDefault("CORRELATION_ID", "admin-close"),
		CausationID:   dbconfig.EnvOrDefault("CAUSATION_ID", "admin-close"),
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
		CompanyID:     dbconfig.EnvOrDefault("COMPANY_ID", "system"),
		UserID:        dbconfig.EnvOrDefault("USER_ID", "system"),
		CorrelationID: dbconfig.EnvOrDefault("CORRELATION_ID", "admin-decline"),
		CausationID:   dbconfig.EnvOrDefault("CAUSATION_ID", "admin-decline"),
	}
	dealID := os.Getenv("DEAL_ID")
	return uc.Execute(context.Background(), meta, auctionID, dealID)
}

func usage() {
	fmt.Println("Usage: admin <close-auction|decline-deal>")
	fmt.Println("Required env: AUCTION_ID, PGHOST, PGUSER, PGDATABASE")
	fmt.Println("Optional env: DEAL_ID (for idempotent decline)")
}
