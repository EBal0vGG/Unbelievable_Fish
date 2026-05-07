package app

import (
	"context"
	"testing"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

type memAccountRepo struct {
	accounts map[string]*wallet.Account
}

func (m *memAccountRepo) Create(ctx context.Context, account *wallet.Account) error {
	m.accounts[account.CompanyID()] = account
	return nil
}

func (m *memAccountRepo) LoadByCompany(ctx context.Context, companyID string) (*wallet.Account, error) {
	a, ok := m.accounts[companyID]
	if !ok {
		return nil, ErrAccountNotFound
	}
	return a, nil
}

func (m *memAccountRepo) LoadByCompanyForUpdate(ctx context.Context, companyID string) (*wallet.Account, error) {
	return m.LoadByCompany(ctx, companyID)
}

func (m *memAccountRepo) Save(ctx context.Context, account *wallet.Account) error {
	m.accounts[account.CompanyID()] = account
	return nil
}

func (m *memAccountRepo) ExistsByCompany(ctx context.Context, companyID string) (bool, error) {
	_, ok := m.accounts[companyID]
	return ok, nil
}

type memDepositRepo struct {
	deposits map[string]*wallet.AuctionDeposit
}

func depKey(auctionID, companyID string) string { return auctionID + "|" + companyID }

func (m *memDepositRepo) Find(ctx context.Context, auctionID, companyID string) (*wallet.AuctionDeposit, error) {
	d, ok := m.deposits[depKey(auctionID, companyID)]
	if !ok {
		return nil, nil
	}
	return d, nil
}

func (m *memDepositRepo) Create(ctx context.Context, deposit *wallet.AuctionDeposit) error {
	m.deposits[depKey(deposit.AuctionID, deposit.CompanyID)] = deposit
	return nil
}

func (m *memDepositRepo) Save(ctx context.Context, deposit *wallet.AuctionDeposit) error {
	m.deposits[depKey(deposit.AuctionID, deposit.CompanyID)] = deposit
	return nil
}

func (m *memDepositRepo) ListByAuction(ctx context.Context, auctionID string) ([]*wallet.AuctionDeposit, error) {
	return nil, nil
}

type memLedgerRepo struct {
	entries []wallet.LedgerEntry
}

func (m *memLedgerRepo) Append(ctx context.Context, entry wallet.LedgerEntry) error {
	m.entries = append(m.entries, entry)
	return nil
}

func (m *memLedgerRepo) ExistsByReference(ctx context.Context, companyID, referenceType, referenceID string, typ wallet.LedgerEntryType) (bool, error) {
	for _, e := range m.entries {
		if e.CompanyID == companyID && e.ReferenceType == referenceType && e.ReferenceID == referenceID && e.EntryType == typ {
			return true, nil
		}
	}
	return false, nil
}

type memProcessedTopUp struct {
	ids map[string]struct{}
}

func (m *memProcessedTopUp) InsertIfNew(ctx context.Context, externalPaymentID, companyID, accountID string, amount int64) (bool, error) {
	if _, ok := m.ids[externalPaymentID]; ok {
		return false, nil
	}
	m.ids[externalPaymentID] = struct{}{}
	return true, nil
}

func TestCreateAccountIdempotent(t *testing.T) {
	ar := &memAccountRepo{accounts: map[string]*wallet.Account{}}
	uc, _ := NewCreateAccount(ar, RandomHexID{})
	if err := uc.Execute(context.Background(), "c1"); err != nil {
		t.Fatal(err)
	}
	if err := uc.Execute(context.Background(), "c1"); err != nil {
		t.Fatal(err)
	}
	if len(ar.accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(ar.accounts))
	}
}

func TestConfirmTopUpIdempotent(t *testing.T) {
	ar := &memAccountRepo{accounts: map[string]*wallet.Account{}}
	acc, _ := wallet.NewAccount("a1", "c1", wallet.CurrencyRUB)
	ar.accounts["c1"] = acc
	ledger := &memLedgerRepo{}
	pt := &memProcessedTopUp{ids: map[string]struct{}{}}
	uc, _ := NewConfirmTopUp(ar, ledger, pt, RandomHexID{}, fixedClock{t: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)})
	ext := "pay-1"
	if err := uc.Execute(context.Background(), "c1", 1000, ext); err != nil {
		t.Fatal(err)
	}
	if err := uc.Execute(context.Background(), "c1", 1000, ext); err != nil {
		t.Fatal(err)
	}
	if ar.accounts["c1"].Available() != 1000 {
		t.Fatalf("balance=%d", ar.accounts["c1"].Available())
	}
}

func TestReserveAuctionDepositTwiceNoOp(t *testing.T) {
	ar := &memAccountRepo{accounts: map[string]*wallet.Account{}}
	acc, _ := wallet.NewAccount("a1", "c1", wallet.CurrencyRUB)
	_ = acc.Deposit(100_000)
	ar.accounts["c1"] = acc
	dr := &memDepositRepo{deposits: map[string]*wallet.AuctionDeposit{}}
	lr := &memLedgerRepo{}
	uc, _ := NewReserveAuctionDeposit(ar, dr, lr, RandomHexID{}, fixedClock{t: time.Now()})
	if err := uc.Execute(context.Background(), "c1", "auc1", 100_000, wallet.CurrencyRUB); err != nil {
		t.Fatal(err)
	}
	if err := uc.Execute(context.Background(), "c1", "auc1", 100_000, wallet.CurrencyRUB); err != nil {
		t.Fatal(err)
	}
	if acc.Available() != 100_000-5_000 || acc.Held() != 5_000 {
		t.Fatalf("avail=%d held=%d", acc.Available(), acc.Held())
	}
}

