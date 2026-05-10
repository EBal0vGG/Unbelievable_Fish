package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	billingpg "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/postgres"
	catalogapp "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
	catalog "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/domain"
	catalogpg "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/postgres"
	dealsapp "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	dealspg "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/postgres"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/dbconfig"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/logging"
	integration "github.com/EBal0vGG/Unbelievable_Fish/internal/integration/runtime"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/adapters/billingdeposit"
	tradingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	tradingpg "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	logger := logging.New("chain_runner")
	db, ok := openRealPostgres()
	if !ok {
		logging.Fatal(logger, "database_config_missing", "required", "PGHOST,PGUSER,PGDATABASE")
	}
	defer db.Close()

	if err := applyMigrations(db); err != nil {
		logging.Fatal(logger, "migrations_apply_failed", "error", err)
	}
	if err := truncateAll(db); err != nil {
		logging.Fatal(logger, "truncate_failed", "error", err)
	}

	if err := runChains(db); err != nil {
		logging.Fatal(logger, "chains_failed", "error", err)
	}
	logger.Info("chains_verified")
}

func runChains(db *sql.DB) error {
	lotRepo := catalogpg.NewLotRepository(db)
	outboxRepo := catalogpg.NewOutboxRepository(db)
	txManager := catalogpg.NewTransactionManager(db, nil)
	catalogService := catalogapp.NewCatalogService(
		newMemoryFishRepo(),
		newMemoryUnitRepo(),
		newMemoryProcessingTypeRepo(),
		newMemoryProductRepo(),
		lotRepo,
		outboxRepo,
		catalogapp.NewRandomIDGenerator(),
		txManager,
	)

	tradingUOW := tradingpg.NewUnitOfWork(db)
	dealsUOW := dealspg.NewUnitOfWork(db)
	dealProjectionRepo := dealspg.NewProjectionRepository(db)
	auctionLister := tradingpg.NewAuctionLister(db)

	startsAt := time.Now().Add(-time.Hour)
	endsAt := time.Now().Add(time.Hour)
	auctionID := "auc-chain"
	lotID := ""

	fishID, err := catalogService.CreateFish(context.Background(), catalogapp.CreateFishCommand{
		Name:        "Fish",
		Description: "Fish",
	})
	if err != nil {
		return fmt.Errorf("create fish: %w", err)
	}

	sellerCtx := catalogapp.WithActor(context.Background(), catalogapp.Actor{
		CompanyID:           "seller-1",
		Kind:                catalogapp.ActorKindCompany,
		SellerCatalogAccess: true,
	})
	productID, _, err := catalogService.CreateProduct(sellerCtx, catalogapp.CreateProductCommand{
		FishID:         fishID,
		Weight:         1.5,
		Unit:           "kg",
		Size:           "M",
		ProcessingType: catalog.ProcessingType("frozen"),
	})
	if err != nil {
		return fmt.Errorf("create product: %w", err)
	}

	if err := catalogService.PublishProduct(sellerCtx, productID); err != nil {
		return fmt.Errorf("publish product: %w", err)
	}

	ctxWithCompany := sellerCtx
	lotID, _, err = catalogService.CreateLot(ctxWithCompany, catalogapp.CreateLotCommand{
		ProductID:              productID,
		Photo:                  "photo",
		Quantity:               10,
		StartPrice:             100,
		MinBidStep:             10,
		AuctionStartsAt:        startsAt,
		AuctionDurationMinutes: int64(endsAt.Sub(startsAt).Minutes()),
	})
	if err != nil {
		return fmt.Errorf("create lot: %w", err)
	}

	if err := catalogService.AssignAuctionID(ctxWithCompany, lotID, auctionID); err != nil {
		return fmt.Errorf("assign auction: %w", err)
	}
	if err := catalogService.PublishLot(ctxWithCompany, lotID); err != nil {
		return fmt.Errorf("publish lot: %w", err)
	}

	bAccounts := billingpg.NewAccountRepository(db)
	bLedger := billingpg.NewLedgerRepository(db)
	bDeposits := billingpg.NewAuctionDepositRepository(db)
	bProcessed := billingpg.NewProcessedTopUpRepository(db)
	bEvents := billingpg.NewOutboxRepository(db)
	bTx := billingpg.NewTransactionManager(db, nil)
	bCreateAccount, err := billingapp.NewCreateAccount(bAccounts, billingapp.RandomHexID{}, bEvents)
	if err != nil {
		return err
	}
	bReleaseExcept, err := billingapp.NewReleaseAuctionDepositsExceptCandidates(
		bAccounts, bDeposits, bLedger, billingapp.RandomHexID{}, nil, bEvents,
	)
	if err != nil {
		return err
	}
	bConfirm, err := billingapp.NewConfirmTopUp(bAccounts, bLedger, bProcessed, billingapp.RandomHexID{}, nil, bEvents)
	if err != nil {
		return err
	}
	bReserve, err := billingapp.NewReserveAuctionDeposit(bAccounts, bDeposits, bLedger, billingapp.RandomHexID{}, nil, bEvents)
	if err != nil {
		return err
	}
	bCapture, err := billingapp.NewCaptureAuctionDeposit(bAccounts, bDeposits, bLedger, billingapp.RandomHexID{}, nil, bEvents)
	if err != nil {
		return err
	}

	runtime, err := integration.New(db, integration.Dependencies{
		Catalog:                                catalogService,
		TradingUOW:                             tradingUOW,
		DealsUOW:                               dealsUOW,
		ProjectionRepo:                         dealProjectionRepo,
		AuctionLister:                          auctionLister,
		DealLister:                             dealspg.NewDealDeadlineLister(db),
		BillingTx:                              bTx,
		CreateAccount:                          bCreateAccount,
		ReleaseAuctionDepositsExceptCandidates: bReleaseExcept,
		CaptureAuctionDeposit:                  bCapture,
	})
	if err != nil {
		return err
	}

	if err := runtime.Relay.RunOnce(context.Background(), runtime.Bus, 100); err != nil {
		return fmt.Errorf("relay lot published: %w", err)
	}
	if err := bTx.WithinTx(context.Background(), func(txCtx context.Context) error {
		if err := bCreateAccount.Execute(txCtx, "buyer-1"); err != nil {
			return err
		}
		return bConfirm.Execute(txCtx, "buyer-1", 500_000, "chain-runner-bootstrap:"+auctionID)
	}); err != nil {
		return fmt.Errorf("bootstrap buyer wallet: %w", err)
	}

	depositSvc := billingdeposit.NewService(bCreateAccount, bReserve)
	placeBidUC, err := tradingapp.NewPlaceBid(tradingUOW, depositSvc)
	if err != nil {
		return err
	}
	closeAuctionUC, err := tradingapp.NewCloseAuction(tradingUOW)
	if err != nil {
		return err
	}
	if err := placeBidUC.Execute(context.Background(), tradingMetaWithCompany("buyer-1"), tradingapp.AuctionID(auctionID), 150, endsAt.Add(-time.Minute)); err != nil {
		return fmt.Errorf("place bid: %w", err)
	}
	if err := closeAuctionUC.Execute(context.Background(), tradingMetaAdmin(), tradingapp.AuctionID(auctionID)); err != nil {
		return fmt.Errorf("close auction: %w", err)
	}

	if err := runtime.Relay.RunOnce(context.Background(), runtime.Bus, 100); err != nil {
		return fmt.Errorf("relay auction won: %w", err)
	}

	dealItem, err := dealspg.NewDealRepository(db).GetActiveDealByAuctionID(context.Background(), auctionID)
	if err != nil {
		return fmt.Errorf("deal not created: %w", err)
	}
	slog.Info("deal_created", "component", "chain_runner", "deal_id", dealItem.ID(), "buyer_id", dealItem.CustomerID())
	return nil
}

