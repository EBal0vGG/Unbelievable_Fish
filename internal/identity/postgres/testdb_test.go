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

type integrationCompanyRecord struct {
	companyID string
	name      string
	inn       string
	ogrn      string
	status    string
	createdAt time.Time
}

type integrationUserRecord struct {
	userID          string
	companyID       sql.NullString
	name            string
	role            string
	login           string
	passwordHash    string
	termsAcceptedAt sql.NullTime
	termsVersion    sql.NullString
}

type integrationStore struct {
	mu        sync.Mutex
	companies map[string]integrationCompanyRecord
	users     map[string]integrationUserRecord
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
			companies: make(map[string]integrationCompanyRecord),
			users:     make(map[string]integrationUserRecord),
		}
		d.stores[name] = store
	}

	return &integrationConn{store: store}, nil
}

type integrationConn struct {
	store *integrationStore
}

func (c *integrationConn) Prepare(query string) (driver.Stmt, error) { return integrationStmt{}, nil }
func (c *integrationConn) Close() error                              { return nil }
func (c *integrationConn) Begin() (driver.Tx, error)                 { return integrationTx{}, nil }

func (c *integrationConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	switch {
	case strings.Contains(query, "INSERT INTO identity_companies"):
		return c.execCompanyInsert(args)
	case strings.Contains(query, "INSERT INTO identity_users"):
		return c.execUserInsert(args)
	case strings.Contains(query, "INSERT INTO outbox_messages"):
		return driver.RowsAffected(1), nil
	default:
		return nil, errors.New("unsupported exec query")
	}
}

func (c *integrationConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if len(args) != 1 && len(args) != 2 {
		return nil, errors.New("unexpected query args length")
	}

	switch {
	case strings.Contains(query, "FROM identity_companies") && strings.Contains(query, "SELECT 1"):
		companyID := args[0].Value.(string)
		c.store.mu.Lock()
		defer c.store.mu.Unlock()
		if _, ok := c.store.companies[companyID]; !ok {
			return &integrationRows{}, nil
		}
		return &integrationRows{values: [][]driver.Value{{1}}}, nil
	case strings.Contains(query, "FROM identity_companies") && strings.Contains(query, "WHERE inn = $1 AND ogrn = $2"):
		inn := args[0].Value.(string)
		ogrn := args[1].Value.(string)
		c.store.mu.Lock()
		defer c.store.mu.Unlock()
		for _, record := range c.store.companies {
			if record.inn == inn && record.ogrn == ogrn {
				return &integrationRows{values: [][]driver.Value{{
					record.companyID,
					record.name,
					record.inn,
					record.ogrn,
					record.status,
					record.createdAt,
				}}}, nil
			}
		}
		return &integrationRows{}, nil
	case strings.Contains(query, "FROM identity_companies"):
		companyID := args[0].Value.(string)
		c.store.mu.Lock()
		defer c.store.mu.Unlock()
		record, ok := c.store.companies[companyID]
		if !ok {
			return &integrationRows{}, nil
		}
		return &integrationRows{values: [][]driver.Value{{
			record.companyID,
			record.name,
			record.inn,
			record.ogrn,
			record.status,
			record.createdAt,
		}}}, nil
	case strings.Contains(query, "FROM identity_users") && strings.Contains(query, "SELECT 1"):
		login := args[0].Value.(string)
		c.store.mu.Lock()
		defer c.store.mu.Unlock()
		for _, record := range c.store.users {
			if record.login == login {
				return &integrationRows{values: [][]driver.Value{{1}}}, nil
			}
		}
		return &integrationRows{}, nil
	case strings.Contains(query, "FROM identity_users") && strings.Contains(query, "WHERE user_id = $1"):
		userID := args[0].Value.(string)
		c.store.mu.Lock()
		defer c.store.mu.Unlock()
		record, ok := c.store.users[userID]
		if !ok {
			return &integrationRows{}, nil
		}
		return &integrationRows{values: [][]driver.Value{{
			record.userID,
			nullStringValue(record.companyID),
			record.name,
			record.role,
			record.login,
			record.passwordHash,
			nullTimeValue(record.termsAcceptedAt),
			nullStringValue(record.termsVersion),
		}}}, nil
	case strings.Contains(query, "FROM identity_users") && strings.Contains(query, "WHERE login = $1"):
		login := args[0].Value.(string)
		c.store.mu.Lock()
		defer c.store.mu.Unlock()
		for _, record := range c.store.users {
			if record.login == login {
				return &integrationRows{values: [][]driver.Value{{
					record.userID,
					nullStringValue(record.companyID),
					record.name,
					record.role,
					record.login,
					record.passwordHash,
					nullTimeValue(record.termsAcceptedAt),
					nullStringValue(record.termsVersion),
				}}}, nil
			}
		}
		return &integrationRows{}, nil
	default:
		return nil, errors.New("unsupported query")
	}
}

