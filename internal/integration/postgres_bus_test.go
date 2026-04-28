package integration

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"sync"
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
)

func TestPostgresOutboxBusChains(t *testing.T) {
	db := openIntegrationDB(t, "combined")

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
	auctionID := "auc-e2e"
	lotID := "lot-e2e"

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

	createAuctionUC, err := tradingapp.NewCreateAuction(tradingUOW, fixedAuctionIDFactory{auctionID: tradingapp.AuctionID(auctionID)})
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
			return errors.New("empty winner list")
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

type fixedAuctionIDFactory struct {
	auctionID tradingapp.AuctionID
}

func (f fixedAuctionIDFactory) NewID() (tradingapp.AuctionID, error) {
	return f.auctionID, nil
}

type combinedStore struct {
	mu          sync.Mutex
	lots        map[string]catalogLotRecord
	auctions    map[string]tradingAuctionRecord
	bids        []tradingBidRecord
	winners     []tradingWinnerRecord
	deals       map[string]dealRecord
	projections map[string]projectionRecord
	selections  map[string]selectionRecord
	outbox      []outboxRecord
}

type combinedDriver struct {
	mu     sync.Mutex
	stores map[string]*combinedStore
}

func newCombinedDriver() *combinedDriver {
	return &combinedDriver{stores: make(map[string]*combinedStore)}
}

func (d *combinedDriver) Open(name string) (driver.Conn, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	store, ok := d.stores[name]
	if !ok {
		store = &combinedStore{
			lots:        make(map[string]catalogLotRecord),
			auctions:    make(map[string]tradingAuctionRecord),
			bids:        make([]tradingBidRecord, 0),
			winners:     make([]tradingWinnerRecord, 0),
			deals:       make(map[string]dealRecord),
			projections: make(map[string]projectionRecord),
			selections:  make(map[string]selectionRecord),
			outbox:      make([]outboxRecord, 0),
		}
		d.stores[name] = store
	}
	return &combinedConn{store: store}, nil
}

type combinedConn struct {
	store *combinedStore
}

func (c *combinedConn) Prepare(query string) (driver.Stmt, error) { return combinedStmt{}, nil }
func (c *combinedConn) Close() error                              { return nil }
func (c *combinedConn) Begin() (driver.Tx, error)                 { return combinedTx{}, nil }

func (c *combinedConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	switch {
	case strings.Contains(query, "INSERT INTO catalog_lots"):
		return c.execCatalogLotInsert(args)
	case strings.Contains(query, "INSERT INTO trading_auctions"):
		return c.execTradingAuctionInsert(args)
	case strings.Contains(query, "INSERT INTO trading_bids"):
		return c.execTradingBidInsert(args)
	case strings.Contains(query, "DELETE FROM trading_auction_winners"):
		return c.execTradingWinnersDelete(args)
	case strings.Contains(query, "INSERT INTO trading_auction_winners"):
		return c.execTradingWinnersInsert(args)
	case strings.Contains(query, "INSERT INTO deals"):
		return c.execDealInsert(args)
	case strings.Contains(query, "INSERT INTO deal_projections"):
		return c.execProjectionInsert(args)
	case strings.Contains(query, "INSERT INTO deal_winner_selections"):
		return c.execSelectionInsert(args)
	case strings.Contains(query, "INSERT INTO outbox_messages"):
		return c.execOutboxInsert(args)
	case strings.Contains(query, "UPDATE outbox_messages") && strings.Contains(query, "published_at"):
		return c.execOutboxUpdate(args)
	case strings.Contains(query, "UPDATE outbox_messages") && strings.Contains(query, "attempts"):
		return c.execOutboxFailure(args)
	default:
		return nil, errors.New("unsupported exec query")
	}
}

func (c *combinedConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	switch {
	case strings.Contains(query, "UPDATE outbox_messages") && strings.Contains(query, "locked_at"):
		return c.queryOutboxWithLock(args)
	}
	if len(args) != 1 {
		return nil, errors.New("unexpected query args length")
	}
	arg := args[0].Value
	switch {
	case strings.Contains(query, "FROM catalog_lots"):
		return c.queryCatalogLots(query, arg.(string))
	case strings.Contains(query, "FROM trading_auctions"):
		return c.queryTradingAuctions(arg.(string))
	case strings.Contains(query, "FROM trading_bids"):
		return c.queryTradingBids(arg.(string))
	case strings.Contains(query, "FROM deals") && strings.Contains(query, "WHERE deal_id"):
		return c.queryDealsByID(arg.(string))
	case strings.Contains(query, "FROM deals") && strings.Contains(query, "WHERE auction_id"):
		return c.queryDealsByAuctionID(arg.(string))
	case strings.Contains(query, "FROM deal_projections"):
		return c.queryDealProjections(arg.(string))
	case strings.Contains(query, "FROM deal_winner_selections"):
		return c.querySelections(arg.(string))
	case strings.Contains(query, "FROM outbox_messages"):
		return c.queryOutbox(toInt64(arg))
	default:
		return nil, errors.New("unsupported query")
	}
}

type combinedStmt struct{}

func (combinedStmt) Close() error  { return nil }
func (combinedStmt) NumInput() int { return -1 }
func (combinedStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("unsupported")
}
func (combinedStmt) Query([]driver.Value) (driver.Rows, error) { return nil, errors.New("unsupported") }

type combinedTx struct{}

func (combinedTx) Commit() error   { return nil }
func (combinedTx) Rollback() error { return nil }

type combinedRows struct {
	values [][]driver.Value
	pos    int
}

func (r *combinedRows) Columns() []string {
	if len(r.values) == 0 {
		return []string{}
	}
	cols := make([]string, len(r.values[0]))
	for i := range cols {
		cols[i] = "col"
	}
	return cols
}

func (r *combinedRows) Close() error { return nil }
func (r *combinedRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.pos])
	r.pos++
	return nil
}

