package integration

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	billingpg "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/postgres"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
	catalogapp "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
	catalog "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/domain"
	catalogpg "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/postgres"
	dealsapp "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	deal "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
	dealspg "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/postgres"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/dbconfig"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/eventbus/inmemory"
	outbox "github.com/EBal0vGG/Unbelievable_Fish/internal/infra/outbox"
	outboxpg "github.com/EBal0vGG/Unbelievable_Fish/internal/infra/outbox/postgres"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/shared/events"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/adapters/billingdeposit"
	tradingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	auction "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
	tradingpg "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresOutboxBusChains_RealPG(t *testing.T) {
	db, ok := openRealPostgres(t)
	if !ok {
		return
	}
	if err := applyMigrations(t, db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := truncateAll(t, db); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	catalogLotRepo := catalogpg.NewLotRepository(db)
	catalogOutbox := catalogpg.NewOutboxRepository(db)
	catalogTx := catalogpg.NewTransactionManager(db, nil)
	catalogService := catalogapp.NewCatalogService(
		nil, nil, nil, nil,
		catalogLotRepo,
		catalogOutbox,
		nil,
		catalogTx,
	)

	tradingUOW := tradingpg.NewUnitOfWork(db)
	dealsUOW := dealspg.NewUnitOfWork(db)
	dealProjectionRepo := dealspg.NewProjectionRepository(db)

	bus := inmemory.NewBus()
	relayRepo := outboxpg.NewRepository(db)
	relay := outbox.NewRelay(relayRepo, map[string]outbox.Decoder{
		"catalog.LotPublished":       outbox.JSONDecoder[catalog.LotPublished](),
		"trading.AuctionPublished":   outbox.JSONDecoder[auction.AuctionPublished](),
		"trading.BidPlaced":          outbox.JSONDecoder[auction.BidPlaced](),
		"trading.AuctionClosed":      outbox.JSONDecoder[auction.AuctionClosed](),
		"trading.AuctionWon":         outbox.JSONDecoder[auction.AuctionWon](),
		"billing.AccountCreated":     outbox.JSONDecoder[wallet.AccountCreated](),
		"billing.BalanceToppedUp":    outbox.JSONDecoder[wallet.BalanceToppedUp](),
		"billing.AuctionDepositReserved": outbox.JSONDecoder[wallet.AuctionDepositReserved](),
	})

	startsAt := time.Now().Add(-time.Hour)
	endsAt := time.Now().Add(time.Hour)
	auctionID := "auc-real"
	lotID := "lot-real"

	lot, _, err := catalog.NewLot(
		lotID,
		"prod-1",
		"seller-1",
		"photo",
		10,
		100,
		10,
		catalog.NewAuctionScheduleAt(startsAt, time.Hour),
	)
	if err != nil {
		t.Fatalf("new lot error: %v", err)
	}
	if _, err := lot.AssignAuctionID(auctionID); err != nil {
		t.Fatalf("assign auction error: %v", err)
	}
	lotEvents, err := lot.Publish(true, catalog.ProductSnapshot{
		ProductID:      "prod-1",
		Name:           "Fish",
		Weight:         10,
		Unit:           "kg",
		Size:           "M",
		ProcessingType: catalog.ProcessingType("frozen"),
	})
	if err != nil {
		t.Fatalf("publish lot error: %v", err)
	}
	if err := catalogLotRepo.Save(context.Background(), lot); err != nil {
		t.Fatalf("save lot error: %v", err)
	}
	if err := catalogOutbox.Add(context.Background(), lotEvents); err != nil {
		t.Fatalf("outbox add error: %v", err)
	}

	createAuctionUC, err := tradingapp.NewCreateAuction(tradingUOW, fixedAuctionIDFactoryReal{auctionID: tradingapp.AuctionID(auctionID)})
	if err != nil {
		t.Fatalf("create auction constructor error: %v", err)
	}
	publishAuctionUC, err := tradingapp.NewPublishAuction(tradingUOW)
	if err != nil {
		t.Fatalf("publish auction constructor error: %v", err)
	}
	createProjectionUC := dealsapp.NewCreateProjection(dealProjectionRepo)
	createSelectionUC, err := dealsapp.NewCreateDealSelectionFromAuctionWon(dealsUOW)
	if err != nil {
		t.Fatalf("selection constructor error: %v", err)
	}

	bus.Subscribe("catalog.LotPublished", func(ctx context.Context, envelope events.Envelope) error {
		evt := envelope.Payload.(catalog.LotPublished)
		if _, err := createAuctionUC.Execute(ctx, tradingMeta(), evt.LotID, startsAt, endsAt, evt.StartPrice, evt.MinBidStep); err != nil {
			return err
		}
		if err := publishAuctionUC.Execute(ctx, tradingMeta(), tradingapp.AuctionID(evt.AuctionID)); err != nil {
			return err
		}
		return createProjectionUC.Execute(ctx, dealsMeta(), evt.AuctionID, evt.SellerCompanyID, deal.ProductSnapshot{Name: "Fish"}, evt.StartPrice, envelope.OccurredAt)
	})

	bus.Subscribe("trading.AuctionWon", func(ctx context.Context, envelope events.Envelope) error {
		evt := envelope.Payload.(auction.AuctionWon)
		if len(evt.WinnerCompanyID) == 0 {
			return nil
		}
		if err := createSelectionUC.Execute(ctx, dealsMeta(), evt.AuctionID, evt.WinnerCompanyID, evt.FinalPrice, envelope.OccurredAt); err != nil {
			return err
		}
		return catalogService.HandleAuctionWon(ctx, catalogapp.AuctionWonDTO{
			AuctionID:       evt.AuctionID,
			FinalPrice:      evt.FinalPrice,
			WinnerCompanyID: evt.WinnerCompanyID[0],
		})
	})

	if err := relay.RunOnce(context.Background(), bus, 100); err != nil {
		t.Fatalf("relay lot published error: %v", err)
	}

	bootstrapBuyerWalletReal(t, db, "buyer-1", 500_000)
	depositSvc := newBillingDepositService(t, db)
	placeBidUC, err := tradingapp.NewPlaceBid(tradingUOW, depositSvc)
	if err != nil {
		t.Fatalf("place bid constructor error: %v", err)
	}
	closeAuctionUC, err := tradingapp.NewCloseAuction(tradingUOW)
	if err != nil {
		t.Fatalf("close auction constructor error: %v", err)
	}
	if err := placeBidUC.Execute(context.Background(), tradingMetaWithCompany("buyer-1"), tradingapp.AuctionID(auctionID), 150, endsAt.Add(-time.Minute)); err != nil {
		t.Fatalf("place bid error: %v", err)
	}
	if err := closeAuctionUC.Execute(context.Background(), tradingMeta(), tradingapp.AuctionID(auctionID)); err != nil {
		t.Fatalf("close auction error: %v", err)
	}

	if err := relay.RunOnce(context.Background(), bus, 100); err != nil {
		t.Fatalf("relay auction won error: %v", err)
	}

	dealItem, err := dealspg.NewDealRepository(db).GetByAuctionID(context.Background(), auctionID)
	if err != nil {
		t.Fatalf("expected deal after auction won: %v", err)
	}
	if dealItem.CustomerID() != "buyer-1" {
		t.Fatalf("expected deal for buyer-1, got %s", dealItem.CustomerID())
	}
}