func openRealPostgres() (*sql.DB, bool) {
	db, err := dbconfig.OpenPostgresDockerComposeDefaults(5)
	if err != nil {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, false
	}
	return db, true
}

func applyMigrations(db *sql.DB) error {
	root := repoRoot()
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

func truncateAll(db *sql.DB) error {
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

func repoRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	dir := filepath.Dir(filename)
	return filepath.Clean(filepath.Join(dir, "..", ".."))
}

func tradingMeta() tradingapp.CommandMeta {
	return tradingapp.CommandMeta{
		CompanyID:     "buyer-1",
		UserID:        "buyer-1",
		CorrelationID: "corr-1",
		CausationID:   "cause-1",
	}
}

func tradingMetaWithCompany(companyID string) tradingapp.CommandMeta {
	return tradingapp.CommandMeta{
		CompanyID:     companyID,
		UserID:        companyID,
		CorrelationID: "corr-1",
		CausationID:   "cause-1",
	}
}

func tradingMetaAdmin() tradingapp.CommandMeta {
	return tradingapp.CommandMeta{
		CompanyID:     "admin-1",
		UserID:        "admin-1",
		ActorKind:     tradingapp.ActorKindPlatformAdmin,
		CorrelationID: "corr-1",
		CausationID:   "cause-1",
	}
}

func dealsMeta() dealsapp.CommandMeta {
	return dealsapp.CommandMeta{
		CompanyID:     "buyer-1",
		UserID:        "buyer-1",
		CorrelationID: "corr-1",
		CausationID:   "cause-1",
	}
}

type memoryFishRepo struct {
	items map[string]*catalog.Fish
}

func newMemoryFishRepo() *memoryFishRepo {
	return &memoryFishRepo{items: make(map[string]*catalog.Fish)}
}

func (r *memoryFishRepo) Get(ctx context.Context, fishID string) (*catalog.Fish, error) {
	_ = ctx
	item, ok := r.items[fishID]
	if !ok {
		return nil, catalogapp.ErrNotFound
	}
	return item, nil
}

func (r *memoryFishRepo) List(ctx context.Context) ([]*catalog.Fish, error) {
	_ = ctx
	out := make([]*catalog.Fish, 0, len(r.items))
	for _, fish := range r.items {
		out = append(out, fish)
	}
	return out, nil
}

func (r *memoryFishRepo) Exists(ctx context.Context, fishID string) (bool, error) {
	_ = ctx
	_, ok := r.items[fishID]
	return ok, nil
}

func (r *memoryFishRepo) Save(ctx context.Context, fish *catalog.Fish) error {
	_ = ctx
	r.items[fish.ID()] = fish
	return nil
}

type memoryProductRepo struct {
	items map[string]*catalog.Product
}

func newMemoryProductRepo() *memoryProductRepo {
	return &memoryProductRepo{items: make(map[string]*catalog.Product)}
}

func (r *memoryProductRepo) Get(ctx context.Context, productID string) (*catalog.Product, error) {
	_ = ctx
	item, ok := r.items[productID]
	if !ok {
		return nil, catalogapp.ErrNotFound
	}
	return item, nil
}

func (r *memoryProductRepo) Save(ctx context.Context, product *catalog.Product) error {
	_ = ctx
	r.items[product.ID()] = product
	return nil
}

func (r *memoryProductRepo) List(ctx context.Context) ([]*catalog.Product, error) {
	_ = ctx
	out := make([]*catalog.Product, 0, len(r.items))
	for _, p := range r.items {
		out = append(out, p)
	}
	return out, nil
}

func (r *memoryProductRepo) ListBySellerCompany(ctx context.Context, sellerCompanyID string) ([]*catalog.Product, error) {
	_ = ctx
	var out []*catalog.Product
	for _, p := range r.items {
		if p.SellerCompanyID() == sellerCompanyID {
			out = append(out, p)
		}
	}
	return out, nil
}

type memoryUnitRepo struct{}

func newMemoryUnitRepo() *memoryUnitRepo { return &memoryUnitRepo{} }

func (r *memoryUnitRepo) Exists(ctx context.Context, unit string) (bool, error) {
	_ = ctx
	return unit == "kg" || unit == "g" || unit == "ton", nil
}

type memoryProcessingTypeRepo struct{}

func newMemoryProcessingTypeRepo() *memoryProcessingTypeRepo { return &memoryProcessingTypeRepo{} }

func (r *memoryProcessingTypeRepo) Exists(ctx context.Context, processingType string) (bool, error) {
	_ = ctx
	return processingType == "frozen" || processingType == "chilled" || processingType == "live", nil
}