// Catalog lot handling
type catalogLotRecord struct {
	lotID                  string
	productID              string
	auctionID              sql.NullString
	sellerCompanyID        string
	photo                  sql.NullString
	quantity               float64
	startPrice             int64
	minBidStep             int64
	curPrice               int64
	finalPrice             int64
	status                 string
	auctionStartsAt        time.Time
	auctionDurationMinutes int64
}

func (c *combinedConn) execCatalogLotInsert(args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 13 {
		return nil, errors.New("unexpected lot args length")
	}
	record := catalogLotRecord{
		lotID:                  args[0].Value.(string),
		productID:              args[1].Value.(string),
		auctionID:              toNullString(args[2].Value),
		sellerCompanyID:        args[3].Value.(string),
		photo:                  toNullString(args[4].Value),
		quantity:               args[5].Value.(float64),
		startPrice:             args[6].Value.(int64),
		minBidStep:             args[7].Value.(int64),
		curPrice:               args[8].Value.(int64),
		finalPrice:             args[9].Value.(int64),
		status:                 args[10].Value.(string),
		auctionStartsAt:        args[11].Value.(time.Time),
		auctionDurationMinutes: args[12].Value.(int64),
	}
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	c.store.lots[record.lotID] = record
	return driver.RowsAffected(1), nil
}

func (c *combinedConn) queryCatalogLots(query, arg string) (driver.Rows, error) {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	var record catalogLotRecord
	var ok bool
	if strings.Contains(query, "WHERE lot_id") {
		record, ok = c.store.lots[arg]
	} else {
		for _, r := range c.store.lots {
			if r.auctionID.Valid && r.auctionID.String == arg {
				record = r
				ok = true
				break
			}
		}
	}
	if !ok {
		return &combinedRows{}, nil
	}
	return &combinedRows{values: [][]driver.Value{{
		record.lotID,
		record.productID,
		nullStringValue(record.auctionID),
		record.sellerCompanyID,
		nullStringValue(record.photo),
		record.quantity,
		record.startPrice,
		record.minBidStep,
		record.curPrice,
		record.finalPrice,
		record.status,
		record.auctionStartsAt,
		record.auctionDurationMinutes,
	}}}, nil
}

// Trading handling
type tradingAuctionRecord struct {
	auctionID       string
	lotID           string
	state           string
	startsAt        time.Time
	endsAt          time.Time
	currentPrice    int64
	minBidStep      int64
	leaderCompanyID string
}

