package postgres

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/payment/fake"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/dbconfig"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/integrationtest"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestConfirmTopUpAtomicAndIdempotent_RealPG(t *testing.T) {
	db, ok := openRealPostgres(t)
	if !ok {
		return
	}
	if err := applyMigrations(t, db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := truncateBillingTables(t, db); err != nil {
		t.Fatalf("truncate billing tables: %v", err)
	}

	ctx := context.Background()
	txm := NewTransactionManager(db, nil)
	accounts := NewAccountRepository(db)
	ledger := NewLedgerRepository(db)
	processed := NewProcessedTopUpRepository(db)
	createAccount, err := billingapp.NewCreateAccount(accounts, billingapp.RandomHexID{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	confirmTopUp, err := billingapp.NewConfirmTopUp(accounts, ledger, processed, billingapp.RandomHexID{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	companyID := "co-topup-1"
	if err := txm.WithinTx(ctx, func(txCtx context.Context) error {
		return createAccount.Execute(txCtx, companyID)
	}); err != nil {
		t.Fatalf("create account: %v", err)
	}

	externalPaymentID := "external-pay-1"
	if err := txm.WithinTx(ctx, func(txCtx context.Context) error {
		return confirmTopUp.Execute(txCtx, companyID, 2000, externalPaymentID)
	}); err != nil {
		t.Fatalf("confirm topup first: %v", err)
	}
	if err := txm.WithinTx(ctx, func(txCtx context.Context) error {
		return confirmTopUp.Execute(txCtx, companyID, 2000, externalPaymentID)
	}); err != nil {
		t.Fatalf("confirm topup duplicate: %v", err)
	}

	acc, err := accounts.LoadByCompany(ctx, companyID)
	if err != nil {
		t.Fatalf("load account: %v", err)
	}
	if acc.Available() != 2000 || acc.Held() != 0 {
		t.Fatalf("unexpected account amounts: available=%d held=%d", acc.Available(), acc.Held())
	}

	var ledgerCount int
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM billing_ledger_entries
WHERE company_id = $1 AND reference_type = 'external_payment' AND reference_id = $2 AND type = $3
`, companyID, externalPaymentID, string(wallet.LedgerTopUpConfirmed)).Scan(&ledgerCount); err != nil {
		t.Fatalf("count ledger: %v", err)
	}
	if ledgerCount != 1 {
		t.Fatalf("expected one topup ledger entry, got %d", ledgerCount)
	}

	var processedCount int
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM billing_processed_top_ups WHERE external_payment_id = $1
`, externalPaymentID).Scan(&processedCount); err != nil {
		t.Fatalf("count processed topups: %v", err)
	}
	if processedCount != 1 {
		t.Fatalf("expected one processed topup row, got %d", processedCount)
	}
}

func TestTopUpCreateAndFakeConfirmFlow_RealPG(t *testing.T) {
	db, ok := openRealPostgres(t)
	if !ok {
		return
	}
	if err := applyMigrations(t, db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := truncateBillingTables(t, db); err != nil {
		t.Fatalf("truncate billing tables: %v", err)
	}

	ctx := context.Background()
	txm := NewTransactionManager(db, nil)
	accounts := NewAccountRepository(db)
	ledger := NewLedgerRepository(db)
	processed := NewProcessedTopUpRepository(db)
	topUps := NewTopUpRepository(db)
	createAccount, err := billingapp.NewCreateAccount(accounts, billingapp.RandomHexID{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	confirmTopUp, err := billingapp.NewConfirmTopUp(accounts, ledger, processed, billingapp.RandomHexID{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	createTopUpUC, err := billingapp.NewCreateTopUp(
		createAccount,
		accounts,
		topUps,
		fake.Provider{},
		fake.ProviderName,
		billingapp.RandomHexID{},
		nil,
		"http://localhost",
	)
	if err != nil {
		t.Fatal(err)
	}
	confirmByProv, err := billingapp.NewConfirmTopUpByProvider(topUps, confirmTopUp, nil)
	if err != nil {
		t.Fatal(err)
	}

	companyID := "co-topup-flow-1"
	if err := txm.WithinTx(ctx, func(txCtx context.Context) error {
		return createAccount.Execute(txCtx, companyID)
	}); err != nil {
		t.Fatalf("create account: %v", err)
	}

	var tu *wallet.TopUp
	if err := txm.WithinTx(ctx, func(txCtx context.Context) error {
		var err error
		tu, err = createTopUpUC.Execute(txCtx, companyID, 7777, wallet.CurrencyRUB)
		return err
	}); err != nil {
		t.Fatalf("create top-up: %v", err)
	}
	if tu.Status != wallet.TopUpPending {
		t.Fatalf("expected PENDING, got %s", tu.Status)
	}

	accBefore, err := accounts.LoadByCompany(ctx, companyID)
	if err != nil {
		t.Fatal(err)
	}
	if accBefore.Available() != 0 {
		t.Fatalf("expected zero balance before confirm, got %d", accBefore.Available())
	}

	if err := txm.WithinTx(ctx, func(txCtx context.Context) error {
		return confirmByProv.ExecuteByTopUpID(txCtx, tu.ID)
	}); err != nil {
		t.Fatalf("confirm top-up: %v", err)
	}

	accAfter, err := accounts.LoadByCompany(ctx, companyID)
	if err != nil {
		t.Fatal(err)
	}
	if accAfter.Available() != 7777 {
		t.Fatalf("expected balance 7777, got %d", accAfter.Available())
	}

	var topUpStatus string
	if err := db.QueryRowContext(ctx, `SELECT status FROM billing_top_ups WHERE id = $1`, tu.ID).Scan(&topUpStatus); err != nil {
		t.Fatalf("load top-up row: %v", err)
	}
	if topUpStatus != string(wallet.TopUpSucceeded) {
		t.Fatalf("top-up status want SUCCEEDED, got %s", topUpStatus)
	}

	var ledgerCount int
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM billing_ledger_entries
WHERE company_id = $1 AND reference_id = $2 AND type = $3
`, companyID, tu.ProviderPaymentID, string(wallet.LedgerTopUpConfirmed)).Scan(&ledgerCount); err != nil {
		t.Fatalf("count ledger: %v", err)
	}
	if ledgerCount != 1 {
		t.Fatalf("expected one ledger entry, got %d", ledgerCount)
	}

	var processedCount int
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM billing_processed_top_ups WHERE external_payment_id = $1
`, tu.ProviderPaymentID).Scan(&processedCount); err != nil {
		t.Fatalf("count processed: %v", err)
	}
	if processedCount != 1 {
		t.Fatalf("expected one processed row, got %d", processedCount)
	}

	if err := txm.WithinTx(ctx, func(txCtx context.Context) error {
		return confirmByProv.ExecuteByTopUpID(txCtx, tu.ID)
	}); err != nil {
		t.Fatalf("duplicate confirm: %v", err)
	}
	accDup, err := accounts.LoadByCompany(ctx, companyID)
	if err != nil {
		t.Fatal(err)
	}
	if accDup.Available() != 7777 {
		t.Fatalf("balance changed on duplicate confirm: %d", accDup.Available())
	}
}

func TestReserveDepositDuplicateAndConcurrent_RealPG(t *testing.T) {
	db, ok := openRealPostgres(t)
	if !ok {
		return
	}
	if err := applyMigrations(t, db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := truncateBillingTables(t, db); err != nil {
		t.Fatalf("truncate billing tables: %v", err)
	}

	ctx := context.Background()
	txm := NewTransactionManager(db, nil)
	accounts := NewAccountRepository(db)
	deposits := NewAuctionDepositRepository(db)
	ledger := NewLedgerRepository(db)
	processed := NewProcessedTopUpRepository(db)
	createAccount, err := billingapp.NewCreateAccount(accounts, billingapp.RandomHexID{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	confirmTopUp, err := billingapp.NewConfirmTopUp(accounts, ledger, processed, billingapp.RandomHexID{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	reserve, err := billingapp.NewReserveAuctionDeposit(accounts, deposits, ledger, billingapp.RandomHexID{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	companyID := "co-reserve-1"
	if err := txm.WithinTx(ctx, func(txCtx context.Context) error {
		return createAccount.Execute(txCtx, companyID)
	}); err != nil {
		t.Fatalf("create account: %v", err)
	}
	if err := txm.WithinTx(ctx, func(txCtx context.Context) error {
		return confirmTopUp.Execute(txCtx, companyID, 100000, "ext-reserve-topup")
	}); err != nil {
		t.Fatalf("topup before reserve: %v", err)
	}

	// duplicate reserve for same auction must be no-op on second call.
	if err := txm.WithinTx(ctx, func(txCtx context.Context) error {
		return reserve.Execute(txCtx, companyID, "auc-dup", 100000, wallet.CurrencyRUB)
	}); err != nil {
		t.Fatalf("reserve first: %v", err)
	}
	if err := txm.WithinTx(ctx, func(txCtx context.Context) error {
		return reserve.Execute(txCtx, companyID, "auc-dup", 100000, wallet.CurrencyRUB)
	}); err != nil {
		t.Fatalf("reserve duplicate: %v", err)
	}

	var depCount int
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM billing_auction_deposits WHERE company_id = $1 AND auction_id = $2
`, companyID, "auc-dup").Scan(&depCount); err != nil {
		t.Fatalf("count deposits: %v", err)
	}
	if depCount != 1 {
		t.Fatalf("expected one deposit row, got %d", depCount)
	}

	var reserveLedgerCount int
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM billing_ledger_entries
WHERE company_id = $1 AND type = $2 AND reference_type = 'auction_deposit'
`, companyID, string(wallet.LedgerBidDepositReserved)).Scan(&reserveLedgerCount); err != nil {
		t.Fatalf("count reserve ledger: %v", err)
	}
	if reserveLedgerCount != 1 {
		t.Fatalf("expected one reserve ledger entry, got %d", reserveLedgerCount)
	}

	// concurrent reserve for different auctions with exactly one deposit budget.
	companyConcurrent := "co-reserve-concurrent"
	if err := txm.WithinTx(ctx, func(txCtx context.Context) error {
		return createAccount.Execute(txCtx, companyConcurrent)
	}); err != nil {
		t.Fatalf("create concurrent account: %v", err)
	}
	if err := txm.WithinTx(ctx, func(txCtx context.Context) error {
		return confirmTopUp.Execute(txCtx, companyConcurrent, 5000, "ext-concurrent-topup")
	}); err != nil {
		t.Fatalf("topup concurrent account: %v", err)
	}

	errs := make([]error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		errs[0] = txm.WithinTx(ctx, func(txCtx context.Context) error {
			return reserve.Execute(txCtx, companyConcurrent, "auc-c-1", 100000, wallet.CurrencyRUB)
		})
	}()
	go func() {
		defer wg.Done()
		errs[1] = txm.WithinTx(ctx, func(txCtx context.Context) error {
			return reserve.Execute(txCtx, companyConcurrent, "auc-c-2", 100000, wallet.CurrencyRUB)
		})
	}()
	wg.Wait()

	successes := 0
	failures := 0
	for _, err := range errs {
		if err == nil {
			successes++
			continue
		}
		if err == billingapp.ErrInsufficientFundsForDeposit {
			failures++
			continue
		}
		t.Fatalf("unexpected reserve error: %v", err)
	}
	if successes != 1 || failures != 1 {
		t.Fatalf("expected exactly one success and one insufficient error, got success=%d failure=%d", successes, failures)
	}

	accConcurrent, err := accounts.LoadByCompany(ctx, companyConcurrent)
	if err != nil {
		t.Fatalf("load concurrent account: %v", err)
	}
	if accConcurrent.Available() != 0 || accConcurrent.Held() != 5000 {
		t.Fatalf("unexpected concurrent account amounts: available=%d held=%d", accConcurrent.Available(), accConcurrent.Held())
	}
}

func TestDealInvoiceLister_ListExpired_SkipLocked_RealPG(t *testing.T) {
	db, ok := openRealPostgres(t)
	if !ok {
		return
	}
	if err := applyMigrations(t, db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := truncateBillingTables(t, db); err != nil {
		t.Fatalf("truncate billing tables: %v", err)
	}

	ctx := context.Background()
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	duePast := now.Add(-time.Hour)
	dueFuture := now.Add(time.Hour)

	_, err := db.ExecContext(ctx, `
INSERT INTO billing_deal_invoices (
  id, deal_id, auction_id, buyer_company_id, seller_company_id,
  goods_amount, platform_fee_due_amount, total_amount, currency, status,
  provider, provider_invoice_id, payment_url, due_at, created_at
) VALUES
  ('inv-exp-1', 'deal-1', 'a1', 'b1', 's1', 100, 10, 110, 'RUB', 'PAYMENT_PENDING',
   'fake', 'p1', 'http://x', $1, $3),
  ('inv-exp-2', 'deal-2', 'a1', 'b2', 's1', 100, 10, 110, 'RUB', 'PAYMENT_PENDING',
   'fake', 'p2', 'http://x', $2, $3),
  ('inv-future', 'deal-3', 'a1', 'b3', 's1', 100, 10, 110, 'RUB', 'PAYMENT_PENDING',
   'fake', 'p3', 'http://x', $4, $3),
  ('inv-paid', 'deal-4', 'a1', 'b4', 's1', 100, 10, 110, 'RUB', 'PAID',
   'fake', 'p4', 'http://x', $1, $3)
`, duePast, duePast, now, dueFuture)
	if err != nil {
		t.Fatalf("insert invoices: %v", err)
	}

	lister := NewDealInvoiceLister(db)
	txm := NewTransactionManager(db, nil)
	var ids []string
	if err := txm.WithinTx(ctx, func(txCtx context.Context) error {
		var err error
		ids, err = lister.ListExpired(txCtx, now, 10)
		return err
	}); err != nil {
		t.Fatalf("list expired: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 expired ids, got %v", ids)
	}
	if ids[0] != "inv-exp-1" || ids[1] != "inv-exp-2" {
		t.Fatalf("order: want [inv-exp-1 inv-exp-2], got %v", ids)
	}
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
		t.Skipf("postgres not reachable (%v); start: docker compose up -d postgres", err)
		return nil, false
	}
	t.Cleanup(func() { _ = db.Close() })
	integrationtest.AcquireSharedPostgresAdvisoryLock(t, db)
	return db, true
}

func applyMigrations(t *testing.T, db *sql.DB) error {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("cannot resolve caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
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

func truncateBillingTables(t *testing.T, db *sql.DB) error {
	t.Helper()
	_, err := db.Exec(`
TRUNCATE TABLE
    billing_processed_top_ups,
    billing_ledger_entries,
    billing_auction_deposits,
    billing_seller_payouts,
    billing_deal_invoices,
    billing_top_ups,
    billing_accounts
`)
	return err
}
