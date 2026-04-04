package postgres

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

	"github.com/EBal0vGG/Unbelievable_Fish/domain/catalog"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
)

type lotStore struct {
	mu      sync.Mutex
	records map[string]lotRecord
}

type lotRecord struct {
	lotID           string
	productID       string
	auctionID       sql.NullString
	sellerCompanyID string
	photo           sql.NullString
	quantity        float64
	startPrice      int64
	curPrice        int64
	finalPrice      int64
	status          string
	auctionStartsAt time.Time
}

func cloneRecords(src map[string]lotRecord) map[string]lotRecord {
	out := make(map[string]lotRecord, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

type lotRepoDriver struct {
	mu     sync.Mutex
	stores map[string]*lotStore
}

func newLotRepoDriver() *lotRepoDriver {
	return &lotRepoDriver{stores: make(map[string]*lotStore)}
}

func (d *lotRepoDriver) Open(name string) (driver.Conn, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	store, ok := d.stores[name]
	if !ok {
		store = &lotStore{records: make(map[string]lotRecord)}
		d.stores[name] = store
	}

	return &lotRepoConn{store: store}, nil
}

func (d *lotRepoDriver) store(name string) *lotStore {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.stores[name]
}

type lotRepoConn struct {
	store   *lotStore
	txStore map[string]lotRecord
}

func (c *lotRepoConn) Prepare(query string) (driver.Stmt, error) { return lotRepoStmt{}, nil }
func (c *lotRepoConn) Close() error                              { return nil }
func (c *lotRepoConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *lotRepoConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	c.txStore = cloneRecords(c.store.records)
	return &lotRepoTx{conn: c}, nil
}

func (c *lotRepoConn) ExecContext(_ context.Context, _ string, args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 11 {
		return nil, errors.New("unexpected args length")
	}

	record := lotRecord{
		lotID:           args[0].Value.(string),
		productID:       args[1].Value.(string),
		auctionID:       toNullString(args[2].Value),
		sellerCompanyID: args[3].Value.(string),
		photo:           toNullString(args[4].Value),
		quantity:        args[5].Value.(float64),
		startPrice:      args[6].Value.(int64),
		curPrice:        args[7].Value.(int64),
		finalPrice:      args[8].Value.(int64),
		status:          args[9].Value.(string),
		auctionStartsAt: args[10].Value.(time.Time),
	}

	if c.txStore != nil {
		c.txStore[record.lotID] = record
		return driver.RowsAffected(1), nil
	}

	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	c.store.records[record.lotID] = record
	return driver.RowsAffected(1), nil
}

func (c *lotRepoConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if len(args) != 1 {
		return nil, errors.New("unexpected args length")
	}

	var record lotRecord
	var ok bool

	if strings.Contains(query, "WHERE lot_id = $1") {
		record, ok = c.lookupByLotID(args[0].Value.(string))
	} else if strings.Contains(query, "WHERE auction_id = $1") {
		record, ok = c.lookupByAuctionID(args[0].Value.(string))
	} else {
		return nil, errors.New("unsupported query")
	}

	if !ok {
		return &lotRepoRows{}, nil
	}

	return &lotRepoRows{
		values: [][]driver.Value{{
			record.lotID,
			record.productID,
			nullStringValue(record.auctionID),
			record.sellerCompanyID,
			nullStringValue(record.photo),
			record.quantity,
			record.startPrice,
			record.curPrice,
			record.finalPrice,
			record.status,
			record.auctionStartsAt,
		}},
	}, nil
}

func (c *lotRepoConn) lookupByLotID(lotID string) (lotRecord, bool) {
	if c.txStore != nil {
		record, ok := c.txStore[lotID]
		return record, ok
	}
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	record, ok := c.store.records[lotID]
	return record, ok
}

func (c *lotRepoConn) lookupByAuctionID(auctionID string) (lotRecord, bool) {
	records := c.txStore
	if records == nil {
		c.store.mu.Lock()
		defer c.store.mu.Unlock()
		records = c.store.records
	}

	for _, record := range records {
		if record.auctionID.Valid && record.auctionID.String == auctionID {
			return record, true
		}
	}
	return lotRecord{}, false
}

type lotRepoTx struct {
	conn *lotRepoConn
}

func (t *lotRepoTx) Commit() error {
	t.conn.store.mu.Lock()
	defer t.conn.store.mu.Unlock()
	t.conn.store.records = cloneRecords(t.conn.txStore)
	t.conn.txStore = nil
	return nil
}

func (t *lotRepoTx) Rollback() error {
	t.conn.txStore = nil
	return nil
}

type lotRepoStmt struct{}

func (lotRepoStmt) Close() error                               { return nil }
func (lotRepoStmt) NumInput() int                              { return -1 }
func (lotRepoStmt) Exec([]driver.Value) (driver.Result, error) { return driver.RowsAffected(0), nil }
func (lotRepoStmt) Query([]driver.Value) (driver.Rows, error)  { return &lotRepoRows{}, nil }

type lotRepoRows struct {
	values [][]driver.Value
	index  int
}

func (lotRepoRows) Columns() []string {
	return []string{
		"lot_id",
		"product_id",
		"auction_id",
		"seller_company_id",
		"photo",
		"quantity",
		"start_price",
		"cur_price",
		"final_price",
		"status",
		"auction_starts_at",
	}
}

func (lotRepoRows) Close() error { return nil }

func (r *lotRepoRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}

func toNullString(value any) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: value.(string), Valid: true}
}