type tradingBidRecord struct {
	auctionID       string
	bidderCompanyID string
	amount          int64
	placedAt        time.Time
}

type tradingWinnerRecord struct {
	auctionID string
	place     int
	companyID string
	amount    int64
	placedAt  time.Time
}

func (c *combinedConn) execTradingAuctionInsert(args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 8 {
		return nil, errors.New("unexpected auction args length")
	}
	record := tradingAuctionRecord{
		auctionID:       args[0].Value.(string),
		lotID:           args[1].Value.(string),
		state:           args[2].Value.(string),
		startsAt:        args[3].Value.(time.Time),
		endsAt:          args[4].Value.(time.Time),
		currentPrice:    args[5].Value.(int64),
		minBidStep:      args[6].Value.(int64),
		leaderCompanyID: args[7].Value.(string),
	}
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	c.store.auctions[record.auctionID] = record
	return driver.RowsAffected(1), nil
}

func (c *combinedConn) execTradingBidInsert(args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 4 {
		return nil, errors.New("unexpected bid args length")
	}
	record := tradingBidRecord{
		auctionID:       args[0].Value.(string),
		bidderCompanyID: args[1].Value.(string),
		amount:          args[2].Value.(int64),
		placedAt:        args[3].Value.(time.Time),
	}
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	c.store.bids = append(c.store.bids, record)
	return driver.RowsAffected(1), nil
}

func (c *combinedConn) execTradingWinnersDelete(args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 1 {
		return nil, errors.New("unexpected winners delete args length")
	}
	auctionID := args[0].Value.(string)
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	remaining := c.store.winners[:0]
	for _, record := range c.store.winners {
		if record.auctionID != auctionID {
			remaining = append(remaining, record)
		}
	}
	c.store.winners = remaining
	return driver.RowsAffected(1), nil
}

func (c *combinedConn) execTradingWinnersInsert(args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 5 {
		return nil, errors.New("unexpected winners insert args length")
	}
	record := tradingWinnerRecord{
		auctionID: args[0].Value.(string),
		place:     toInt(args[1].Value),
		companyID: args[2].Value.(string),
		amount:    args[3].Value.(int64),
		placedAt:  args[4].Value.(time.Time),
	}
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	c.store.winners = append(c.store.winners, record)
	return driver.RowsAffected(1), nil
}

func (c *combinedConn) queryTradingAuctions(auctionID string) (driver.Rows, error) {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	record, ok := c.store.auctions[auctionID]
	if !ok {
		return &combinedRows{}, nil
	}
	return &combinedRows{values: [][]driver.Value{{
		record.auctionID,
		record.lotID,
		record.state,
		record.startsAt,
		record.endsAt,
		record.currentPrice,
		record.minBidStep,
		record.leaderCompanyID,
	}}}, nil
}

func (c *combinedConn) queryTradingBids(auctionID string) (driver.Rows, error) {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	var rows []tradingBidRecord
	for _, record := range c.store.bids {
		if record.auctionID == auctionID {
			rows = append(rows, record)
		}
	}
	values := make([][]driver.Value, 0, len(rows))
	for _, row := range rows {
		values = append(values, []driver.Value{row.bidderCompanyID, row.amount, row.placedAt})
	}
	return &combinedRows{values: values}, nil
}

// Deals handling
type dealRecord struct {
	dealID                string
	customerID            string
	supplierID            string
	auctionID             string
	quantity              int64
	unitPrice             int64
	status                string
	typeName              string
	createdAt             time.Time
	confirmedAt           sql.NullTime
	contractSignDeadline  sql.NullTime
	paymentDeadline       sql.NullTime
	contractNumber        sql.NullString
	contractPrepared      sql.NullTime
	contractSigned        sql.NullTime
	contractSignedBy      sql.NullString
	signatureRef          sql.NullString
	documentURL           sql.NullString
	productID             string
	productName           string
	productDescription    string
	productCategory       string
	productWeight         float64
	productUnit           string
	productSize           string
	productProcessingType string
	productVolume         float64
	productOrigin         string
}

type projectionRecord struct {
	auctionID             string
	supplierID            string
	startPrice            int64
	publishedAt           time.Time
	status                string
	productID             string
	productName           string
	productDescription    string
	productCategory       string
	productWeight         float64
	productUnit           string
	productSize           string
	productProcessingType string
	productVolume         float64
	productOrigin         string
}