func TestReleaseCaptureDepositIdempotent(t *testing.T) {
	clk := fixedClock{t: time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC)}

	ar := &memAccountRepo{accounts: map[string]*wallet.Account{}}
	acc, _ := wallet.NewAccount("a1", "c1", wallet.CurrencyRUB)
	_ = acc.Deposit(10_000)
	_ = acc.Reserve(2_000)
	ar.accounts["c1"] = acc
	dep, _ := wallet.NewAuctionDeposit("auc1", "c1", "a1", 2000, wallet.CurrencyRUB, clk.t)
	dr := &memDepositRepo{deposits: map[string]*wallet.AuctionDeposit{depKey("auc1", "c1"): dep}}
	lr := &memLedgerRepo{}

	rel, _ := NewReleaseAuctionDeposit(ar, dr, lr, RandomHexID{}, clk)
	if err := rel.Execute(context.Background(), "c1", "auc1", "r"); err != nil {
		t.Fatal(err)
	}
	if acc.Available() != 10_000 || acc.Held() != 0 {
		t.Fatalf("after release: avail=%d held=%d", acc.Available(), acc.Held())
	}
	if err := rel.Execute(context.Background(), "c1", "auc1", "r"); err != nil {
		t.Fatal(err)
	}

	ar2 := &memAccountRepo{accounts: map[string]*wallet.Account{}}
	acc2, _ := wallet.NewAccount("a2", "c2", wallet.CurrencyRUB)
	_ = acc2.Deposit(10_000)
	_ = acc2.Reserve(3000)
	ar2.accounts["c2"] = acc2
	dep2, _ := wallet.NewAuctionDeposit("auc2", "c2", "a2", 3000, wallet.CurrencyRUB, clk.t)
	dr2 := &memDepositRepo{deposits: map[string]*wallet.AuctionDeposit{depKey("auc2", "c2"): dep2}}
	lr2 := &memLedgerRepo{}
	cap, _ := NewCaptureAuctionDeposit(ar2, dr2, lr2, RandomHexID{}, clk)
	if err := cap.Execute(context.Background(), "c2", "auc2", "penalty"); err != nil {
		t.Fatal(err)
	}
	if acc2.Available() != 7_000 || acc2.Held() != 0 {
		t.Fatalf("after capture: avail=%d held=%d", acc2.Available(), acc2.Held())
	}
	if err := cap.Execute(context.Background(), "c2", "auc2", "penalty"); err != nil {
		t.Fatal(err)
	}
}

func TestCapturePlatformFeeFromDeposit(t *testing.T) {
	clk := fixedClock{t: time.Date(2025, 3, 2, 0, 0, 0, 0, time.UTC)}
	ar := &memAccountRepo{accounts: map[string]*wallet.Account{}}
	acc, _ := wallet.NewAccount("a1", "c1", wallet.CurrencyRUB)
	_ = acc.Deposit(20_000)
	_ = acc.Reserve(5_000)
	ar.accounts["c1"] = acc
	dep, _ := wallet.NewAuctionDeposit("auc1", "c1", "a1", 5000, wallet.CurrencyRUB, clk.t)
	dr := &memDepositRepo{deposits: map[string]*wallet.AuctionDeposit{depKey("auc1", "c1"): dep}}
	lr := &memLedgerRepo{}
	uc, _ := NewCapturePlatformFeeFromDeposit(ar, dr, lr, RandomHexID{}, clk)
	// fee = 3% of 100_000 = 3000, deposit 5000 -> capture 3000, release remainder 2000
	if err := uc.Execute(context.Background(), "c1", "auc1", 100_000); err != nil {
		t.Fatal(err)
	}
	if acc.Held() != 0 || acc.Available() != 17_000 {
		t.Fatalf("avail=%d held=%d", acc.Available(), acc.Held())
	}
	if err := uc.Execute(context.Background(), "c1", "auc1", 100_000); err != nil {
		t.Fatal(err)
	}
}

func TestCapturePlatformFeeDepositLessThanFeeRecordsDue(t *testing.T) {
	clk := fixedClock{t: time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC)}
	ar := &memAccountRepo{accounts: map[string]*wallet.Account{}}
	acc, _ := wallet.NewAccount("a1", "c1", wallet.CurrencyRUB)
	_ = acc.Deposit(10_000)
	_ = acc.Reserve(2_000)
	ar.accounts["c1"] = acc
	dep, _ := wallet.NewAuctionDeposit("auc1", "c1", "a1", 2000, wallet.CurrencyRUB, clk.t)
	dr := &memDepositRepo{deposits: map[string]*wallet.AuctionDeposit{depKey("auc1", "c1"): dep}}
	lr := &memLedgerRepo{}
	uc, _ := NewCapturePlatformFeeFromDeposit(ar, dr, lr, RandomHexID{}, clk)
	// fee 3% of 100_000 = 3000 > deposit 2000 -> full capture + PLATFORM_FEE_DUE 1000
	if err := uc.Execute(context.Background(), "c1", "auc1", 100_000); err != nil {
		t.Fatal(err)
	}
	if acc.Available() != 8_000 || acc.Held() != 0 {
		t.Fatalf("avail=%d held=%d", acc.Available(), acc.Held())
	}
	if dep.Status != wallet.DepositCaptured {
		t.Fatalf("deposit status=%s", dep.Status)
	}
	var dueFound bool
	for _, e := range lr.entries {
		if e.EntryType == wallet.LedgerPlatformFeeDue {
			dueFound = true
			if e.Amount != 1000 {
				t.Fatalf("due amount=%d", e.Amount)
			}
		}
	}
	if !dueFound {
		t.Fatal("expected PLATFORM_FEE_DUE ledger entry")
	}
}

type fixedClock struct {
	t time.Time
}

func (f fixedClock) Now() time.Time { return f.t }