func (c *integrationConn) execCompanyInsert(args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 6 {
		return nil, errors.New("unexpected company args length")
	}

	record := integrationCompanyRecord{
		companyID: args[0].Value.(string),
		name:      args[1].Value.(string),
		inn:       args[2].Value.(string),
		ogrn:      args[3].Value.(string),
		status:    args[4].Value.(string),
		createdAt: args[5].Value.(time.Time),
	}

	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	c.store.companies[record.companyID] = record
	return driver.RowsAffected(1), nil
}

func (c *integrationConn) execUserInsert(args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 8 {
		return nil, errors.New("unexpected user args length")
	}

	record := integrationUserRecord{
		userID:          args[0].Value.(string),
		companyID:       namedValueString(args[1]),
		name:            args[2].Value.(string),
		role:            args[3].Value.(string),
		login:           args[4].Value.(string),
		passwordHash:    args[5].Value.(string),
		termsAcceptedAt: namedValueTime(args[6]),
		termsVersion:    namedValueString(args[7]),
	}

	c.store.mu.Lock()
	defer c.store.mu.Unlock()
	c.store.users[record.userID] = record
	return driver.RowsAffected(1), nil
}

type integrationTx struct{}

func (integrationTx) Commit() error   { return nil }
func (integrationTx) Rollback() error { return nil }

type integrationStmt struct{}

func (integrationStmt) Close() error  { return nil }
func (integrationStmt) NumInput() int { return -1 }
func (integrationStmt) Exec(args []driver.Value) (driver.Result, error) {
	return nil, errors.New("not supported")
}
func (integrationStmt) Query(args []driver.Value) (driver.Rows, error) {
	return nil, errors.New("not supported")
}

type integrationRows struct {
	values [][]driver.Value
	index  int
}

func (r *integrationRows) Columns() []string {
	if len(r.values) == 0 {
		return []string{}
	}
	cols := make([]string, len(r.values[0]))
	for i := range cols {
		cols[i] = "c"
	}
	return cols
}

func (r *integrationRows) Close() error { return nil }

func (r *integrationRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}

var (
	registerDriverOnce sync.Once
	driverName         = "identity-postgres-test"
	testDriver         = newIntegrationDriver()
)

func openIntegrationDB(t *testing.T, name string) *sql.DB {
	t.Helper()

	registerDriverOnce.Do(func() {
		sql.Register(driverName, testDriver)
	})

	db, err := sql.Open(driverName, name)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func namedValueTime(value driver.NamedValue) sql.NullTime {
	if value.Value == nil {
		return sql.NullTime{}
	}
	tm, ok := value.Value.(time.Time)
	if !ok {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: tm, Valid: true}
}

func namedValueString(value driver.NamedValue) sql.NullString {
	if value.Value == nil {
		return sql.NullString{}
	}
	text, ok := value.Value.(string)
	if !ok {
		return sql.NullString{}
	}
	return sql.NullString{String: text, Valid: true}
}

func nullTimeValue(value sql.NullTime) driver.Value {
	if !value.Valid {
		return nil
	}
	return value.Time
}

func nullStringValue(value sql.NullString) driver.Value {
	if !value.Valid {
		return nil
	}
	return value.String
}