type selectionRecord struct {
	auctionID             string
	candidates            []byte
	currentIndex          int
	status                string
	finalPrice            int64
	wonAt                 time.Time
	supplierID            string
	dealID                sql.NullString
	productID             string
	productName           string
	productDescription    string
	productCategory       string
	productWeight         float64
	productUnit           string
	productSize           string
	productProcessingType string
	productVolume         float64
	productOrigin         string
}

func (c *combinedConn) execDealInsert(args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 28 {
		return nil, errors.New("unexpected deal args length")
	}
	record := dealRecord{
		dealID:                args[0].Value.(string),
		customerID:            args[1].Value.(string),
		supplierID:            args[2].Value.(string),
		auctionID:             args[3].Value.(string),
		quantity:              args[4].Value.(int64),
		unitPrice:             args[5].Value.(int64),
		status:                args[6].Value.(string),
		typeName:              args[7].Value.(string),
		createdAt:             args[8].Value.(time.Time),
		confirmedAt:           toNullTime(args[9].Value),
		contractSignDeadline:  toNullTime(args[10].Value),
		paymentDeadline:       toNullTime(args[11].Value),
		contractNumber:        toNullString(args[12].Value),
		contractPrepared:      toNullTime(args[13].Value),
		contractSigned:        toNullTime(args[14].Value),
		contractSignedBy:      toNullString(args[15].Value),
		signatureRef:          toNullString(args[16].Value),
		documentURL:           toNullString(args[17].Value),
		productID:             args[18].Value.(string),
		productName:           args[19].Value.(string),
		productDescription:    args[20].Value.(string),
		productCategory:       args[21].Value.(string),
		productWeight:         args[22].Value.(float64),
		productUnit:           args[23].Value.(string),
		productSize:           args[24].Value.(string),
		productProcessingType: args[25].Value.(string),
		productVolume:         args[26].Value.(float64),
		productOrigin:         args[27].Value.(string),
	}
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	c.store.deals[record.dealID] = record
	return driver.RowsAffected(1), nil
}

func (c *combinedConn) execProjectionInsert(args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 15 {
		return nil, errors.New("unexpected projection args length")
	}
	record := projectionRecord{
		auctionID:             args[0].Value.(string),
		supplierID:            args[1].Value.(string),
		startPrice:            args[2].Value.(int64),
		publishedAt:           args[3].Value.(time.Time),
		status:                args[4].Value.(string),
		productID:             args[5].Value.(string),
		productName:           args[6].Value.(string),
		productDescription:    args[7].Value.(string),
		productCategory:       args[8].Value.(string),
		productWeight:         args[9].Value.(float64),
		productUnit:           args[10].Value.(string),
		productSize:           args[11].Value.(string),
		productProcessingType: args[12].Value.(string),
		productVolume:         args[13].Value.(float64),
		productOrigin:         args[14].Value.(string),
	}
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	c.store.projections[record.auctionID] = record
	return driver.RowsAffected(1), nil
}

func (c *combinedConn) execSelectionInsert(args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 18 {
		return nil, errors.New("unexpected selection args length")
	}
	record := selectionRecord{
		auctionID:             args[0].Value.(string),
		candidates:            args[1].Value.([]byte),
		currentIndex:          toInt(args[2].Value),
		status:                args[3].Value.(string),
		finalPrice:            args[4].Value.(int64),
		wonAt:                 args[5].Value.(time.Time),
		supplierID:            args[6].Value.(string),
		dealID:                toNullString(args[7].Value),
		productID:             args[8].Value.(string),
		productName:           args[9].Value.(string),
		productDescription:    args[10].Value.(string),
		productCategory:       args[11].Value.(string),
		productWeight:         args[12].Value.(float64),
		productUnit:           args[13].Value.(string),
		productSize:           args[14].Value.(string),
		productProcessingType: args[15].Value.(string),
		productVolume:         args[16].Value.(float64),
		productOrigin:         args[17].Value.(string),
	}
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	c.store.selections[record.auctionID] = record
	return driver.RowsAffected(1), nil
}

