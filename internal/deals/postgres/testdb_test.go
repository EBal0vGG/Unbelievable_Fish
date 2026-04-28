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

type integrationDealRecord struct {
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

type integrationProjectionRecord struct {
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

type integrationSelectionRecord struct {
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
	mu          sync.Mutex
	deals       map[string]integrationDealRecord
	projections map[string]integrationProjectionRecord
	selections  map[string]integrationSelectionRecord
	outbox      []integrationOutboxRecord
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
			deals:       make(map[string]integrationDealRecord),
			projections: make(map[string]integrationProjectionRecord),
			selections:  make(map[string]integrationSelectionRecord),
			outbox:      make([]integrationOutboxRecord, 0),
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

func (c *integrationConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if len(args) != 1 {
		return nil, errors.New("unexpected query args length")
	}
	arg := args[0].Value.(string)
	switch {
	case strings.Contains(query, "FROM deals") && strings.Contains(query, "WHERE deal_id"):
		record, ok := c.lookupDealByID(arg)
		if !ok {
			return &integrationRows{}, nil
		}
		return &integrationRows{values: [][]driver.Value{dealRow(record)}}, nil
	case strings.Contains(query, "FROM deals") && strings.Contains(query, "WHERE auction_id"):
		record, ok := c.lookupDealByAuctionID(arg)
		if !ok {
			return &integrationRows{}, nil
		}
		return &integrationRows{values: [][]driver.Value{dealRow(record)}}, nil
	case strings.Contains(query, "FROM deal_projections"):
		record, ok := c.lookupProjection(arg)
		if !ok {
			return &integrationRows{}, nil
		}
		return &integrationRows{values: [][]driver.Value{projectionRow(record)}}, nil
	case strings.Contains(query, "FROM deal_winner_selections"):
		record, ok := c.lookupSelection(arg)
		if !ok {
			return &integrationRows{}, nil
		}
		return &integrationRows{values: [][]driver.Value{selectionRow(record)}}, nil
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

func (c *integrationConn) execDealInsert(args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 28 {
		return nil, errors.New("unexpected deal args length")
	}
	record := integrationDealRecord{
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

func (c *integrationConn) execProjectionInsert(args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 15 {
		return nil, errors.New("unexpected projection args length")
	}
	record := integrationProjectionRecord{
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

func (c *integrationConn) execSelectionInsert(args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 18 {
		return nil, errors.New("unexpected selection args length")
	}
	record := integrationSelectionRecord{
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

func (c *integrationConn) lookupDealByID(id string) (integrationDealRecord, bool) {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	record, ok := c.store.deals[id]
	return record, ok
}

func (c *integrationConn) lookupDealByAuctionID(auctionID string) (integrationDealRecord, bool) {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	for _, record := range c.store.deals {
		if record.auctionID == auctionID {
			return record, true
		}
	}
	return integrationDealRecord{}, false
}

func (c *integrationConn) lookupProjection(auctionID string) (integrationProjectionRecord, bool) {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	record, ok := c.store.projections[auctionID]
	return record, ok
}

func (c *integrationConn) lookupSelection(auctionID string) (integrationSelectionRecord, bool) {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	record, ok := c.store.selections[auctionID]
	return record, ok
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

func dealRow(record integrationDealRecord) []driver.Value {
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

func projectionRow(record integrationProjectionRecord) []driver.Value {
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

func selectionRow(record integrationSelectionRecord) []driver.Value {
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

func toNullTime(value any) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	if t, ok := value.(time.Time); ok {
		return sql.NullTime{Time: t, Valid: true}
	}
	if nt, ok := value.(sql.NullTime); ok {
		return nt
	}
	return sql.NullTime{}
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

var (
	registerIntegrationDriverOnce sync.Once
	dealsTestDriver               = newIntegrationDriver()
)

func openIntegrationDB(t *testing.T, name string) *sql.DB {
	t.Helper()
	registerIntegrationDriverOnce.Do(func() {
		sql.Register("deals-test-driver", dealsTestDriver)
	})
	db, err := sql.Open("deals-test-driver", name)
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
