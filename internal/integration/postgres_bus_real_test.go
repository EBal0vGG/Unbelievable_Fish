package integration

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/payment/fake"
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
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/integrationtest"
	outbox "github.com/EBal0vGG/Unbelievable_Fish/internal/infra/outbox"
	outboxpg "github.com/EBal0vGG/Unbelievable_Fish/internal/infra/outbox/postgres"
	integrationruntime "github.com/EBal0vGG/Unbelievable_Fish/internal/integration/runtime"
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
	relay := outbox.NewRelay(relayRepo, integrationruntime.DefaultDecoders())

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
	if err := closeAuctionUC.Execute(context.Background(), tradingMetaAdmin(), tradingapp.AuctionID(auctionID)); err != nil {
		t.Fatalf("close auction error: %v", err)
	}

	if err := relay.RunOnce(context.Background(), bus, 100); err != nil {
		t.Fatalf("relay auction won error: %v", err)
	}

	dealItem, err := dealspg.NewDealRepository(db).GetActiveDealByAuctionID(context.Background(), auctionID)
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
	relay := outbox.NewRelay(relayRepo, integrationruntime.DefaultDecoders())

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

	if err := closeAuctionUC.Execute(ctx, tradingMetaAdmin(), tradingapp.AuctionID(auctionID)); err != nil {
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

func TestWinnerRejectCapturesDepositAndPromotesNext_RealPG(t *testing.T) {
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

	relayRepo := outboxpg.NewRepository(db)
	relay := outbox.NewRelay(relayRepo, integrationruntime.DefaultDecoders())

	startsAt := time.Now().Add(-time.Hour)
	endsAt := time.Now().Add(time.Hour)
	auctionID := "auc-decline-next"
	lotID := "lot-decline-next"

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
	ctx := context.Background()
	if err := catalogLotRepo.Save(ctx, lot); err != nil {
		t.Fatalf("save lot error: %v", err)
	}
	if err := catalogOutbox.Add(ctx, lotEvents); err != nil {
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
	handleDeclined, err := dealsapp.NewHandleDealDeclined(dealsUOW)
	if err != nil {
		t.Fatalf("handle declined: %v", err)
	}
	cancelDealUC, err := dealsapp.NewCancelDeal(dealsUOW)
	if err != nil {
		t.Fatalf("cancel deal: %v", err)
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
	captureUC, err := billingapp.NewCaptureAuctionDeposit(bAccounts, bDeposits, bLedger, billingapp.RandomHexID{}, nil, bEvents)
	if err != nil {
		t.Fatalf("capture uc: %v", err)
	}

	bus := inmemory.NewBus()

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

	bus.Subscribe("deals.DealCancelled", func(ctx context.Context, envelope events.Envelope) error {
		evt := envelope.Payload.(deal.DealCancelled)
		auctionMeta := ""
		if envelope.Meta != nil {
			auctionMeta = envelope.Meta["auction_id"]
		}
		return handleDeclined.Execute(ctx, dealsMeta(), auctionMeta, evt.DealID)
	})

	bus.Subscribe("deals.WinnerRejected", func(ctx context.Context, envelope events.Envelope) error {
		evt := envelope.Payload.(deal.WinnerRejected)
		return billingTx.WithinTx(ctx, func(txCtx context.Context) error {
			return captureUC.Execute(txCtx, evt.CompanyID, evt.AuctionID, evt.Reason)
		})
	})

	if err := relay.RunOnce(ctx, bus, 100); err != nil {
		t.Fatalf("relay lot published error: %v", err)
	}

	buyers := []string{"decl-a", "decl-b", "decl-c"}
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

	amounts := []int64{150, 160, 170}
	for i, b := range buyers {
		meta := tradingMetaWithCompany(b)
		tBid := endsAt.Add(time.Duration(-40+10*i) * time.Minute)
		if err := placeBidUC.Execute(ctx, meta, tradingapp.AuctionID(auctionID), amounts[i], tBid); err != nil {
			t.Fatalf("place bid %s: %v", b, err)
		}
	}

	if err := closeAuctionUC.Execute(ctx, tradingMetaAdmin(), tradingapp.AuctionID(auctionID)); err != nil {
		t.Fatalf("close auction error: %v", err)
	}

	if err := relay.RunOnce(ctx, bus, 100); err != nil {
		t.Fatalf("relay auction won error: %v", err)
	}

	dealRepo := dealspg.NewDealRepository(db)
	selRepo := dealspg.NewSelectionRepository(db)
	sel, err := selRepo.GetByAuctionID(ctx, auctionID)
	if err != nil {
		t.Fatalf("winner selection: %v", err)
	}
	d1, err := dealRepo.GetByID(ctx, sel.DealID)
	if err != nil {
		t.Fatalf("first deal: %v", err)
	}
	if d1.CustomerID() != "decl-c" {
		t.Fatalf("expected first winner decl-c, got %s", d1.CustomerID())
	}

	cancelMeta := dealsapp.CommandMeta{
		CompanyID:     d1.CustomerID(),
		UserID:        "u-decline",
		CorrelationID: "c-decline",
		CausationID:   "c-decline",
	}
	if err := cancelDealUC.Execute(ctx, cancelMeta, d1.ID(), deal.DealCancelReasonWinnerRejected); err != nil {
		t.Fatalf("winner cancel: %v", err)
	}

	for i := 0; i < 12; i++ {
		if err := relay.RunOnce(ctx, bus, 100); err != nil {
			t.Fatalf("relay after cancel %d: %v", i, err)
		}
	}

	sel, err = selRepo.GetByAuctionID(ctx, auctionID)
	if err != nil {
		t.Fatalf("selection after fallback: %v", err)
	}
	d2, err := dealRepo.GetByID(ctx, sel.DealID)
	if err != nil {
		t.Fatalf("second deal: %v", err)
	}
	if d2.CustomerID() != "decl-b" {
		t.Fatalf("expected fallback deal for decl-b, got %s", d2.CustomerID())
	}
	if active, err := dealRepo.GetActiveDealByAuctionID(ctx, auctionID); err != nil {
		t.Fatalf("get active deal: %v", err)
	} else if active.ID() != d2.ID() {
		t.Fatalf("GetActiveDealByAuctionID should match selection.DealID, got %s vs %s", active.ID(), d2.ID())
	}

	var topStatus string
	if err := db.QueryRowContext(ctx, `
SELECT status FROM billing_auction_deposits WHERE auction_id = $1 AND company_id = 'decl-c'
`, auctionID).Scan(&topStatus); err != nil {
		t.Fatalf("deposit decl-c: %v", err)
	}
	if topStatus != "CAPTURED" {
		t.Fatalf("expected decl-c deposit CAPTURED, got %s", topStatus)
	}
	for _, b := range []string{"decl-b", "decl-a"} {
		var st string
		if err := db.QueryRowContext(ctx, `
SELECT status FROM billing_auction_deposits WHERE auction_id = $1 AND company_id = $2
`, auctionID, b).Scan(&st); err != nil {
			t.Fatalf("deposit %s: %v", b, err)
		}
		if st != "HELD" {
			t.Fatalf("expected %s HELD, got %s", b, st)
		}
	}

	var capCount int
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM billing_ledger_entries
WHERE company_id = 'decl-c' AND type = $1
`, string(wallet.LedgerBidDepositCaptured)).Scan(&capCount); err != nil {
		t.Fatalf("count capture ledger: %v", err)
	}
	if capCount != 1 {
		t.Fatalf("expected one capture ledger row, got %d", capCount)
	}

	if err := billingTx.WithinTx(ctx, func(txCtx context.Context) error {
		return captureUC.Execute(txCtx, "decl-c", auctionID, deal.DealCancelReasonWinnerRejected)
	}); err != nil {
		t.Fatalf("idempotent capture execute: %v", err)
	}
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM billing_ledger_entries
WHERE company_id = 'decl-c' AND type = $1
`, string(wallet.LedgerBidDepositCaptured)).Scan(&capCount); err != nil {
		t.Fatalf("count capture ledger after replay: %v", err)
	}
	if capCount != 1 {
		t.Fatalf("after idempotent capture expected 1 ledger row, got %d", capCount)
	}
}

func TestWinnerConfirmLeavesDepositsHeld_RealPG(t *testing.T) {
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

	relayRepo := outboxpg.NewRepository(db)
	relay := outbox.NewRelay(relayRepo, integrationruntime.DefaultDecoders())

	startsAt := time.Now().Add(-time.Hour)
	endsAt := time.Now().Add(time.Hour)
	auctionID := "auc-confirm-stage8"
	lotID := "lot-confirm-stage8"

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
	ctx := context.Background()
	if err := catalogLotRepo.Save(ctx, lot); err != nil {
		t.Fatalf("save lot error: %v", err)
	}
	if err := catalogOutbox.Add(ctx, lotEvents); err != nil {
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

	bus := inmemory.NewBus()

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

	if err := relay.RunOnce(ctx, bus, 100); err != nil {
		t.Fatalf("relay lot published error: %v", err)
	}

	buyers := []string{"conf-a", "conf-b", "conf-c"}
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

	amounts := []int64{150, 160, 170}
	for i, b := range buyers {
		meta := tradingMetaWithCompany(b)
		tBid := endsAt.Add(time.Duration(-40+10*i) * time.Minute)
		if err := placeBidUC.Execute(ctx, meta, tradingapp.AuctionID(auctionID), amounts[i], tBid); err != nil {
			t.Fatalf("place bid %s: %v", b, err)
		}
	}

	if err := closeAuctionUC.Execute(ctx, tradingMetaAdmin(), tradingapp.AuctionID(auctionID)); err != nil {
		t.Fatalf("close auction error: %v", err)
	}

	if err := relay.RunOnce(ctx, bus, 100); err != nil {
		t.Fatalf("relay auction won error: %v", err)
	}

	dealRepo := dealspg.NewDealRepository(db)
	selRepo := dealspg.NewSelectionRepository(db)
	sel, err := selRepo.GetByAuctionID(ctx, auctionID)
	if err != nil {
		t.Fatalf("winner selection: %v", err)
	}
	d1, err := dealRepo.GetByID(ctx, sel.DealID)
	if err != nil {
		t.Fatalf("deal: %v", err)
	}
	if d1.CustomerID() != "conf-c" {
		t.Fatalf("expected winner conf-c, got %s", d1.CustomerID())
	}

	confirmUC, err := dealsapp.NewConfirmDeal(dealsUOW)
	if err != nil {
		t.Fatalf("confirm uc: %v", err)
	}
	confirmMeta := dealsapp.CommandMeta{
		CompanyID:     d1.CustomerID(),
		UserID:        "u-confirm",
		CorrelationID: "c-confirm",
		CausationID:   "c-confirm",
	}
	if err := confirmUC.Execute(ctx, confirmMeta, d1.ID()); err != nil {
		t.Fatalf("confirm deal: %v", err)
	}

	for i := 0; i < 8; i++ {
		if err := relay.RunOnce(ctx, bus, 100); err != nil {
			t.Fatalf("relay after confirm %d: %v", i, err)
		}
	}

	var selStatus string
	if err := db.QueryRowContext(ctx, `
SELECT status FROM deal_winner_selections WHERE auction_id = $1
`, auctionID).Scan(&selStatus); err != nil {
		t.Fatalf("selection status: %v", err)
	}
	if selStatus != "confirmed_pending_payment" {
		t.Fatalf("expected selection confirmed_pending_payment, got %s", selStatus)
	}

	for _, b := range buyers {
		var st string
		if err := db.QueryRowContext(ctx, `
SELECT status FROM billing_auction_deposits WHERE auction_id = $1 AND company_id = $2
`, auctionID, b).Scan(&st); err != nil {
			t.Fatalf("deposit %s: %v", b, err)
		}
		if st != "HELD" {
			t.Fatalf("expected %s HELD after winner confirm, got %s", b, st)
		}
	}

	var feeCount int
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM billing_ledger_entries WHERE type = $1
`, string(wallet.LedgerPlatformFeeCaptured)).Scan(&feeCount); err != nil {
		t.Fatalf("platform fee count: %v", err)
	}
	if feeCount != 0 {
		t.Fatalf("expected no PLATFORM_FEE_CAPTURED after stage-8 confirm, got %d", feeCount)
	}

	var releaseCount int
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM billing_ledger_entries WHERE type = $1
`, string(wallet.LedgerBidDepositReleased)).Scan(&releaseCount); err != nil {
		t.Fatalf("release count: %v", err)
	}
	if releaseCount != 0 {
		t.Fatalf("expected no BID_DEPOSIT_RELEASED (all top-3 still candidates), got %d", releaseCount)
	}
}

// TestStage9DealInvoiceFlow_RealPG: RequestPayment → outbox → billing invoice (fee net of HELD deposit), fake-confirm → PAID; deposits stay HELD.
func TestStage9DealInvoiceFlow_RealPG(t *testing.T) {
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

	relayRepo := outboxpg.NewRepository(db)
	relay := outbox.NewRelay(relayRepo, integrationruntime.DefaultDecoders())

	startsAt := time.Now().Add(-time.Hour)
	endsAt := time.Now().Add(time.Hour)
	auctionID := "auc-stage9-inv"
	lotID := "lot-stage9-inv"

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
	ctx := context.Background()
	if err := catalogLotRepo.Save(ctx, lot); err != nil {
		t.Fatalf("save lot error: %v", err)
	}
	if err := catalogOutbox.Add(ctx, lotEvents); err != nil {
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
	dealInvRepo := billingpg.NewDealInvoiceRepository(db)
	releaseExcept, err := billingapp.NewReleaseAuctionDepositsExceptCandidates(
		bAccounts, bDeposits, bLedger, billingapp.RandomHexID{}, nil, bEvents,
	)
	if err != nil {
		t.Fatalf("release except uc: %v", err)
	}
	createDealInvUC, err := billingapp.NewCreateDealInvoice(
		dealInvRepo,
		bDeposits,
		fake.Provider{},
		fake.ProviderName,
		billingapp.RandomHexID{},
		nil,
		bEvents,
		"http://localhost:8085",
	)
	if err != nil {
		t.Fatalf("create deal invoice uc: %v", err)
	}
	confirmDealInvUC, err := billingapp.NewConfirmDealInvoicePaid(dealInvRepo, bEvents, nil)
	if err != nil {
		t.Fatalf("confirm deal invoice uc: %v", err)
	}

	bus := inmemory.NewBus()

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

	bus.Subscribe("deals.PaymentRequested", func(ctx context.Context, envelope events.Envelope) error {
		evt := envelope.Payload.(deal.PaymentRequested)
		cur := wallet.Currency(evt.Currency)
		if evt.Currency == "" {
			cur = wallet.CurrencyRUB
		}
		var due time.Time
		if evt.DueDate != nil {
			due = *evt.DueDate
		}
		return billingTx.WithinTx(ctx, func(txCtx context.Context) error {
			_, err := createDealInvUC.Execute(txCtx, billingapp.CreateDealInvoiceCommand{
				DealID:          evt.DealID,
				AuctionID:       evt.AuctionID,
				BuyerCompanyID:  evt.BuyerCompanyID,
				SellerCompanyID: evt.SellerCompanyID,
				GoodsAmount:     evt.GoodsAmount,
				Currency:        cur,
				DueAt:           due,
			})
			return err
		})
	})

	if err := relay.RunOnce(ctx, bus, 100); err != nil {
		t.Fatalf("relay lot published error: %v", err)
	}

	buyers := []string{"st9-a", "st9-b", "st9-c"}
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

	amounts := []int64{150, 160, 170}
	for i, b := range buyers {
		meta := tradingMetaWithCompany(b)
		tBid := endsAt.Add(time.Duration(-40+10*i) * time.Minute)
		if err := placeBidUC.Execute(ctx, meta, tradingapp.AuctionID(auctionID), amounts[i], tBid); err != nil {
			t.Fatalf("place bid %s: %v", b, err)
		}
	}

	if err := closeAuctionUC.Execute(ctx, tradingMetaAdmin(), tradingapp.AuctionID(auctionID)); err != nil {
		t.Fatalf("close auction error: %v", err)
	}

	if err := relay.RunOnce(ctx, bus, 100); err != nil {
		t.Fatalf("relay auction won error: %v", err)
	}

	selRepo := dealspg.NewSelectionRepository(db)
	sel, err := selRepo.GetByAuctionID(ctx, auctionID)
	if err != nil {
		t.Fatalf("winner selection: %v", err)
	}
	dealRepo := dealspg.NewDealRepository(db)
	d1, err := dealRepo.GetByID(ctx, sel.DealID)
	if err != nil {
		t.Fatalf("deal: %v", err)
	}
	if d1.CustomerID() != "st9-c" {
		t.Fatalf("expected winner st9-c, got %s", d1.CustomerID())
	}

	confirmUC, err := dealsapp.NewConfirmDeal(dealsUOW)
	if err != nil {
		t.Fatalf("confirm uc: %v", err)
	}
	confirmMeta := dealsapp.CommandMeta{
		CompanyID:     d1.CustomerID(),
		UserID:        "u-confirm",
		CorrelationID: "c-stage9",
		CausationID:   "c-stage9",
	}
	if err := confirmUC.Execute(ctx, confirmMeta, d1.ID()); err != nil {
		t.Fatalf("confirm deal: %v", err)
	}

	prepareUC, err := dealsapp.NewPrepareContract(dealsUOW)
	if err != nil {
		t.Fatalf("prepare uc: %v", err)
	}
	if err := prepareUC.Execute(ctx, confirmMeta, d1.ID(), "CNT-ST9", "https://contracts/st9.pdf"); err != nil {
		t.Fatalf("prepare contract: %v", err)
	}
	signUC, err := dealsapp.NewSignContract(dealsUOW)
	if err != nil {
		t.Fatalf("sign uc: %v", err)
	}
	if err := signUC.Execute(ctx, confirmMeta, d1.ID(), "sig-st9"); err != nil {
		t.Fatalf("sign contract: %v", err)
	}
	reqPayUC, err := dealsapp.NewRequestPayment(dealsUOW)
	if err != nil {
		t.Fatalf("request payment uc: %v", err)
	}
	if err := reqPayUC.Execute(ctx, confirmMeta, d1.ID(), "", nil); err != nil {
		t.Fatalf("request payment: %v", err)
	}

	for i := 0; i < 20; i++ {
		if err := relay.RunOnce(ctx, bus, 100); err != nil {
			t.Fatalf("relay payment requested %d: %v", i, err)
		}
	}

	var invID, status string
	var goods, feeDue, total int64
	err = db.QueryRowContext(ctx, `
SELECT id, status, goods_amount, platform_fee_due_amount, total_amount
FROM billing_deal_invoices WHERE deal_id = $1
`, d1.ID()).Scan(&invID, &status, &goods, &feeDue, &total)
	if err != nil {
		t.Fatalf("load invoice: %v", err)
	}
	if status != "PAYMENT_PENDING" {
		t.Fatalf("expected PAYMENT_PENDING, got %s", status)
	}
	if goods != 170 {
		t.Fatalf("goods_amount: want 170 got %d", goods)
	}
	var held int64
	if err := db.QueryRowContext(ctx, `
SELECT amount FROM billing_auction_deposits WHERE auction_id = $1 AND company_id = $2
`, auctionID, "st9-c").Scan(&held); err != nil {
		t.Fatalf("winner deposit: %v", err)
	}
	feeFull := int64(170 * 3 / 100)
	wantFee := feeFull - held
	if wantFee < 0 {
		wantFee = 0
	}
	if feeDue != wantFee {
		t.Fatalf("platform_fee_due_amount: want %d got %d (held=%d, fee_full=%d)", wantFee, feeDue, held, feeFull)
	}
	if total != goods+feeDue {
		t.Fatalf("total: want %d got %d", goods+feeDue, total)
	}

	var payURL string
	if err := db.QueryRowContext(ctx, `SELECT payment_url FROM billing_deal_invoices WHERE id = $1`, invID).Scan(&payURL); err != nil {
		t.Fatalf("payment_url: %v", err)
	}
	if payURL == "" || !strings.Contains(payURL, "/billing/invoices/") || !strings.Contains(payURL, "/fake-confirm") {
		t.Fatalf("unexpected payment_url (want .../invoices/{id}/fake-confirm): %q", payURL)
	}

	if err := billingTx.WithinTx(ctx, func(txCtx context.Context) error {
		return confirmDealInvUC.Execute(txCtx, invID)
	}); err != nil {
		t.Fatalf("confirm invoice: %v", err)
	}

	if err := db.QueryRowContext(ctx, `SELECT status FROM billing_deal_invoices WHERE id = $1`, invID).Scan(&status); err != nil {
		t.Fatalf("reload invoice: %v", err)
	}
	if status != "PAID" {
		t.Fatalf("expected PAID, got %s", status)
	}

	for _, b := range buyers {
		var st string
		if err := db.QueryRowContext(ctx, `
SELECT status FROM billing_auction_deposits WHERE auction_id = $1 AND company_id = $2
`, auctionID, b).Scan(&st); err != nil {
			t.Fatalf("deposit %s: %v", b, err)
		}
		if st != "HELD" {
			t.Fatalf("expected %s HELD after invoice paid, got %s", b, st)
		}
	}
}

// TestStage11InvoicePaymentTimeout_RealPG: unpaid invoice past due → EXPIRED, rank1 deposit captured, next deal for rank2; second capture idempotent.
func TestStage11InvoicePaymentTimeout_RealPG(t *testing.T) {
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

	relayRepo := outboxpg.NewRepository(db)
	relay := outbox.NewRelay(relayRepo, integrationruntime.DefaultDecoders())

	startsAt := time.Now().Add(-time.Hour)
	endsAt := time.Now().Add(time.Hour)
	auctionID := "auc-stage11-payto"
	lotID := "lot-stage11-payto"

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
	ctx := context.Background()
	if err := catalogLotRepo.Save(ctx, lot); err != nil {
		t.Fatalf("save lot error: %v", err)
	}
	if err := catalogOutbox.Add(ctx, lotEvents); err != nil {
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
	dealInvRepo := billingpg.NewDealInvoiceRepository(db)
	releaseExcept, err := billingapp.NewReleaseAuctionDepositsExceptCandidates(
		bAccounts, bDeposits, bLedger, billingapp.RandomHexID{}, nil, bEvents,
	)
	if err != nil {
		t.Fatalf("release except uc: %v", err)
	}
	createDealInvUC, err := billingapp.NewCreateDealInvoice(
		dealInvRepo,
		bDeposits,
		fake.Provider{},
		fake.ProviderName,
		billingapp.RandomHexID{},
		nil,
		bEvents,
		"http://localhost:8085",
	)
	if err != nil {
		t.Fatalf("create deal invoice uc: %v", err)
	}
	expireDealInvUC, err := billingapp.NewExpireDealInvoice(dealInvRepo, bEvents, nil)
	if err != nil {
		t.Fatalf("expire deal invoice uc: %v", err)
	}
	invLister := billingpg.NewDealInvoiceLister(db)
	captureUC, err := billingapp.NewCaptureAuctionDeposit(bAccounts, bDeposits, bLedger, billingapp.RandomHexID{}, nil, bEvents)
	if err != nil {
		t.Fatalf("capture uc: %v", err)
	}
	handleDeclined, err := dealsapp.NewHandleDealDeclined(dealsUOW)
	if err != nil {
		t.Fatalf("handle declined: %v", err)
	}
	handleInvoiceExpired, err := dealsapp.NewHandleDealInvoiceExpired(dealsUOW, nil)
	if err != nil {
		t.Fatalf("handle invoice expired: %v", err)
	}

	bus := inmemory.NewBus()

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

	bus.Subscribe("deals.PaymentRequested", func(ctx context.Context, envelope events.Envelope) error {
		evt := envelope.Payload.(deal.PaymentRequested)
		cur := wallet.Currency(evt.Currency)
		if evt.Currency == "" {
			cur = wallet.CurrencyRUB
		}
		var due time.Time
		if evt.DueDate != nil {
			due = *evt.DueDate
		}
		return billingTx.WithinTx(ctx, func(txCtx context.Context) error {
			_, err := createDealInvUC.Execute(txCtx, billingapp.CreateDealInvoiceCommand{
				DealID:          evt.DealID,
				AuctionID:       evt.AuctionID,
				BuyerCompanyID:  evt.BuyerCompanyID,
				SellerCompanyID: evt.SellerCompanyID,
				GoodsAmount:     evt.GoodsAmount,
				Currency:        cur,
				DueAt:           due,
			})
			return err
		})
	})

	bus.Subscribe("deals.DealCancelled", func(ctx context.Context, envelope events.Envelope) error {
		evt := envelope.Payload.(deal.DealCancelled)
		if evt.Reason == deal.DealCancelReasonPaymentTimeout {
			return nil
		}
		auctionMeta := ""
		if envelope.Meta != nil {
			auctionMeta = envelope.Meta["auction_id"]
		}
		return handleDeclined.Execute(ctx, dealsMeta(), auctionMeta, evt.DealID)
	})

	bus.Subscribe("deals.WinnerRejected", func(ctx context.Context, envelope events.Envelope) error {
		evt := envelope.Payload.(deal.WinnerRejected)
		return billingTx.WithinTx(ctx, func(txCtx context.Context) error {
			return captureUC.Execute(txCtx, evt.CompanyID, evt.AuctionID, evt.Reason)
		})
	})

	bus.Subscribe("billing.DealInvoiceExpired", func(ctx context.Context, envelope events.Envelope) error {
		evt := envelope.Payload.(wallet.DealInvoiceExpired)
		return handleInvoiceExpired.Execute(ctx, evt)
	})

	if err := relay.RunOnce(ctx, bus, 100); err != nil {
		t.Fatalf("relay lot published error: %v", err)
	}

	buyers := []string{"st11-a", "st11-b", "st11-c"}
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

	amounts := []int64{150, 160, 170}
	for i, b := range buyers {
		meta := tradingMetaWithCompany(b)
		tBid := endsAt.Add(time.Duration(-40+10*i) * time.Minute)
		if err := placeBidUC.Execute(ctx, meta, tradingapp.AuctionID(auctionID), amounts[i], tBid); err != nil {
			t.Fatalf("place bid %s: %v", b, err)
		}
	}

	if err := closeAuctionUC.Execute(ctx, tradingMetaAdmin(), tradingapp.AuctionID(auctionID)); err != nil {
		t.Fatalf("close auction error: %v", err)
	}

	if err := relay.RunOnce(ctx, bus, 100); err != nil {
		t.Fatalf("relay auction won error: %v", err)
	}

	selRepo := dealspg.NewSelectionRepository(db)
	sel, err := selRepo.GetByAuctionID(ctx, auctionID)
	if err != nil {
		t.Fatalf("winner selection: %v", err)
	}
	dealRepo := dealspg.NewDealRepository(db)
	d1, err := dealRepo.GetByID(ctx, sel.DealID)
	if err != nil {
		t.Fatalf("first deal: %v", err)
	}
	if d1.CustomerID() != "st11-c" {
		t.Fatalf("expected first winner st11-c, got %s", d1.CustomerID())
	}

	confirmUC, err := dealsapp.NewConfirmDeal(dealsUOW)
	if err != nil {
		t.Fatalf("confirm uc: %v", err)
	}
	confirmMeta := dealsapp.CommandMeta{
		CompanyID:     d1.CustomerID(),
		UserID:        "u-st11",
		CorrelationID: "c-st11",
		CausationID:   "c-st11",
	}
	if err := confirmUC.Execute(ctx, confirmMeta, d1.ID()); err != nil {
		t.Fatalf("confirm deal: %v", err)
	}
	prepareUC, err := dealsapp.NewPrepareContract(dealsUOW)
	if err != nil {
		t.Fatalf("prepare uc: %v", err)
	}
	if err := prepareUC.Execute(ctx, confirmMeta, d1.ID(), "CNT-ST11", "https://contracts/st11.pdf"); err != nil {
		t.Fatalf("prepare contract: %v", err)
	}
	signUC, err := dealsapp.NewSignContract(dealsUOW)
	if err != nil {
		t.Fatalf("sign uc: %v", err)
	}
	if err := signUC.Execute(ctx, confirmMeta, d1.ID(), "sig-st11"); err != nil {
		t.Fatalf("sign contract: %v", err)
	}
	reqPayUC, err := dealsapp.NewRequestPayment(dealsUOW)
	if err != nil {
		t.Fatalf("request payment uc: %v", err)
	}
	if err := reqPayUC.Execute(ctx, confirmMeta, d1.ID(), "", nil); err != nil {
		t.Fatalf("request payment: %v", err)
	}

	for i := 0; i < 20; i++ {
		if err := relay.RunOnce(ctx, bus, 100); err != nil {
			t.Fatalf("relay payment requested %d: %v", i, err)
		}
	}

	var invID, invStatus string
	if err := db.QueryRowContext(ctx, `
SELECT id, status FROM billing_deal_invoices WHERE deal_id = $1
`, d1.ID()).Scan(&invID, &invStatus); err != nil {
		t.Fatalf("load invoice: %v", err)
	}
	if invStatus != "PAYMENT_PENDING" {
		t.Fatalf("expected PAYMENT_PENDING, got %s", invStatus)
	}

	pastDue := time.Now().Add(-2 * time.Hour).UTC()
	if _, err := db.ExecContext(ctx, `UPDATE billing_deal_invoices SET due_at = $1 WHERE id = $2`, pastDue, invID); err != nil {
		t.Fatalf("backdate due_at: %v", err)
	}

	if err := billingTx.WithinTx(ctx, func(txCtx context.Context) error {
		ids, err := invLister.ListExpired(txCtx, time.Now().UTC(), 100)
		if err != nil {
			return err
		}
		for _, id := range ids {
			if err := expireDealInvUC.Execute(txCtx, id); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("expire invoices tx: %v", err)
	}

	for i := 0; i < 25; i++ {
		if err := relay.RunOnce(ctx, bus, 100); err != nil {
			t.Fatalf("relay after expire %d: %v", i, err)
		}
	}

	if err := db.QueryRowContext(ctx, `SELECT status FROM billing_deal_invoices WHERE id = $1`, invID).Scan(&invStatus); err != nil {
		t.Fatalf("invoice status: %v", err)
	}
	if invStatus != "EXPIRED" {
		t.Fatalf("expected EXPIRED invoice, got %s", invStatus)
	}

	var payoutForFirstDeal int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM billing_seller_payouts WHERE deal_id = $1`, d1.ID()).Scan(&payoutForFirstDeal); err != nil {
		t.Fatalf("seller payout for timed-out deal: %v", err)
	}
	if payoutForFirstDeal != 0 {
		t.Fatalf("expected no billing_seller_payouts for unpaid/cancelled first deal, got %d", payoutForFirstDeal)
	}

	sel, err = selRepo.GetByAuctionID(ctx, auctionID)
	if err != nil {
		t.Fatalf("selection after timeout: %v", err)
	}
	d2, err := dealRepo.GetByID(ctx, sel.DealID)
	if err != nil {
		t.Fatalf("second deal: %v", err)
	}
	if d2.CustomerID() != "st11-b" {
		t.Fatalf("expected fallback deal for st11-b, got %s", d2.CustomerID())
	}
	if d2.Status() == deal.DealStatusCancelled {
		t.Fatal("new deal should not be cancelled")
	}

	var topStatus string
	if err := db.QueryRowContext(ctx, `
SELECT status FROM billing_auction_deposits WHERE auction_id = $1 AND company_id = 'st11-c'
`, auctionID).Scan(&topStatus); err != nil {
		t.Fatalf("deposit st11-c: %v", err)
	}
	if topStatus != "CAPTURED" {
		t.Fatalf("expected st11-c deposit CAPTURED, got %s", topStatus)
	}
	for _, b := range []string{"st11-b", "st11-a"} {
		var st string
		if err := db.QueryRowContext(ctx, `
SELECT status FROM billing_auction_deposits WHERE auction_id = $1 AND company_id = $2
`, auctionID, b).Scan(&st); err != nil {
			t.Fatalf("deposit %s: %v", b, err)
		}
		if st != "HELD" {
			t.Fatalf("expected %s HELD, got %s", b, st)
		}
	}

	var capCount int
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM billing_ledger_entries
WHERE company_id = 'st11-c' AND type = $1
`, string(wallet.LedgerBidDepositCaptured)).Scan(&capCount); err != nil {
		t.Fatalf("count capture ledger: %v", err)
	}
	if capCount != 1 {
		t.Fatalf("expected one capture ledger row, got %d", capCount)
	}

	if err := billingTx.WithinTx(ctx, func(txCtx context.Context) error {
		return captureUC.Execute(txCtx, "st11-c", auctionID, deal.DealCancelReasonPaymentTimeout)
	}); err != nil {
		t.Fatalf("idempotent capture: %v", err)
	}
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM billing_ledger_entries
WHERE company_id = 'st11-c' AND type = $1
`, string(wallet.LedgerBidDepositCaptured)).Scan(&capCount); err != nil {
		t.Fatalf("count capture after replay: %v", err)
	}
	if capCount != 1 {
		t.Fatalf("after idempotent capture expected 1 ledger row, got %d", capCount)
	}
}

// TestStage10FinalizePaymentAfterInvoicePaid_RealPG: after Stage-9 invoice confirm, relay processes
// DealInvoicePaid → deal paid + WinnerSelectionFinalized; then settlement releases loser deposits and captures fee from winner HELD deposit; second relay is stable.
func TestStage10FinalizePaymentAfterInvoicePaid_RealPG(t *testing.T) {
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

	relayRepo := outboxpg.NewRepository(db)
	relay := outbox.NewRelay(relayRepo, integrationruntime.DefaultDecoders())

	// Unique IDs: CreateAuction is a no-op if the row already exists (fixed ID + stale row → wrong schedule / "already ended").
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	auctionID := "auc-stage10-" + suffix
	lotID := "lot-stage10-" + suffix
	sched := catalog.NewAuctionScheduleAt(time.Now().Add(-time.Hour), time.Hour)
	endsAt := sched.EndsAt()

	lot, _, err := catalog.NewLot(
		lotID,
		"prod-1",
		"seller-1",
		"photo",
		10,
		100,
		10,
		sched,
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
	ctx := context.Background()
	if err := catalogLotRepo.Save(ctx, lot); err != nil {
		t.Fatalf("save lot error: %v", err)
	}
	if err := catalogOutbox.Add(ctx, lotEvents); err != nil {
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
	dealInvRepo := billingpg.NewDealInvoiceRepository(db)
	releaseExcept, err := billingapp.NewReleaseAuctionDepositsExceptCandidates(
		bAccounts, bDeposits, bLedger, billingapp.RandomHexID{}, nil, bEvents,
	)
	if err != nil {
		t.Fatalf("release except uc: %v", err)
	}
	createDealInvUC, err := billingapp.NewCreateDealInvoice(
		dealInvRepo,
		bDeposits,
		fake.Provider{},
		fake.ProviderName,
		billingapp.RandomHexID{},
		nil,
		bEvents,
		"http://localhost:8085",
	)
	if err != nil {
		t.Fatalf("create deal invoice uc: %v", err)
	}
	confirmDealInvUC, err := billingapp.NewConfirmDealInvoicePaid(dealInvRepo, bEvents, nil)
	if err != nil {
		t.Fatalf("confirm deal invoice uc: %v", err)
	}
	settleWinnerUC, err := billingapp.NewSettleWinnerDepositAfterInvoicePaid(
		bAccounts, bDeposits, bLedger, billingapp.RandomHexID{}, nil, bEvents,
	)
	if err != nil {
		t.Fatalf("settle winner uc: %v", err)
	}
	sellerPayoutRepo := billingpg.NewSellerPayoutRepository(db)
	createSellerPayoutUC, err := billingapp.NewCreateSellerPayout(
		sellerPayoutRepo, dealInvRepo, billingapp.RandomHexID{}, nil, bEvents,
	)
	if err != nil {
		t.Fatalf("create seller payout uc: %v", err)
	}
	handleDealInvPaidUC, err := dealsapp.NewHandleDealInvoicePaid(dealsUOW)
	if err != nil {
		t.Fatalf("handle deal invoice paid: %v", err)
	}

	bus := inmemory.NewBus()

	bus.Subscribe("catalog.LotPublished", func(ctx context.Context, envelope events.Envelope) error {
		evt := envelope.Payload.(catalog.LotPublished)
		aStarts, aEnds := evt.AuctionStartsAt, evt.AuctionEndsAt
		if aStarts.IsZero() || aEnds.IsZero() {
			return errors.New("missing auction schedule in LotPublished")
		}
		minStep := evt.MinBidStep
		if minStep <= 0 {
			minStep = 1
		}
		if _, err := createAuctionUC.Execute(ctx, tradingMeta(), evt.LotID, aStarts, aEnds, evt.StartPrice, minStep); err != nil {
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

	bus.Subscribe("deals.PaymentRequested", func(ctx context.Context, envelope events.Envelope) error {
		evt := envelope.Payload.(deal.PaymentRequested)
		cur := wallet.Currency(evt.Currency)
		if evt.Currency == "" {
			cur = wallet.CurrencyRUB
		}
		var due time.Time
		if evt.DueDate != nil {
			due = *evt.DueDate
		}
		return billingTx.WithinTx(ctx, func(txCtx context.Context) error {
			_, err := createDealInvUC.Execute(txCtx, billingapp.CreateDealInvoiceCommand{
				DealID:          evt.DealID,
				AuctionID:       evt.AuctionID,
				BuyerCompanyID:  evt.BuyerCompanyID,
				SellerCompanyID: evt.SellerCompanyID,
				GoodsAmount:     evt.GoodsAmount,
				Currency:        cur,
				DueAt:           due,
			})
			return err
		})
	})

	bus.Subscribe("billing.DealInvoicePaid", func(ctx context.Context, envelope events.Envelope) error {
		evt := envelope.Payload.(wallet.DealInvoicePaid)
		return handleDealInvPaidUC.Execute(ctx, evt)
	})

	bus.Subscribe("deals.WinnerSelectionFinalized", func(ctx context.Context, envelope events.Envelope) error {
		evt := envelope.Payload.(deal.WinnerSelectionFinalized)
		return billingTx.WithinTx(ctx, func(txCtx context.Context) error {
			inv, err := dealInvRepo.LoadByDealIDForUpdate(txCtx, evt.DealID)
			if err != nil {
				return err
			}
			if inv.Status != wallet.InvoicePaid {
				return wallet.ErrInvoiceNotPayable
			}
			if inv.AuctionID != evt.AuctionID || inv.BuyerCompanyID != evt.CompanyID {
				return billingapp.ErrSellerPayoutInvoiceMismatch
			}
			if inv.GoodsAmount != evt.GoodsAmount || inv.PlatformFeeDueAmount != evt.PlatformFeeDueAmount {
				return billingapp.ErrSellerPayoutInvoiceMismatch
			}
			if err := settleWinnerUC.Execute(txCtx, evt.AuctionID, evt.CompanyID, evt.GoodsAmount, evt.PlatformFeeDueAmount, "WINNER_FINALIZED"); err != nil {
				return err
			}
			if err := releaseExcept.Execute(txCtx, evt.AuctionID, []string{evt.CompanyID}, "WINNER_FINALIZED"); err != nil {
				return err
			}
			_, err = createSellerPayoutUC.Execute(txCtx, billingapp.CreateSellerPayoutCommand{
				DealID:               evt.DealID,
				AuctionID:            evt.AuctionID,
				BuyerCompanyID:       evt.CompanyID,
				GoodsAmount:          evt.GoodsAmount,
				PlatformFeeDueAmount: evt.PlatformFeeDueAmount,
			})
			return err
		})
	})

	if err := relay.RunOnce(ctx, bus, 100); err != nil {
		t.Fatalf("relay lot published error: %v", err)
	}

	buyers := []string{"st10-a", "st10-b", "st10-c"}
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

	amounts := []int64{150, 160, 170}
	for i, b := range buyers {
		meta := tradingMetaWithCompany(b)
		tBid := endsAt.Add(time.Duration(-40+10*i) * time.Minute)
		if err := placeBidUC.Execute(ctx, meta, tradingapp.AuctionID(auctionID), amounts[i], tBid); err != nil {
			t.Fatalf("place bid %s: %v", b, err)
		}
	}

	if err := closeAuctionUC.Execute(ctx, tradingMetaAdmin(), tradingapp.AuctionID(auctionID)); err != nil {
		t.Fatalf("close auction error: %v", err)
	}

	if err := relay.RunOnce(ctx, bus, 100); err != nil {
		t.Fatalf("relay auction won error: %v", err)
	}

	selRepo := dealspg.NewSelectionRepository(db)
	sel, err := selRepo.GetByAuctionID(ctx, auctionID)
	if err != nil {
		t.Fatalf("winner selection: %v", err)
	}
	dealRepo := dealspg.NewDealRepository(db)
	d1, err := dealRepo.GetByID(ctx, sel.DealID)
	if err != nil {
		t.Fatalf("deal: %v", err)
	}
	if d1.CustomerID() != "st10-c" {
		t.Fatalf("expected winner st10-c, got %s", d1.CustomerID())
	}

	confirmUC, err := dealsapp.NewConfirmDeal(dealsUOW)
	if err != nil {
		t.Fatalf("confirm uc: %v", err)
	}
	confirmMeta := dealsapp.CommandMeta{
		CompanyID:     d1.CustomerID(),
		UserID:        "u-confirm",
		CorrelationID: "c-stage10",
		CausationID:   "c-stage10",
	}
	if err := confirmUC.Execute(ctx, confirmMeta, d1.ID()); err != nil {
		t.Fatalf("confirm deal: %v", err)
	}

	prepareUC, err := dealsapp.NewPrepareContract(dealsUOW)
	if err != nil {
		t.Fatalf("prepare uc: %v", err)
	}
	if err := prepareUC.Execute(ctx, confirmMeta, d1.ID(), "CNT-ST10", "https://contracts/st10.pdf"); err != nil {
		t.Fatalf("prepare contract: %v", err)
	}
	signUC, err := dealsapp.NewSignContract(dealsUOW)
	if err != nil {
		t.Fatalf("sign uc: %v", err)
	}
	if err := signUC.Execute(ctx, confirmMeta, d1.ID(), "sig-st10"); err != nil {
		t.Fatalf("sign contract: %v", err)
	}
	reqPayUC, err := dealsapp.NewRequestPayment(dealsUOW)
	if err != nil {
		t.Fatalf("request payment uc: %v", err)
	}
	if err := reqPayUC.Execute(ctx, confirmMeta, d1.ID(), "", nil); err != nil {
		t.Fatalf("request payment: %v", err)
	}

	for i := 0; i < 20; i++ {
		if err := relay.RunOnce(ctx, bus, 100); err != nil {
			t.Fatalf("relay payment requested %d: %v", i, err)
		}
	}

	var invID string
	if err := db.QueryRowContext(ctx, `SELECT id FROM billing_deal_invoices WHERE deal_id = $1`, d1.ID()).Scan(&invID); err != nil {
		t.Fatalf("load invoice: %v", err)
	}

	if err := billingTx.WithinTx(ctx, func(txCtx context.Context) error {
		return confirmDealInvUC.Execute(txCtx, invID)
	}); err != nil {
		t.Fatalf("confirm invoice: %v", err)
	}

	for i := 0; i < 30; i++ {
		if err := relay.RunOnce(ctx, bus, 100); err != nil {
			t.Fatalf("relay stage10 finalize %d: %v", i, err)
		}
	}

	d1, err = dealRepo.GetByID(ctx, sel.DealID)
	if err != nil {
		t.Fatalf("reload deal: %v", err)
	}
	if d1.Status() != deal.DealStatusPaid {
		t.Fatalf("deal status: want paid got %s", d1.Status())
	}
	sel, err = selRepo.GetByAuctionID(ctx, auctionID)
	if err != nil {
		t.Fatalf("reload selection: %v", err)
	}
	if sel.Status != deal.WinnerSelectionStatusFinalized {
		t.Fatalf("selection: want finalized got %s", sel.Status)
	}

	var wStatus string
	if err := db.QueryRowContext(ctx, `
SELECT status FROM billing_auction_deposits WHERE auction_id = $1 AND company_id = 'st10-c'
`, auctionID).Scan(&wStatus); err != nil {
		t.Fatalf("winner deposit: %v", err)
	}
	if wStatus != "CAPTURED" {
		t.Fatalf("winner deposit: want CAPTURED got %s", wStatus)
	}
	for _, b := range []string{"st10-a", "st10-b"} {
		var st string
		if err := db.QueryRowContext(ctx, `
SELECT status FROM billing_auction_deposits WHERE auction_id = $1 AND company_id = $2
`, auctionID, b).Scan(&st); err != nil {
			t.Fatalf("deposit %s: %v", b, err)
		}
		if st != "RELEASED" {
			t.Fatalf("loser %s: want RELEASED got %s", b, st)
		}
	}

	var ledgerRows int
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM billing_ledger_entries
WHERE company_id = 'st10-c' AND type = $1
`, string(wallet.LedgerPlatformFeeCaptured)).Scan(&ledgerRows); err != nil {
		t.Fatalf("count fee ledger: %v", err)
	}
	if ledgerRows != 1 {
		t.Fatalf("winner PLATFORM_FEE_CAPTURED rows: want 1 got %d", ledgerRows)
	}

	var goodsAmt int64
	var payoutAmt int64
	var payoutStatus string
	var payoutCount int
	if err := db.QueryRowContext(ctx, `SELECT goods_amount FROM billing_deal_invoices WHERE deal_id = $1`, d1.ID()).Scan(&goodsAmt); err != nil {
		t.Fatalf("invoice goods_amount: %v", err)
	}
	if err := db.QueryRowContext(ctx, `
SELECT count(*), coalesce((SELECT amount FROM billing_seller_payouts WHERE deal_id = $1 LIMIT 1), 0),
       coalesce((SELECT status FROM billing_seller_payouts WHERE deal_id = $1 LIMIT 1), '')
`, d1.ID()).Scan(&payoutCount, &payoutAmt, &payoutStatus); err != nil {
		t.Fatalf("seller payout row: %v", err)
	}
	if payoutCount != 1 {
		t.Fatalf("billing_seller_payouts rows: want 1 got %d", payoutCount)
	}
	if payoutAmt != goodsAmt {
		t.Fatalf("payout amount: want %d (goods) got %d", goodsAmt, payoutAmt)
	}
	if payoutStatus != "PENDING" {
		t.Fatalf("payout status: want PENDING got %s", payoutStatus)
	}
	var outboxPayoutCreated int
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM outbox_messages
WHERE event_type = 'billing.SellerPayoutCreated' AND aggregate_id = $1
`, d1.ID()).Scan(&outboxPayoutCreated); err != nil {
		t.Fatalf("outbox SellerPayoutCreated: %v", err)
	}
	if outboxPayoutCreated != 1 {
		t.Fatalf("billing.SellerPayoutCreated outbox: want 1 got %d", outboxPayoutCreated)
	}

	var ledgerRowsAfterReplay int
	for i := 0; i < 15; i++ {
		if err := relay.RunOnce(ctx, bus, 100); err != nil {
			t.Fatalf("replay relay %d: %v", i, err)
		}
	}
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM billing_ledger_entries
WHERE company_id = 'st10-c' AND type = $1
`, string(wallet.LedgerPlatformFeeCaptured)).Scan(&ledgerRowsAfterReplay); err != nil {
		t.Fatalf("count fee ledger replay: %v", err)
	}
	if ledgerRowsAfterReplay != ledgerRows {
		t.Fatalf("fee ledger changed after replay: was %d now %d", ledgerRows, ledgerRowsAfterReplay)
	}
	var payoutCountReplay int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM billing_seller_payouts WHERE deal_id = $1`, d1.ID()).Scan(&payoutCountReplay); err != nil {
		t.Fatalf("payout count after replay: %v", err)
	}
	if payoutCountReplay != 1 {
		t.Fatalf("after relay replay: want 1 seller payout row got %d", payoutCountReplay)
	}
	var outboxPayoutReplay int
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM outbox_messages
WHERE event_type = 'billing.SellerPayoutCreated' AND aggregate_id = $1
`, d1.ID()).Scan(&outboxPayoutReplay); err != nil {
		t.Fatalf("outbox payout after replay: %v", err)
	}
	if outboxPayoutReplay != outboxPayoutCreated {
		t.Fatalf("SellerPayoutCreated outbox rows changed after replay: was %d now %d", outboxPayoutCreated, outboxPayoutReplay)
	}

	var payoutID string
	if err := db.QueryRowContext(ctx, `SELECT id FROM billing_seller_payouts WHERE deal_id = $1 LIMIT 1`, d1.ID()).Scan(&payoutID); err != nil {
		t.Fatalf("payout id: %v", err)
	}
	createAccUC, err := billingapp.NewCreateAccount(bAccounts, billingapp.RandomHexID{}, bEvents)
	if err != nil {
		t.Fatalf("create account uc: %v", err)
	}
	markReadyUC, err := billingapp.NewMarkSellerPayoutReady(sellerPayoutRepo, nil, bEvents)
	if err != nil {
		t.Fatalf("mark payout ready uc: %v", err)
	}
	markPaidUC, err := billingapp.NewMarkSellerPayoutPaid(
		sellerPayoutRepo, bAccounts, bLedger, createAccUC, billingapp.RandomHexID{}, nil, bEvents,
	)
	if err != nil {
		t.Fatalf("mark payout paid uc: %v", err)
	}
	if err := billingTx.WithinTx(ctx, func(txCtx context.Context) error {
		_, err := markReadyUC.Execute(txCtx, payoutID)
		return err
	}); err != nil {
		t.Fatalf("mark payout ready tx: %v", err)
	}
	if err := billingTx.WithinTx(ctx, func(txCtx context.Context) error {
		_, err := markPaidUC.Execute(txCtx, payoutID)
		return err
	}); err != nil {
		t.Fatalf("mark payout paid tx: %v", err)
	}
	if err := billingTx.WithinTx(ctx, func(txCtx context.Context) error {
		_, err := markPaidUC.Execute(txCtx, payoutID)
		return err
	}); err != nil {
		t.Fatalf("mark payout paid idempotent tx: %v", err)
	}
	var payoutSt string
	if err := db.QueryRowContext(ctx, `SELECT status FROM billing_seller_payouts WHERE id = $1`, payoutID).Scan(&payoutSt); err != nil {
		t.Fatalf("payout status reload: %v", err)
	}
	if payoutSt != "PAID" {
		t.Fatalf("payout status after paid: want PAID got %s", payoutSt)
	}
	var sellerAvail int64
	if err := db.QueryRowContext(ctx, `
SELECT a.available_amount FROM billing_accounts a
WHERE a.company_id = (SELECT seller_company_id FROM billing_seller_payouts WHERE id = $1)
`, payoutID).Scan(&sellerAvail); err != nil {
		t.Fatalf("seller available: %v", err)
	}
	if sellerAvail != goodsAmt {
		t.Fatalf("seller available after payout: want %d got %d", goodsAmt, sellerAvail)
	}
	var creditLedger int
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM billing_ledger_entries
WHERE company_id = (SELECT seller_company_id FROM billing_seller_payouts WHERE id = $1)
  AND type = $2 AND reference_type = 'seller_payout' AND reference_id = $1
`, payoutID, string(wallet.LedgerSellerPayoutCredited)).Scan(&creditLedger); err != nil {
		t.Fatalf("payout credit ledger count: %v", err)
	}
	if creditLedger != 1 {
		t.Fatalf("SELLER_PAYOUT_CREDITED rows: want 1 got %d", creditLedger)
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
	integrationtest.AcquireSharedPostgresAdvisoryLock(t, db)
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
    billing_seller_payouts,
    billing_deal_invoices,
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