func (c *combinedConn) queryDealsByID(id string) (driver.Rows, error) {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	record, ok := c.store.deals[id]
	if !ok {
		return &combinedRows{}, nil
	}
	return &combinedRows{values: [][]driver.Value{dealRow(record)}}, nil
}

func (c *combinedConn) queryDealsByAuctionID(auctionID string) (driver.Rows, error) {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	for _, record := range c.store.deals {
		if record.auctionID == auctionID {
			return &combinedRows{values: [][]driver.Value{dealRow(record)}}, nil
		}
	}
	return &combinedRows{}, nil
}

func (c *combinedConn) queryDealProjections(auctionID string) (driver.Rows, error) {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	record, ok := c.store.projections[auctionID]
	if !ok {
		return &combinedRows{}, nil
	}
	return &combinedRows{values: [][]driver.Value{projectionRow(record)}}, nil
}

func (c *combinedConn) querySelections(auctionID string) (driver.Rows, error) {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	record, ok := c.store.selections[auctionID]
	if !ok {
		return &combinedRows{}, nil
	}
	return &combinedRows{values: [][]driver.Value{selectionRow(record)}}, nil
}

// Outbox handling
type outboxRecord struct {
	id            string
	eventType     string
	sourceContext string
	payload       []byte
	occurredAt    time.Time
	createdAt     time.Time
	publishedAt   sql.NullTime
	lockedAt      sql.NullTime
	attempts      int
	failedAt      sql.NullTime
	lastError     string
}

func (c *combinedConn) execOutboxInsert(args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 13 {
		return nil, errors.New("unexpected outbox args length")
	}
	payload, ok := args[4].Value.([]byte)
	if !ok {
		return nil, errors.New("payload must be []byte")
	}
	record := outboxRecord{
		id:            args[0].Value.(string),
		eventType:     args[2].Value.(string),
		sourceContext: args[11].Value.(string),
		payload:       append([]byte(nil), payload...),
		occurredAt:    args[5].Value.(time.Time),
		createdAt:     args[6].Value.(time.Time),
		publishedAt:   sql.NullTime{},
		lockedAt:      sql.NullTime{},
		attempts:      0,
		failedAt:      sql.NullTime{},
		lastError:     "",
	}
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	c.store.outbox = append(c.store.outbox, record)
	return driver.RowsAffected(1), nil
}

func (c *combinedConn) execOutboxUpdate(args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 2 {
		return nil, errors.New("unexpected outbox update args length")
	}
	publishedAt := args[0].Value.(time.Time)
	id := args[1].Value.(string)
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	for i := range c.store.outbox {
		if c.store.outbox[i].id == id {
			c.store.outbox[i].publishedAt = sql.NullTime{Time: publishedAt, Valid: true}
			return driver.RowsAffected(1), nil
		}
	}
	return driver.RowsAffected(0), nil
}

func (c *combinedConn) execOutboxFailure(args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 4 {
		return nil, errors.New("unexpected outbox failure args length")
	}
	attempts := toInt(args[0].Value)
	errMsg, _ := args[1].Value.(string)
	failedAt := toNullTime(args[2].Value)
	id := args[3].Value.(string)
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	for i := range c.store.outbox {
		if c.store.outbox[i].id == id {
			c.store.outbox[i].attempts = attempts
			c.store.outbox[i].lastError = errMsg
			c.store.outbox[i].failedAt = failedAt
			return driver.RowsAffected(1), nil
		}
	}
	return driver.RowsAffected(0), nil
}

func (c *combinedConn) queryOutbox(limit int64) (driver.Rows, error) {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	var out []outboxRecord
	for _, record := range c.store.outbox {
		if !record.publishedAt.Valid && !record.failedAt.Valid {
			out = append(out, record)
		}
		if limit > 0 && int64(len(out)) >= limit {
			break
		}
	}
	values := make([][]driver.Value, 0, len(out))
	for _, row := range out {
		values = append(values, []driver.Value{
			row.id,
			row.eventType,
			row.sourceContext,
			row.payload,
			row.occurredAt,
			row.attempts,
		})
	}
	return &combinedRows{values: values}, nil
}