func bootstrapBuyerWalletReal(t *testing.T, db *sql.DB, buyerID string, credit int64) {
	t.Helper()
	txm := billingpg.NewTransactionManager(db, nil)
	accounts := billingpg.NewAccountRepository(db)
	ledger := billingpg.NewLedgerRepository(db)
	processed := billingpg.NewProcessedTopUpRepository(db)
	events := billingpg.NewOutboxRepository(db)
	createAccount, err := billingapp.NewCreateAccount(accounts, billingapp.RandomHexID{}, events)
	if err != nil {
		t.Fatalf("create account uc: %v", err)
	}
	confirmTopUp, err := billingapp.NewConfirmTopUp(accounts, ledger, processed, billingapp.RandomHexID{}, nil, events)
	if err != nil {
		t.Fatalf("confirm top up uc: %v", err)
	}
	ctx := context.Background()
	extID := "bootstrap-real:" + buyerID
	if err := txm.WithinTx(ctx, func(txCtx context.Context) error {
		if err := createAccount.Execute(txCtx, buyerID); err != nil {
			return err
		}
		return confirmTopUp.Execute(txCtx, buyerID, credit, extID)
	}); err != nil {
		t.Fatalf("bootstrap wallet: %v", err)
	}
}

func newBillingDepositService(t *testing.T, db *sql.DB) *billingdeposit.Service {
	t.Helper()
	accounts := billingpg.NewAccountRepository(db)
	ledger := billingpg.NewLedgerRepository(db)
	deposits := billingpg.NewAuctionDepositRepository(db)
	events := billingpg.NewOutboxRepository(db)
	createAccount, err := billingapp.NewCreateAccount(accounts, billingapp.RandomHexID{}, events)
	if err != nil {
		t.Fatal(err)
	}
	reserve, err := billingapp.NewReserveAuctionDeposit(accounts, deposits, ledger, billingapp.RandomHexID{}, nil, events)
	if err != nil {
		t.Fatal(err)
	}
	return billingdeposit.NewService(createAccount, reserve)
}

