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
)

type integrationLotRecord struct {
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
	auctionDurationMinutes int64
}

type integrationOutboxRecord struct {
	id          string
	eventID     string
	eventType   string
	aggregateID string
	payload     []byte
	occurredAt  time.Time
	createdAt   time.Time
	sourceContext string
	publishedAt sql.NullTime
}

type integrationStore struct {
	mu     sync.Mutex
	lots   map[string]integrationLotRecord
	outbox []integrationOutboxRecord
}

func cloneLots(src map[string]integrationLotRecord) map[string]integrationLotRecord {
	out := make(map[string]integrationLotRecord, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func cloneOutbox(src []integrationOutboxRecord) []integrationOutboxRecord {
	out := make([]integrationOutboxRecord, len(src))
	copy(out, src)
	return out
}

type integrationDriver struct {
	mu     sync.Mutex
	stores map[string]*integrationStore
}

func newIntegrationDriver() *integrationDriver {
	return &integrationDriver{stores: make(map[string]*integrationStore)}
}

func (d *integrationDriver) Open(name string) (driver.Conn, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	store, ok := d.stores[name]
	if !ok {
		store = &integrationStore{
			lots:   make(map[string]integrationLotRecord),
			outbox: make([]integrationOutboxRecord, 0),
		}
		d.stores[name] = store
	}

	return &integrationConn{store: store}, nil
}

func (d *integrationDriver) store(name string) *integrationStore {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.stores[name]
}

type integrationConn struct {
	store    *integrationStore
	txLots   map[string]integrationLotRecord
	txOutbox []integrationOutboxRecord
}

func (c *integrationConn) Prepare(query string) (driver.Stmt, error) { return integrationStmt{}, nil }
func (c *integrationConn) Close() error                              { return nil }
func (c *integrationConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *integrationConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()

	c.txLots = cloneLots(c.store.lots)
	c.txOutbox = cloneOutbox(c.store.outbox)
	return &integrationTx{conn: c}, nil
}

func (c *integrationConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	switch {
	case strings.Contains(query, "INSERT INTO catalog_lots"):
		return c.execLotInsert(args)
	case strings.Contains(query, "INSERT INTO outbox_messages"):
		return c.execOutboxInsert(args)
	default:
		return nil, errors.New("unsupported exec query")
	}
}

func (c *integrationConn) execLotInsert(args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 12 {
		return nil, errors.New("unexpected lot args length")
	}

	record := integrationLotRecord{
		lotID:           args[0].Value.(string),
		productID:       args[1].Value.(string),
		auctionID:       toIntegrationNullString(args[2].Value),
		sellerCompanyID: args[3].Value.(string),
		photo:           toIntegrationNullString(args[4].Value),
		quantity:        args[5].Value.(float64),
		startPrice:      args[6].Value.(int64),
		curPrice:        args[7].Value.(int64),
		finalPrice:      args[8].Value.(int64),
		status:          args[9].Value.(string),
		auctionStartsAt: args[10].Value.(time.Time),
		auctionDurationMinutes: args[11].Value.(int64),
	}

	if c.txLots != nil {
		c.txLots[record.lotID] = record
		return driver.RowsAffected(1), nil
	}

	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	c.store.lots[record.lotID] = record
	return driver.RowsAffected(1), nil
}

func (c *integrationConn) execOutboxInsert(args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 13 {
		return nil, errors.New("unexpected outbox args length")
	}

	payload, ok := args[4].Value.([]byte)
	if !ok {
		return nil, errors.New("payload must be []byte")
	}

	record := integrationOutboxRecord{
		id:          args[0].Value.(string),
		eventID:     args[1].Value.(string),
		eventType:   args[2].Value.(string),
		aggregateID: args[3].Value.(string),
		payload:     append([]byte(nil), payload...),
		occurredAt:  args[5].Value.(time.Time),
		createdAt:   args[6].Value.(time.Time),
		sourceContext: args[11].Value.(string),
		publishedAt: sql.NullTime{},
	}

	if c.txOutbox != nil {
		c.txOutbox = append(c.txOutbox, record)
		return driver.RowsAffected(1), nil
	}

	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	c.store.outbox = append(c.store.outbox, record)
	return driver.RowsAffected(1), nil
}

func (c *integrationConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if len(args) != 1 {
		return nil, errors.New("unexpected query args length")
	}

	var (
		record integrationLotRecord
		ok     bool
	)

	switch {
	case strings.Contains(query, "FROM catalog_lots") && strings.Contains(query, "WHERE lot_id = $1"):
		record, ok = c.lookupByLotID(args[0].Value.(string))
	case strings.Contains(query, "FROM catalog_lots") && strings.Contains(query, "WHERE auction_id = $1"):
		record, ok = c.lookupByAuctionID(args[0].Value.(string))
	default:
		return nil, errors.New("unsupported query")
	}

	if !ok {
		return &integrationRows{}, nil
	}

	return &integrationRows{
		values: [][]driver.Value{{
			record.lotID,
			record.productID,
			integrationNullStringValue(record.auctionID),
			record.sellerCompanyID,
			integrationNullStringValue(record.photo),
			record.quantity,
			record.startPrice,
			record.curPrice,
			record.finalPrice,
			record.status,
			record.auctionStartsAt,
			record.auctionDurationMinutes,
		}},
	}, nil
}

func (c *integrationConn) lookupByLotID(lotID string) (integrationLotRecord, bool) {
	if c.txLots != nil {
		record, ok := c.txLots[lotID]
		return record, ok
	}
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	record, ok := c.store.lots[lotID]
	return record, ok
}

func (c *integrationConn) lookupByAuctionID(auctionID string) (integrationLotRecord, bool) {
	records := c.txLots
	if records == nil {
		c.store.mu.Lock()
		defer c.store.mu.Unlock()
		records = c.store.lots
	}

	for _, record := range records {
		if record.auctionID.Valid && record.auctionID.String == auctionID {
			return record, true
		}
	}
	return integrationLotRecord{}, false
}

type integrationTx struct {
	conn *integrationConn
}

func (t *integrationTx) Commit() error {
	t.conn.store.mu.Lock()
	defer t.conn.store.mu.Unlock()

	t.conn.store.lots = cloneLots(t.conn.txLots)
	t.conn.store.outbox = cloneOutbox(t.conn.txOutbox)
	t.conn.txLots = nil
	t.conn.txOutbox = nil
	return nil
}

func (t *integrationTx) Rollback() error {
	t.conn.txLots = nil
	t.conn.txOutbox = nil
	return nil
}

type integrationStmt struct{}

func (integrationStmt) Close() error  { return nil }
func (integrationStmt) NumInput() int { return -1 }
func (integrationStmt) Exec([]driver.Value) (driver.Result, error) {
	return driver.RowsAffected(0), nil
}
func (integrationStmt) Query([]driver.Value) (driver.Rows, error) { return &integrationRows{}, nil }

type integrationRows struct {
	values [][]driver.Value
	index  int
}

func (integrationRows) Columns() []string {
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
		"auction_duration_minutes",
	}
}

func (integrationRows) Close() error { return nil }

func (r *integrationRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}

func toIntegrationNullString(value any) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: value.(string), Valid: true}
}

func integrationNullStringValue(value sql.NullString) any {
	if !value.Valid {
		return nil
	}
	return value.String
}

var (
	registerIntegrationDriverOnce sync.Once
	testIntegrationDriver         = newIntegrationDriver()
)

func openIntegrationDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()

	registerIntegrationDriverOnce.Do(func() {
		sql.Register("catalog-postgres-integration-stub", testIntegrationDriver)
	})

	db, err := sql.Open("catalog-postgres-integration-stub", dsn)
	if err != nil {
		t.Fatalf("unexpected open error: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}
