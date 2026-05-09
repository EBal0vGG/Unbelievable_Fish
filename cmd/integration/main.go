package main

import (
	"context"
	"os"
	"strconv"
	"time"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/payment/fake"
	billingpg "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/postgres"
	catalogapp "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
	catalogpg "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/postgres"
	dealsapp "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	dealspg "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/postgres"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/dbconfig"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/logging"
	integration "github.com/EBal0vGG/Unbelievable_Fish/internal/integration/runtime"
	tradingpg "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/postgres"
)

func runTicker(ctx context.Context, interval time.Duration, fn func(context.Context) error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = fn(ctx)
		}
	}
}

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
	billingOutbox := billingpg.NewOutboxRepository(db)
	createBillingAccount, err := billingapp.NewCreateAccount(
		billingAccounts,
		billingapp.RandomHexID{},
		billingOutbox,
	)
	if err != nil {
		logging.Fatal(logger, "billing_create_account_init_failed", "error", err)
	}
	releaseExcept, err := billingapp.NewReleaseAuctionDepositsExceptCandidates(
		billingAccounts,
		billingpg.NewAuctionDepositRepository(db),
		billingpg.NewLedgerRepository(db),
		billingapp.RandomHexID{},
		nil,
		billingOutbox,
	)
	if err != nil {
		logging.Fatal(logger, "billing_release_except_init_failed", "error", err)
	}

	captureDeposit, err := billingapp.NewCaptureAuctionDeposit(
		billingAccounts,
		billingpg.NewAuctionDepositRepository(db),
		billingpg.NewLedgerRepository(db),
		billingapp.RandomHexID{},
		nil,
		billingOutbox,
	)
	if err != nil {
		logging.Fatal(logger, "billing_capture_deposit_init_failed", "error", err)
	}

	auctionDeposits := billingpg.NewAuctionDepositRepository(db)
	dealInvoiceRepo := billingpg.NewDealInvoiceRepository(db)
	billingPublicBase := dbconfig.EnvOrDefault("BILLING_PUBLIC_BASE_URL", "http://localhost:8085")
	createDealInvoice, err := billingapp.NewCreateDealInvoice(
		dealInvoiceRepo,
		auctionDeposits,
		fake.Provider{},
		fake.ProviderName,
		billingapp.RandomHexID{},
		nil,
		billingOutbox,
		billingPublicBase,
	)
	if err != nil {
		logging.Fatal(logger, "billing_create_deal_invoice_init_failed", "error", err)
	}

	billingLedger := billingpg.NewLedgerRepository(db)
	settleWinnerDeposit, err := billingapp.NewSettleWinnerDepositAfterInvoicePaid(
		billingAccounts,
		auctionDeposits,
		billingLedger,
		billingapp.RandomHexID{},
		nil,
		billingOutbox,
	)
	if err != nil {
		logging.Fatal(logger, "billing_settle_winner_deposit_init_failed", "error", err)
	}

	handleDealInvoicePaid, err := dealsapp.NewHandleDealInvoicePaid(dealsUOW)
	if err != nil {
		logging.Fatal(logger, "deals_handle_deal_invoice_paid_init_failed", "error", err)
	}
	handleDealInvoiceExpired, err := dealsapp.NewHandleDealInvoiceExpired(dealsUOW, nil)
	if err != nil {
		logging.Fatal(logger, "deals_handle_deal_invoice_expired_init_failed", "error", err)
	}
	sellerPayoutRepo := billingpg.NewSellerPayoutRepository(db)
	createSellerPayout, err := billingapp.NewCreateSellerPayout(
		sellerPayoutRepo,
		dealInvoiceRepo,
		billingapp.RandomHexID{},
		nil,
		billingOutbox,
	)
	if err != nil {
		logging.Fatal(logger, "billing_create_seller_payout_init_failed", "error", err)
	}
	expireDealInvoice, err := billingapp.NewExpireDealInvoice(dealInvoiceRepo, billingOutbox, nil)
	if err != nil {
		logging.Fatal(logger, "billing_expire_deal_invoice_init_failed", "error", err)
	}
	invoiceDeadlineLister := billingpg.NewDealInvoiceLister(db)

	runtime, err := integration.New(db, integration.Dependencies{
		Catalog:        catalogService,
		TradingUOW:     tradingUOW,
		DealsUOW:       dealsUOW,
		ProjectionRepo: projectionRepo,
		AuctionLister:  auctionLister,
		DealLister:     dealLister,
		BillingTx:      billingpg.NewTransactionManager(db, nil),
		CreateAccount:  createBillingAccount,
		ReleaseAuctionDepositsExceptCandidates: releaseExcept,
		CaptureAuctionDeposit:                  captureDeposit,
		CreateDealInvoice:                      createDealInvoice,
		HandleDealInvoicePaid:                  handleDealInvoicePaid,
		HandleDealInvoiceExpired:               handleDealInvoiceExpired,
		ExpireDealInvoice:                      expireDealInvoice,
		ExpiredDealInvoiceLister:               invoiceDeadlineLister,
		SettleWinnerDepositAfterInvoicePaid:    settleWinnerDeposit,
		DealInvoices:                           dealInvoiceRepo,
		CreateSellerPayout:                     createSellerPayout,
	})
	if err != nil {
		logging.Fatal(logger, "integration_runtime_init_failed", "error", err)
	}

	ctx := context.Background()
	closeInterval := envDurationSeconds("AUCTION_CLOSE_INTERVAL_SEC", 10)
	closeLimit := envInt("AUCTION_CLOSE_BATCH", 100)
	dealDeadlineInterval := envDurationSeconds("DEAL_DEADLINE_INTERVAL_SEC", 10)
	dealDeadlineLimit := envInt("DEAL_DEADLINE_BATCH", 100)
	invoiceExpireInterval := envDurationSeconds("BILLING_INVOICE_EXPIRE_INTERVAL_SEC", 10)
	invoiceExpireBatch := envInt("BILLING_INVOICE_EXPIRE_BATCH", 100)

	go runTicker(ctx, closeInterval, func(ctx context.Context) error {
		if err := runtime.RunCloseExpired(ctx, time.Now().UTC(), closeLimit); err != nil {
			logger.Error("close_expired_auctions_failed", "component", "scheduler", "error", err)
		}
		return nil
	})
	go runTicker(ctx, dealDeadlineInterval, func(ctx context.Context) error {
		if err := runtime.RunCancelExpiredDeals(ctx, time.Now().UTC(), dealDeadlineLimit); err != nil {
			logger.Error("cancel_expired_deals_failed", "component", "scheduler", "error", err)
		}
		return nil
	})
	go runTicker(ctx, invoiceExpireInterval, func(ctx context.Context) error {
		if err := runtime.RunExpireDealInvoices(ctx, time.Now().UTC(), invoiceExpireBatch); err != nil {
			logger.Error("expire_deal_invoices_failed", "component", "scheduler", "error", err)
		}
		return nil
	})
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