func TestPlaceBidTwoBidsSameCompanySingleDeposit_RealPG(t *testing.T) {
	db, ok := openRealPostgres(t)
	if !ok {
		return
	}
	if err := applyMigrations(t, db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := truncateAll(t, db); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	ctx := context.Background()
	uow := tradingpg.NewUnitOfWork(db)
	depositSvc := newBillingDepositService(t, db)

	startsAt := time.Now().Add(-time.Hour)
	endsAt := time.Now().Add(time.Hour)
	auctionID := tradingapp.AuctionID("auc-deposit-2x")
	const buyer = "buyer-deposit-2x"
	const startPrice int64 = 100_000
	const minStep int64 = 10_000

	createUC, err := tradingapp.NewCreateAuction(uow, fixedAuctionIDFactoryReal{auctionID: auctionID})
	if err != nil {
		t.Fatal(err)
	}
	publishUC, err := tradingapp.NewPublishAuction(uow)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := createUC.Execute(ctx, tradingMeta(), "lot-x", startsAt, endsAt, startPrice, minStep); err != nil {
		t.Fatalf("create auction: %v", err)
	}
	if err := publishUC.Execute(ctx, tradingMeta(), auctionID); err != nil {
		t.Fatalf("publish: %v", err)
	}

	bootstrapBuyerWalletReal(t, db, buyer, 500_000)
	placeBidUC, err := tradingapp.NewPlaceBid(uow, depositSvc)
	if err != nil {
		t.Fatal(err)
	}
	meta := tradingapp.CommandMeta{CompanyID: buyer, UserID: buyer, CorrelationID: "c1", CausationID: "c1"}
	t1 := endsAt.Add(-30 * time.Minute)
	if err := placeBidUC.Execute(ctx, meta, auctionID, 100_000, t1); err != nil {
		t.Fatalf("first bid: %v", err)
	}
	if err := placeBidUC.Execute(ctx, meta, auctionID, 110_000, t1.Add(time.Minute)); err != nil {
		t.Fatalf("second bid: %v", err)
	}

	var reserveCount int
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM billing_ledger_entries
WHERE company_id = $1 AND type = $2 AND reference_type = 'auction_deposit'
`, buyer, string(wallet.LedgerBidDepositReserved)).Scan(&reserveCount); err != nil {
		t.Fatalf("count ledger: %v", err)
	}
	if reserveCount != 1 {
		t.Fatalf("expected one BID_DEPOSIT_RESERVED ledger row, got %d", reserveCount)
	}
}

func TestAuctionWonReleaseNonTopDeposits_RealPG(t *testing.T) {
	db, ok := openRealPostgres(t)
	if !ok {
		return
	}
	if err := applyMigrations(t, db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := truncateAll(t, db); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	catalogLotRepo := catalogpg.NewLotRepository(db)
	catalogOutbox := catalogpg.NewOutboxRepository(db)
	catalogTx := catalogpg.NewTransactionManager(db, nil)
	catalogService := catalogapp.NewCatalogService(
		nil, nil, nil, nil,
		catalogLotRepo,
		catalogOutbox,
		nil,
		catalogTx,
	)

	tradingUOW := tradingpg.NewUnitOfWork(db)
	dealsUOW := dealspg.NewUnitOfWork(db)
	dealProjectionRepo := dealspg.NewProjectionRepository(db)

	bus := inmemory.NewBus()
	relayRepo := outboxpg.NewRepository(db)
	relay := outbox.NewRelay(relayRepo, map[string]outbox.Decoder{
		"catalog.LotPublished":           outbox.JSONDecoder[catalog.LotPublished](),
		"trading.AuctionPublished":       outbox.JSONDecoder[auction.AuctionPublished](),
		"trading.BidPlaced":              outbox.JSONDecoder[auction.BidPlaced](),
		"trading.AuctionClosed":          outbox.JSONDecoder[auction.AuctionClosed](),
		"trading.AuctionWon":             outbox.JSONDecoder[auction.AuctionWon](),
		"billing.AccountCreated":         outbox.JSONDecoder[wallet.AccountCreated](),
		"billing.BalanceToppedUp":        outbox.JSONDecoder[wallet.BalanceToppedUp](),
		"billing.AuctionDepositReserved": outbox.JSONDecoder[wallet.AuctionDepositReserved](),
		"billing.AuctionDepositReleased": outbox.JSONDecoder[wallet.AuctionDepositReleased](),
	})

	startsAt := time.Now().Add(-time.Hour)
	endsAt := time.Now().Add(time.Hour)
	auctionID := "auc-top4"
	lotID := "lot-top4"

	lot, _, err := catalog.NewLot(
		lotID,
		"prod-1",
		"seller-1",
		"photo",
		10,
		100,
		10,
		catalog.NewAuctionScheduleAt(startsAt, time.Hour),
	)
	if err != nil {
		t.Fatalf("new lot error: %v", err)
	}
	if _, err := lot.AssignAuctionID(auctionID); err != nil {
		t.Fatalf("assign auction error: %v", err)
	}
	lotEvents, err := lot.Publish(true, catalog.ProductSnapshot{
		ProductID:      "prod-1",
		Name:           "Fish",
		Weight:         10,
		Unit:           "kg",
		Size:           "M",
		ProcessingType: catalog.ProcessingType("frozen"),
	})
	if err != nil {
		t.Fatalf("publish lot error: %v", err)
	}
	if err := catalogLotRepo.Save(context.Background(), lot); err != nil {
		t.Fatalf("save lot error: %v", err)
	}
	if err := catalogOutbox.Add(context.Background(), lotEvents); err != nil {
		t.Fatalf("outbox add error: %v", err)
	}

	createAuctionUC, err := tradingapp.NewCreateAuction(tradingUOW, fixedAuctionIDFactoryReal{auctionID: tradingapp.AuctionID(auctionID)})
	if err != nil {
		t.Fatalf("create auction constructor error: %v", err)
	}
	publishAuctionUC, err := tradingapp.NewPublishAuction(tradingUOW)
	if err != nil {
		t.Fatalf("publish auction constructor error: %v", err)
	}
	createProjectionUC := dealsapp.NewCreateProjection(dealProjectionRepo)
	createSelectionUC, err := dealsapp.NewCreateDealSelectionFromAuctionWon(dealsUOW)
	if err != nil {
		t.Fatalf("selection constructor error: %v", err)
	}

	billingTx := billingpg.NewTransactionManager(db, nil)
	bAccounts := billingpg.NewAccountRepository(db)
	bLedger := billingpg.NewLedgerRepository(db)
	bDeposits := billingpg.NewAuctionDepositRepository(db)
	bEvents := billingpg.NewOutboxRepository(db)
	releaseExcept, err := billingapp.NewReleaseAuctionDepositsExceptCandidates(
		bAccounts, bDeposits, bLedger, billingapp.RandomHexID{}, nil, bEvents,
	)
	if err != nil {
		t.Fatalf("release except uc: %v", err)
	}

	bus.Subscribe("catalog.LotPublished", func(ctx context.Context, envelope events.Envelope) error {
		evt := envelope.Payload.(catalog.LotPublished)
		if _, err := createAuctionUC.Execute(ctx, tradingMeta(), evt.LotID, startsAt, endsAt, evt.StartPrice, evt.MinBidStep); err != nil {
			return err
		}
		if err := publishAuctionUC.Execute(ctx, tradingMeta(), tradingapp.AuctionID(evt.AuctionID)); err != nil {
			return err
		}
		return createProjectionUC.Execute(ctx, dealsMeta(), evt.AuctionID, evt.SellerCompanyID, deal.ProductSnapshot{Name: "Fish"}, evt.StartPrice, envelope.OccurredAt)
	})

	bus.Subscribe("trading.AuctionWon", func(ctx context.Context, envelope events.Envelope) error {
		evt := envelope.Payload.(auction.AuctionWon)
		if len(evt.WinnerCompanyID) == 0 {
			return nil
		}
		if err := createSelectionUC.Execute(ctx, dealsMeta(), evt.AuctionID, evt.WinnerCompanyID, evt.FinalPrice, envelope.OccurredAt); err != nil {
			return err
		}
		if err := catalogService.HandleAuctionWon(ctx, catalogapp.AuctionWonDTO{
			AuctionID:       evt.AuctionID,
			FinalPrice:      evt.FinalPrice,
			WinnerCompanyID: evt.WinnerCompanyID[0],
		}); err != nil {
			return err
		}
		return billingTx.WithinTx(ctx, func(txCtx context.Context) error {
			return releaseExcept.Execute(txCtx, evt.AuctionID, evt.WinnerCompanyID, "LOST_AUCTION")
		})
	})

	if err := relay.RunOnce(context.Background(), bus, 100); err != nil {
		t.Fatalf("relay lot published error: %v", err)
	}

	ctx := context.Background()
	buyers := []string{"buyer-a", "buyer-b", "buyer-c", "buyer-d"}
	for _, b := range buyers {
		bootstrapBuyerWalletReal(t, db, b, 500_000)
	}
	depositSvc := newBillingDepositService(t, db)
	placeBidUC, err := tradingapp.NewPlaceBid(tradingUOW, depositSvc)
	if err != nil {
		t.Fatalf("place bid constructor error: %v", err)
	}
	closeAuctionUC, err := tradingapp.NewCloseAuction(tradingUOW)
	if err != nil {
		t.Fatalf("close auction constructor error: %v", err)
	}

	// Lot start price 100, min step 10 — each bid must beat current high by >= min step.
	amounts := []int64{150, 160, 170, 180}
	for i, b := range buyers {
		meta := tradingMetaWithCompany(b)
		tBid := endsAt.Add(time.Duration(-40+10*i) * time.Minute)
		if err := placeBidUC.Execute(ctx, meta, tradingapp.AuctionID(auctionID), amounts[i], tBid); err != nil {
			t.Fatalf("place bid %s: %v", b, err)
		}
	}

	if err := closeAuctionUC.Execute(ctx, tradingMeta(), tradingapp.AuctionID(auctionID)); err != nil {
		t.Fatalf("close auction error: %v", err)
	}

	if err := relay.RunOnce(ctx, bus, 100); err != nil {
		t.Fatalf("relay auction won error: %v", err)
	}

	// Top 3 bid rows by amount: 180 (d), 170 (c), 160 (b); lowest high bid 150 (a) is not in top-3 slice.
	for _, b := range []string{"buyer-b", "buyer-c", "buyer-d"} {
		var st string
		if err := db.QueryRowContext(ctx, `
SELECT status FROM billing_auction_deposits WHERE auction_id = $1 AND company_id = $2
`, auctionID, b).Scan(&st); err != nil {
			t.Fatalf("deposit row %s: %v", b, err)
		}
		if st != "HELD" {
			t.Fatalf("expected HELD for %s, got %s", b, st)
		}
	}
	var aStatus string
	if err := db.QueryRowContext(ctx, `
SELECT status FROM billing_auction_deposits WHERE auction_id = $1 AND company_id = 'buyer-a'
`, auctionID).Scan(&aStatus); err != nil {
		t.Fatalf("deposit buyer-a: %v", err)
	}
	if aStatus != "RELEASED" {
		t.Fatalf("expected buyer-a RELEASED, got %s", aStatus)
	}

	var releaseCount int
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM billing_ledger_entries
WHERE company_id = 'buyer-a' AND type = $1
`, string(wallet.LedgerBidDepositReleased)).Scan(&releaseCount); err != nil {
		t.Fatalf("count release ledger: %v", err)
	}
	if releaseCount != 1 {
		t.Fatalf("expected one BID_DEPOSIT_RELEASED for buyer-a, got %d", releaseCount)
	}

	// Idempotent second run (e.g. replay): no extra ledger row.
	if err := billingTx.WithinTx(ctx, func(txCtx context.Context) error {
		return releaseExcept.Execute(txCtx, auctionID, []string{"buyer-d", "buyer-c", "buyer-b"}, "LOST_AUCTION")
	}); err != nil {
		t.Fatalf("second release except: %v", err)
	}
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM billing_ledger_entries
WHERE company_id = 'buyer-a' AND type = $1
`, string(wallet.LedgerBidDepositReleased)).Scan(&releaseCount); err != nil {
		t.Fatalf("count release ledger after replay: %v", err)
	}
	if releaseCount != 1 {
		t.Fatalf("after idempotent replay expected 1 release ledger, got %d", releaseCount)
	}
}

type fixedAuctionIDFactoryReal struct {
	auctionID tradingapp.AuctionID
}

func (f fixedAuctionIDFactoryReal) NewID() (tradingapp.AuctionID, error) {
	return f.auctionID, nil
}

func openRealPostgres(t *testing.T) (*sql.DB, bool) {
	t.Helper()

	db, err := dbconfig.OpenPostgresDockerComposeDefaults(5)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		t.Skipf("postgres not reachable (%v); start: docker compose up -d postgres (defaults match docker-compose.yml)", err)
		return nil, false
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, true
}

func applyMigrations(t *testing.T, db *sql.DB) error {
	t.Helper()

	root := repoRootReal(t)
	migrationsDir := filepath.Join(root, "migrations")
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

func truncateAll(t *testing.T, db *sql.DB) error {
	t.Helper()
	_, err := db.Exec(`
TRUNCATE TABLE
    outbox_messages,
    billing_processed_top_ups,
    billing_ledger_entries,
    billing_auction_deposits,
    billing_top_ups,
    billing_accounts,
    trading_auction_winners,
    trading_bids,
    trading_auctions,
    catalog_lots,
    deal_winner_selections,
    deal_confirmations,
    deal_projections,
    deals
`)
	return err
}

func repoRootReal(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("cannot resolve caller")
	}
	dir := filepath.Dir(filename)
	return filepath.Clean(filepath.Join(dir, "..", ".."))
}
