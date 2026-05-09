package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	dealsapp "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
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

	db, ok := dbconfig.OpenPostgresFromEnv(0)
	if !ok {
		logging.Fatal(logger, "database_config_missing", "required", "PGHOST,PGUSER,PGDATABASE")
	}
	defer db.Close()

	switch command {
	case "close-auction":
		auctionID := os.Getenv("AUCTION_ID")
		if auctionID == "" {
			logging.Fatal(logger, "auction_id_missing", "required", "AUCTION_ID")
		}
		logger.Info("admin_command_started", "command", command, "auction_id", auctionID, "correlation_id", os.Getenv("CORRELATION_ID"))
		if err := closeAuction(db, auctionID); err != nil {
			logging.Fatal(logger, "admin_command_failed", "command", command, "auction_id", auctionID, "error", err)
		}
		logger.Info("admin_command_completed", "command", command, "auction_id", auctionID)
	case "decline-deal":
		auctionID := os.Getenv("AUCTION_ID")
		if auctionID == "" {
			logging.Fatal(logger, "auction_id_missing", "required", "AUCTION_ID")
		}
		logger.Info("admin_command_started", "command", command, "auction_id", auctionID, "correlation_id", os.Getenv("CORRELATION_ID"))
		if err := declineDeal(db, auctionID); err != nil {
			logging.Fatal(logger, "admin_command_failed", "command", command, "auction_id", auctionID, "deal_id", os.Getenv("DEAL_ID"), "error", err)
		}
		logger.Info("admin_command_completed", "command", command, "auction_id", auctionID, "deal_id", os.Getenv("DEAL_ID"))
	case "cancel-deal":
		dealID := os.Getenv("DEAL_ID")
		if dealID == "" {
			logging.Fatal(logger, "deal_id_missing", "required", "DEAL_ID")
		}
		if os.Getenv("COMPANY_ID") == "" {
			logging.Fatal(logger, "company_id_missing", "required", "COMPANY_ID (must be the deal customer, e.g. winning buyer company)")
		}
		logger.Info("admin_command_started", "command", command, "deal_id", dealID, "correlation_id", os.Getenv("CORRELATION_ID"))
		if err := cancelDeal(db); err != nil {
			logging.Fatal(logger, "admin_command_failed", "command", command, "deal_id", dealID, "error", err)
		}
		logger.Info("admin_command_completed", "command", command, "deal_id", dealID)
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

func cancelDeal(db *sql.DB) error {
	uow := dealspg.NewUnitOfWork(db)
	uc, err := dealsapp.NewCancelDeal(uow)
	if err != nil {
		return err
	}
	meta := dealsapp.CommandMeta{
		CompanyID:     os.Getenv("COMPANY_ID"),
		UserID:        dbconfig.EnvOrDefault("USER_ID", ""),
		CorrelationID: dbconfig.EnvOrDefault("CORRELATION_ID", "admin-cancel-deal"),
		CausationID:   dbconfig.EnvOrDefault("CAUSATION_ID", "admin-cancel-deal"),
	}
	reason := dbconfig.EnvOrDefault("CANCEL_REASON", deal.DealCancelReasonWinnerRejected)
	return uc.Execute(context.Background(), meta, os.Getenv("DEAL_ID"), reason)
}

func usage() {
	fmt.Println("Usage: admin <close-auction|decline-deal|cancel-deal>")
	fmt.Println("  close-auction: AUCTION_ID")
	fmt.Println("  decline-deal:  AUCTION_ID, optional DEAL_ID (deal must already be cancelled)")
	fmt.Println("  cancel-deal:   DEAL_ID, COMPANY_ID (deal customer); optional CANCEL_REASON (default WINNER_REJECTED)")
	fmt.Println("Shared: PGHOST, PGUSER, PGDATABASE, optional PGPASSWORD, PGPORT, PGSSLMODE")
}
