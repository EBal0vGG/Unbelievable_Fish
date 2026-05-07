package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

type integrationAuctionRecord struct {
	auctionID       string
	lotID           string
	state           string
	startsAt        time.Time
	endsAt          time.Time
	startPrice      int64
	currentPrice    int64
	minBidStep      int64
	leaderCompanyID string
}

type integrationBidRecord struct {
	auctionID       string
	bidderCompanyID string
	amount          int64
	placedAt        time.Time
}

type integrationWinnerRecord struct {
	auctionID string
	place     int
	companyID string
	amount    int64
	placedAt  time.Time
}

type integrationOutboxRecord struct {
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

type integrationStore struct {
	mu       sync.Mutex
	auctions map[string]integrationAuctionRecord
	bids     []integrationBidRecord
	winners  []integrationWinnerRecord
	outbox   []integrationOutboxRecord
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
			auctions: make(map[string]integrationAuctionRecord),
			bids:     make([]integrationBidRecord, 0),
			winners:  make([]integrationWinnerRecord, 0),
			outbox:   make([]integrationOutboxRecord, 0),
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
	store *integrationStore
}

func (c *integrationConn) Prepare(query string) (driver.Stmt, error) { return integrationStmt{}, nil }
func (c *integrationConn) Close() error                              { return nil }
func (c *integrationConn) Begin() (driver.Tx, error)                 { return integrationTx{}, nil }

func (c *integrationConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	switch {
	case strings.Contains(query, "INSERT INTO trading_auctions"):
		return c.execAuctionInsert(args)
	case strings.Contains(query, "INSERT INTO trading_bids"):
		return c.execBidInsert(args)
	case strings.Contains(query, "DELETE FROM trading_auction_winners"):
		return c.execWinnersDelete(args)
	case strings.Contains(query, "INSERT INTO trading_auction_winners"):
		return c.execWinnersInsert(args)
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

func (c *integrationConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if len(args) != 1 {
		return nil, errors.New("unexpected query args length")
	}
	switch {
	case strings.Contains(query, "FROM trading_auctions"):
		record, ok := c.lookupAuction(args[0].Value.(string))
		if !ok {
			return &integrationRows{}, nil
		}
		return &integrationRows{
			values: [][]driver.Value{{
				record.auctionID,
				record.lotID,
				record.state,
				record.startsAt,
				record.endsAt,
				record.startPrice,
				record.currentPrice,
				record.minBidStep,
				record.leaderCompanyID,
			}},
		}, nil
	case strings.Contains(query, "FROM trading_bids"):
		rows := c.lookupBids(args[0].Value.(string))
		values := make([][]driver.Value, 0, len(rows))
		for _, row := range rows {
			values = append(values, []driver.Value{
				row.bidderCompanyID,
				row.amount,
				row.placedAt,
			})
		}
		return &integrationRows{values: values}, nil
	case strings.Contains(query, "FROM outbox_messages"):
		rows := c.lookupOutbox(toInt64(args[0].Value))
		values := make([][]driver.Value, 0, len(rows))
		for _, row := range rows {
			values = append(values, []driver.Value{
				row.id,
				row.eventType,
				row.sourceContext,
				row.payload,
				row.occurredAt,
				row.attempts,
			})
		}
		return &integrationRows{values: values}, nil
	default:
		return nil, errors.New("unsupported query")
	}
}

func (c *integrationConn) QueryRowContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return c.QueryContext(ctx, query, args)
}

func (c *integrationConn) execAuctionInsert(args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 9 {
		return nil, errors.New("unexpected auction args length")
	}
	record := integrationAuctionRecord{
		auctionID:       args[0].Value.(string),
		lotID:           args[1].Value.(string),
		state:           args[2].Value.(string),
		startsAt:        args[3].Value.(time.Time),
		endsAt:          args[4].Value.(time.Time),
		startPrice:      args[5].Value.(int64),
		currentPrice:    args[6].Value.(int64),
		minBidStep:      args[7].Value.(int64),
		leaderCompanyID: args[8].Value.(string),
	}
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	c.store.auctions[record.auctionID] = record
	return driver.RowsAffected(1), nil
}

func (c *integrationConn) execBidInsert(args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 4 {
		return nil, errors.New("unexpected bid args length")
	}
	record := integrationBidRecord{
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

func (c *integrationConn) execWinnersDelete(args []driver.NamedValue) (driver.Result, error) {
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

func (c *integrationConn) execWinnersInsert(args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 5 {
		return nil, errors.New("unexpected winners insert args length")
	}
	record := integrationWinnerRecord{
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

func (c *integrationConn) execOutboxInsert(args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 13 {
		return nil, errors.New("unexpected outbox args length")
	}
	payload, ok := args[4].Value.([]byte)
	if !ok {
		return nil, errors.New("payload must be []byte")
	}
	record := integrationOutboxRecord{
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

func (c *integrationConn) execOutboxUpdate(args []driver.NamedValue) (driver.Result, error) {
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

func (c *integrationConn) execOutboxFailure(args []driver.NamedValue) (driver.Result, error) {
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

func (c *integrationConn) lookupAuction(auctionID string) (integrationAuctionRecord, bool) {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	record, ok := c.store.auctions[auctionID]
	return record, ok
}

func (c *integrationConn) lookupBids(auctionID string) []integrationBidRecord {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	var out []integrationBidRecord
	for _, record := range c.store.bids {
		if record.auctionID == auctionID {
			out = append(out, record)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].amount != out[j].amount {
			return out[i].amount > out[j].amount
		}
		return out[i].placedAt.Before(out[j].placedAt)
	})
	return out
}

func (c *integrationConn) lookupOutbox(limit int64) []integrationOutboxRecord {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	var out []integrationOutboxRecord
	for _, record := range c.store.outbox {
		if !record.publishedAt.Valid && !record.failedAt.Valid {
			out = append(out, record)
		}
		if limit > 0 && int64(len(out)) >= limit {
			break
		}
	}
	return out
}

type integrationStmt struct{}

func (integrationStmt) Close() error  { return nil }
func (integrationStmt) NumInput() int { return -1 }
func (integrationStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("unsupported")
}
func (integrationStmt) Query([]driver.Value) (driver.Rows, error) {
	return nil, errors.New("unsupported")
}

type integrationTx struct{}

func (integrationTx) Commit() error   { return nil }
func (integrationTx) Rollback() error { return nil }

type integrationRows struct {
	values [][]driver.Value
	pos    int
}

func (r *integrationRows) Columns() []string {
	if len(r.values) == 0 {
		return []string{}
	}
	cols := make([]string, len(r.values[0]))
	for i := range cols {
		cols[i] = "col"
	}
	return cols
}
func (r *integrationRows) Close() error { return nil }
func (r *integrationRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.pos])
	r.pos++
	return nil
}

var (
	registerIntegrationDriverOnce sync.Once
	tradingTestDriver             = newIntegrationDriver()
)

func openIntegrationDB(t *testing.T, name string) *sql.DB {
	t.Helper()
	registerIntegrationDriverOnce.Do(func() {
		sql.Register("trading-test-driver", tradingTestDriver)
	})
	db, err := sql.Open("trading-test-driver", name)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
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