func nullStringValue(value sql.NullString) any {
	if !value.Valid {
		return nil
	}
	return value.String
}

var (
	registerLotRepoDriverOnce sync.Once
	testLotRepoDriver         = newLotRepoDriver()
)

func openLotRepoDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()

	registerLotRepoDriverOnce.Do(func() {
		sql.Register("catalog-lot-repo-stub", testLotRepoDriver)
	})

	db, err := sql.Open("catalog-lot-repo-stub", dsn)
	if err != nil {
		t.Fatalf("unexpected open error: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func newDraftLot(t *testing.T, lotID string) *catalog.Lot {
	t.Helper()

	lot, _, err := catalog.NewLot(
		lotID,
		"prod-1",
		"seller-1",
		"photo-key",
		10.5,
		100,
		catalog.NewAuctionScheduleAt(time.Now().Add(time.Hour)),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return lot
}

func newPublishedLot(t *testing.T, lotID, auctionID string) *catalog.Lot {
	t.Helper()

	lot := newDraftLot(t, lotID)
	if _, err := lot.AssignAuctionID(auctionID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := lot.Publish(true, catalog.ProductSnapshot{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := lot.UpdateCurrentPrice(140); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return lot
}

func TestLotRepositoryWorksOutsideTransaction(t *testing.T) {
	db := openLotRepoDB(t, "outside-tx")
	repo := NewLotRepository(db)

	lot := newDraftLot(t, "lot-1")
	if err := repo.Save(context.Background(), lot); err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}

	stored, err := repo.Get(context.Background(), "lot-1")
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if stored.ID() != "lot-1" {
		t.Fatalf("expected lot id to match, got %s", stored.ID())
	}
	if stored.Status() != catalog.LotStatusDraft {
		t.Fatalf("expected draft status, got %s", stored.Status())
	}
	if stored.Photo() != "photo-key" {
		t.Fatalf("expected photo to match, got %s", stored.Photo())
	}
}

func TestLotRepositoryUsesTxAndCommits(t *testing.T) {
	db := openLotRepoDB(t, "commit-tx")
	repo := NewLotRepository(db)
	txManager := NewTransactionManager(db, nil)

	err := txManager.WithinTx(context.Background(), func(ctx context.Context) error {
		lot := newPublishedLot(t, "lot-2", "auc-2")
		if err := repo.Save(ctx, lot); err != nil {
			return err
		}

		stored, err := repo.GetByAuctionID(ctx, "auc-2")
		if err != nil {
			return err
		}
		if stored.Status() != catalog.LotStatusPublished {
			t.Fatalf("expected published status inside tx, got %s", stored.Status())
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected tx error: %v", err)
	}

	stored, err := repo.Get(context.Background(), "lot-2")
	if err != nil {
		t.Fatalf("unexpected get error after commit: %v", err)
	}
	if stored.Status() != catalog.LotStatusPublished {
		t.Fatalf("expected published status after commit, got %s", stored.Status())
	}
	if stored.CurPrice() != 140 {
		t.Fatalf("expected current price to match, got %d", stored.CurPrice())
	}
}

func TestLotRepositoryRollsBackWhenTransactionFails(t *testing.T) {
	db := openLotRepoDB(t, "rollback-tx")
	repo := NewLotRepository(db)
	txManager := NewTransactionManager(db, nil)
	useCaseErr := errors.New("force rollback")

	err := txManager.WithinTx(context.Background(), func(ctx context.Context) error {
		lot := newDraftLot(t, "lot-3")
		if err := repo.Save(ctx, lot); err != nil {
			return err
		}
		return useCaseErr
	})
	if !errors.Is(err, useCaseErr) {
		t.Fatalf("expected rollback error, got %v", err)
	}

	_, err = repo.Get(context.Background(), "lot-3")
	if !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after rollback, got %v", err)
	}
}
