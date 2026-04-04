package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"sync"
	"testing"
)

type stubTxState struct {
	mu          sync.Mutex
	beginCalls  int
	commitCalls int
	rollbacks   int
}

type stubDriver struct {
	mu     sync.Mutex
	states map[string]*stubTxState
}

func newStubDriver() *stubDriver {
	return &stubDriver{states: make(map[string]*stubTxState)}
}

func (d *stubDriver) Open(name string) (driver.Conn, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	state, ok := d.states[name]
	if !ok {
		state = &stubTxState{}
		d.states[name] = state
	}

	return &stubConn{state: state}, nil
}

func (d *stubDriver) state(name string) *stubTxState {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.states[name]
}

type stubConn struct {
	state *stubTxState
}

func (c *stubConn) Prepare(query string) (driver.Stmt, error) { return stubStmt{}, nil }
func (c *stubConn) Close() error                              { return nil }
func (c *stubConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *stubConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	c.state.mu.Lock()
	c.state.beginCalls++
	c.state.mu.Unlock()

	return &stubTx{state: c.state}, nil
}

type stubTx struct {
	state *stubTxState
}

func (t *stubTx) Commit() error {
	t.state.mu.Lock()
	t.state.commitCalls++
	t.state.mu.Unlock()
	return nil
}

func (t *stubTx) Rollback() error {
	t.state.mu.Lock()
	t.state.rollbacks++
	t.state.mu.Unlock()
	return nil
}

type stubStmt struct{}

func (stubStmt) Close() error                               { return nil }
func (stubStmt) NumInput() int                              { return -1 }
func (stubStmt) Exec([]driver.Value) (driver.Result, error) { return driver.RowsAffected(0), nil }
func (stubStmt) Query([]driver.Value) (driver.Rows, error)  { return stubRows{}, nil }

type stubRows struct{}

func (stubRows) Columns() []string         { return []string{} }
func (stubRows) Close() error              { return nil }
func (stubRows) Next([]driver.Value) error { return io.EOF }

var (
	registerStubDriverOnce sync.Once
	testStubDriver         = newStubDriver()
)

func openStubDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()

	registerStubDriverOnce.Do(func() {
		sql.Register("catalog-postgres-stub", testStubDriver)
	})

	db, err := sql.Open("catalog-postgres-stub", dsn)
	if err != nil {
		t.Fatalf("unexpected open error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestTransactionManagerCommitsAndInjectsTxIntoContext(t *testing.T) {
	db := openStubDB(t, "commit-case")
	manager := NewTransactionManager(db, nil)

	err := manager.WithinTx(context.Background(), func(ctx context.Context) error {
		if _, ok := TxFromContext(ctx); !ok {
			t.Fatalf("expected tx in context")
		}

		dbtx := DBTXFromContext(ctx, db)
		if _, ok := dbtx.(*sql.Tx); !ok {
			t.Fatalf("expected DBTXFromContext to return *sql.Tx")
		}

		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	state := testStubDriver.state("commit-case")
	if state == nil {
		t.Fatalf("expected stub state")
	}
	if state.beginCalls != 1 {
		t.Fatalf("expected 1 begin call, got %d", state.beginCalls)
	}
	if state.commitCalls != 1 {
		t.Fatalf("expected 1 commit call, got %d", state.commitCalls)
	}
	if state.rollbacks != 0 {
		t.Fatalf("expected 0 rollbacks, got %d", state.rollbacks)
	}
}

func TestTransactionManagerRollsBackOnUseCaseError(t *testing.T) {
	db := openStubDB(t, "rollback-case")
	manager := NewTransactionManager(db, nil)
	useCaseErr := errors.New("use case failed")

	err := manager.WithinTx(context.Background(), func(ctx context.Context) error {
		if _, ok := TxFromContext(ctx); !ok {
			t.Fatalf("expected tx in context")
		}
		return useCaseErr
	})
	if !errors.Is(err, useCaseErr) {
		t.Fatalf("expected use case error, got %v", err)
	}

	state := testStubDriver.state("rollback-case")
	if state == nil {
		t.Fatalf("expected stub state")
	}
	if state.beginCalls != 1 {
		t.Fatalf("expected 1 begin call, got %d", state.beginCalls)
	}
	if state.commitCalls != 0 {
		t.Fatalf("expected 0 commit calls, got %d", state.commitCalls)
	}
	if state.rollbacks != 1 {
		t.Fatalf("expected 1 rollback, got %d", state.rollbacks)
	}
}
