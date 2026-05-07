package main

import (
	"context"
	"os"
	"strconv"
	"time"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	billingpg "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/postgres"
	catalogapp "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
	catalogpg "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/postgres"
	dealspg "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/postgres"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/dbconfig"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/logging"
	integration "github.com/EBal0vGG/Unbelievable_Fish/internal/integration/runtime"
	tradingpg "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/postgres"
)

func main() {
	logger := logging.New("integration")
	db, ok := dbconfig.OpenPostgresFromEnv(0)
	if !ok {
		logging.Fatal(logger, "database_config_missing", "required", "PGHOST,PGUSER,PGDATABASE")
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
	dealLister := dealspg.NewDealDeadlineLister(db)
	auctionLister := tradingpg.NewAuctionLister(db)

	billingAccounts := billingpg.NewAccountRepository(db)
	createBillingAccount, err := billingapp.NewCreateAccount(
		billingAccounts,
		billingapp.RandomHexID{},
		billingpg.NewOutboxRepository(db),
	)
	if err != nil {
		logging.Fatal(logger, "billing_create_account_init_failed", "error", err)
	}

	runtime, err := integration.New(db, integration.Dependencies{
		Catalog:        catalogService,
		TradingUOW:     tradingUOW,
		DealsUOW:       dealsUOW,
		ProjectionRepo: projectionRepo,
		AuctionLister:  auctionLister,
		DealLister:     dealLister,
		BillingTx:      billingpg.NewTransactionManager(db, nil),
		CreateAccount:  createBillingAccount,
	})
	if err != nil {
		logging.Fatal(logger, "integration_runtime_init_failed", "error", err)
	}

	ctx := context.Background()
	closeInterval := envDurationSeconds("AUCTION_CLOSE_INTERVAL_SEC", 10)
	closeLimit := envInt("AUCTION_CLOSE_BATCH", 100)
	dealDeadlineInterval := envDurationSeconds("DEAL_DEADLINE_INTERVAL_SEC", 10)
	dealDeadlineLimit := envInt("DEAL_DEADLINE_BATCH", 100)
	go func() {
		ticker := time.NewTicker(closeInterval)
		defer ticker.Stop()
		for range ticker.C {
			if err := runtime.RunCloseExpired(ctx, time.Now().UTC(), closeLimit); err != nil {
				logger.Error("close_expired_auctions_failed", "component", "scheduler", "error", err)
			}
		}
	}()
	go func() {
		ticker := time.NewTicker(dealDeadlineInterval)
		defer ticker.Stop()
		for range ticker.C {
			if err := runtime.RunCancelExpiredDeals(ctx, time.Now().UTC(), dealDeadlineLimit); err != nil {
				logger.Error("cancel_expired_deals_failed", "component", "scheduler", "error", err)
			}
		}
	}()
	for {
		if err := runtime.Relay.RunOnce(ctx, runtime.Bus, 100); err != nil {
			logger.Error("outbox_relay_run_failed", "component", "outbox.relay", "error", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
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