func (c *combinedConn) queryOutboxWithLock(args []driver.NamedValue) (driver.Rows, error) {
	if len(args) != 3 {
		return nil, errors.New("unexpected outbox lock args length")
	}
	lockTime := args[0].Value.(time.Time)
	lockCutoff := args[1].Value.(time.Time)
	limit := toInt64(args[2].Value)

	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	var out []outboxRecord
	for i := range c.store.outbox {
		record := &c.store.outbox[i]
		if record.publishedAt.Valid || record.failedAt.Valid {
			continue
		}
		if record.lockedAt.Valid && record.lockedAt.Time.After(lockCutoff) {
			continue
		}
		record.lockedAt = sql.NullTime{Time: lockTime, Valid: true}
		out = append(out, *record)
		if limit > 0 && int64(len(out)) >= limit {
			break
		}
	}
	values := make([][]driver.Value, 0, len(out))
	for _, row := range out {
		values = append(values, []driver.Value{
			row.id,
			row.eventType,
			row.sourceContext,
			row.payload,
			row.occurredAt,
			row.attempts,
			nil,
			nil,
			nil,
			nil,
		})
	}
	return &combinedRows{values: values}, nil
}

func dealRow(record dealRecord) []driver.Value {
	return []driver.Value{
		record.dealID,
		record.customerID,
		record.supplierID,
		record.auctionID,
		record.quantity,
		record.unitPrice,
		record.status,
		record.typeName,
		record.createdAt,
		nullTimeValue(record.confirmedAt),
		nullTimeValue(record.contractSignDeadline),
		nullTimeValue(record.paymentDeadline),
		nullStringValue(record.contractNumber),
		nullTimeValue(record.contractPrepared),
		nullTimeValue(record.contractSigned),
		nullStringValue(record.contractSignedBy),
		nullStringValue(record.signatureRef),
		nullStringValue(record.documentURL),
		record.productID,
		record.productName,
		record.productDescription,
		record.productCategory,
		record.productWeight,
		record.productUnit,
		record.productSize,
		record.productProcessingType,
		record.productVolume,
		record.productOrigin,
	}
}

func projectionRow(record projectionRecord) []driver.Value {
	return []driver.Value{
		record.auctionID,
		record.supplierID,
		record.startPrice,
		record.publishedAt,
		record.status,
		record.productID,
		record.productName,
		record.productDescription,
		record.productCategory,
		record.productWeight,
		record.productUnit,
		record.productSize,
		record.productProcessingType,
		record.productVolume,
		record.productOrigin,
	}
}

func selectionRow(record selectionRecord) []driver.Value {
	return []driver.Value{
		record.auctionID,
		record.candidates,
		record.currentIndex,
		record.status,
		record.finalPrice,
		record.wonAt,
		record.supplierID,
		nullStringValue(record.dealID),
		record.productID,
		record.productName,
		record.productDescription,
		record.productCategory,
		record.productWeight,
		record.productUnit,
		record.productSize,
		record.productProcessingType,
		record.productVolume,
		record.productOrigin,
	}
}

func toNullString(value any) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	if ns, ok := value.(sql.NullString); ok {
		return ns
	}
	if s, ok := value.(string); ok {
		return sql.NullString{String: s, Valid: s != ""}
	}
	return sql.NullString{}
}

func toNullTime(value any) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	if nt, ok := value.(sql.NullTime); ok {
		return nt
	}
	if t, ok := value.(time.Time); ok {
		return sql.NullTime{Time: t, Valid: true}
	}
	return sql.NullTime{}
}

func nullStringValue(value sql.NullString) any {
	if value.Valid {
		return value.String
	}
	return nil
}

func nullTimeValue(value sql.NullTime) any {
	if value.Valid {
		return value.Time
	}
	return nil
}

func toInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	default:
		return 0
	}
}

func toInt64(value any) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	default:
		return 0
	}
}

var (
	registerCombinedDriverOnce sync.Once
	combinedTestDriver         = newCombinedDriver()
)

func openIntegrationDB(t *testing.T, name string) *sql.DB {
	t.Helper()
	registerCombinedDriverOnce.Do(func() {
		sql.Register("combined-integration-driver", combinedTestDriver)
	})
	db, err := sql.Open("combined-integration-driver", name)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(5)
	t.Cleanup(func() { _ = db.Close() })
	return db
}
