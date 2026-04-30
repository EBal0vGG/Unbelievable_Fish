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

	catalogapp "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
	catalog "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/domain"
	catalogpg "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/postgres"
	dealsapp "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	deal "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
	dealspg "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/postgres"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/eventbus/inmemory"
	outbox "github.com/EBal0vGG/Unbelievable_Fish/internal/infra/outbox"
	outboxpg "github.com/EBal0vGG/Unbelievable_Fish/internal/infra/outbox/postgres"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/shared/events"
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
		"catalog.LotPublished":     outbox.JSONDecoder[catalog.LotPublished](),
		"trading.AuctionPublished": outbox.JSONDecoder[auction.AuctionPublished](),
		"trading.BidPlaced":        outbox.JSONDecoder[auction.BidPlaced](),
		"trading.AuctionClosed":    outbox.JSONDecoder[auction.AuctionClosed](),
		"trading.AuctionWon":       outbox.JSONDecoder[auction.AuctionWon](),
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

	placeBidUC, err := tradingapp.NewPlaceBid(tradingUOW)
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

type fixedAuctionIDFactoryReal struct {
	auctionID tradingapp.AuctionID
}

func (f fixedAuctionIDFactoryReal) NewID() (tradingapp.AuctionID, error) {
	return f.auctionID, nil
}

func openRealPostgres(t *testing.T) (*sql.DB, bool) {
	t.Helper()

	host := os.Getenv("PGHOST")
	user := os.Getenv("PGUSER")
	password := os.Getenv("PGPASSWORD")
	database := os.Getenv("PGDATABASE")
	port := os.Getenv("PGPORT")
	sslmode := os.Getenv("PGSSLMODE")

	if host == "" || user == "" || database == "" {
		t.Skip("real pg not configured (PGHOST/PGUSER/PGDATABASE)")
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
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(5)
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
    trading_auction_winners,
    trading_bids,
    trading_auctions,
    catalog_lots,
    deal_winner_selections,
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
