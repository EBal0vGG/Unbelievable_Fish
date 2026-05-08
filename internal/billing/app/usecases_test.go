package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

// stubFakeProvider matches billing/payment/fake.Provider without importing that package (avoids import cycle in tests).
type stubFakeProvider struct{}

func (stubFakeProvider) CreateTopUp(ctx context.Context, req CreateTopUpRequest) (CreateTopUpResponse, error) {
	return CreateTopUpResponse{
		ProviderPaymentID: "fake-pay-" + req.TopUpID,
		ConfirmationURL:   "/billing/top-ups/" + req.TopUpID + "/fake-confirm",
	}, nil
}

const stubFakeProviderName = "fake"

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

func (m *memDepositRepo) ListByCompany(ctx context.Context, companyID string, limit int) ([]*wallet.AuctionDeposit, error) {
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

type memTopUpRepo struct {
	all []*wallet.TopUp
}

func (m *memTopUpRepo) Create(ctx context.Context, tu *wallet.TopUp) error {
	_ = ctx
	m.all = append(m.all, tu)
	return nil
}

func (m *memTopUpRepo) Save(ctx context.Context, tu *wallet.TopUp) error {
	_ = ctx
	for i, x := range m.all {
		if x.ID == tu.ID {
			m.all[i] = tu
			return nil
		}
	}
	return ErrTopUpNotFound
}

func (m *memTopUpRepo) Load(ctx context.Context, id string) (*wallet.TopUp, error) {
	_ = ctx
	for _, x := range m.all {
		if x.ID == id {
			return x, nil
		}
	}
	return nil, ErrTopUpNotFound
}

func (m *memTopUpRepo) LoadForUpdate(ctx context.Context, id string) (*wallet.TopUp, error) {
	return m.Load(ctx, id)
}

func (m *memTopUpRepo) LoadByProviderPayment(ctx context.Context, provider, providerPaymentID string) (*wallet.TopUp, error) {
	_ = ctx
	for _, x := range m.all {
		if x.Provider == provider && x.ProviderPaymentID == providerPaymentID {
			return x, nil
		}
	}
	return nil, ErrTopUpNotFound
}

func (m *memTopUpRepo) LoadByProviderPaymentForUpdate(ctx context.Context, provider, providerPaymentID string) (*wallet.TopUp, error) {
	return m.LoadByProviderPayment(ctx, provider, providerPaymentID)
}

func (m *memTopUpRepo) ListByCompany(ctx context.Context, companyID string, limit int) ([]*wallet.TopUp, error) {
	_ = ctx
	if limit <= 0 {
		limit = 100
	}
	var out []*wallet.TopUp
	for _, x := range m.all {
		if x.CompanyID == companyID {
			out = append(out, x)
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func TestCreateTopUpDoesNotChangeBalance(t *testing.T) {
	ar := &memAccountRepo{accounts: map[string]*wallet.Account{}}
	tr := &memTopUpRepo{}
	createAccount, _ := NewCreateAccount(ar, RandomHexID{}, nil)
	clk := fixedClock{t: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)}
	createTopUp, err := NewCreateTopUp(createAccount, ar, tr, stubFakeProvider{}, stubFakeProviderName, RandomHexID{}, clk, "http://localhost")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := createTopUp.Execute(ctx, "c-top", 5000, wallet.CurrencyRUB); err != nil {
		t.Fatal(err)
	}
	acc, err := ar.LoadByCompany(ctx, "c-top")
	if err != nil {
		t.Fatal(err)
	}
	if acc.Available() != 0 || acc.Total() != 0 {
		t.Fatalf("balance changed after create top-up: avail=%d total=%d", acc.Available(), acc.Total())
	}
}

func TestConfirmTopUpByProviderCreditsBalanceAndLedger(t *testing.T) {
	ar := &memAccountRepo{accounts: map[string]*wallet.Account{}}
	tr := &memTopUpRepo{}
	ledger := &memLedgerRepo{}
	pt := &memProcessedTopUp{ids: map[string]struct{}{}}
	clk := fixedClock{t: time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC)}
	createAccount, _ := NewCreateAccount(ar, RandomHexID{}, nil)
	createTopUp, _ := NewCreateTopUp(createAccount, ar, tr, stubFakeProvider{}, stubFakeProviderName, RandomHexID{}, clk, "http://localhost")
	confirmTopUp, _ := NewConfirmTopUp(ar, ledger, pt, RandomHexID{}, clk, nil)
	confirmByProv, _ := NewConfirmTopUpByProvider(tr, confirmTopUp, clk)

	ctx := context.Background()
	tu, err := createTopUp.Execute(ctx, "c1", 4200, wallet.CurrencyRUB)
	if err != nil {
		t.Fatal(err)
	}
	if err := confirmByProv.ExecuteByTopUpID(ctx, tu.ID); err != nil {
		t.Fatal(err)
	}
	acc := ar.accounts["c1"]
	if acc.Available() != 4200 {
		t.Fatalf("avail=%d", acc.Available())
	}
	if len(ledger.entries) != 1 || ledger.entries[0].EntryType != wallet.LedgerTopUpConfirmed {
		t.Fatalf("ledger=%+v", ledger.entries)
	}
	if tu.Status != wallet.TopUpSucceeded {
		t.Fatalf("top-up still %s", tu.Status)
	}
}

func TestConfirmTopUpByProviderDuplicateNoOpExtraLedger(t *testing.T) {
	ar := &memAccountRepo{accounts: map[string]*wallet.Account{}}
	tr := &memTopUpRepo{}
	ledger := &memLedgerRepo{}
	pt := &memProcessedTopUp{ids: map[string]struct{}{}}
	clk := fixedClock{t: time.Date(2025, 6, 3, 0, 0, 0, 0, time.UTC)}
	createAccount, _ := NewCreateAccount(ar, RandomHexID{}, nil)
	createTopUp, _ := NewCreateTopUp(createAccount, ar, tr, stubFakeProvider{}, stubFakeProviderName, RandomHexID{}, clk, "http://localhost")
	confirmTopUp, _ := NewConfirmTopUp(ar, ledger, pt, RandomHexID{}, clk, nil)
	confirmByProv, _ := NewConfirmTopUpByProvider(tr, confirmTopUp, clk)

	ctx := context.Background()
	tu, err := createTopUp.Execute(ctx, "c-dup", 1000, wallet.CurrencyRUB)
	if err != nil {
		t.Fatal(err)
	}
	if err := confirmByProv.ExecuteByTopUpID(ctx, tu.ID); err != nil {
		t.Fatal(err)
	}
	if err := confirmByProv.ExecuteByTopUpID(ctx, tu.ID); err != nil {
		t.Fatal(err)
	}
	if len(ledger.entries) != 1 {
		t.Fatalf("expected one ledger entry, got %d", len(ledger.entries))
	}
	if ar.accounts["c-dup"].Available() != 1000 {
		t.Fatalf("balance=%d", ar.accounts["c-dup"].Available())
	}
}

func TestConfirmTopUpByProviderAmountMismatchRejected(t *testing.T) {
	ar := &memAccountRepo{accounts: map[string]*wallet.Account{}}
	tr := &memTopUpRepo{}
	ledger := &memLedgerRepo{}
	pt := &memProcessedTopUp{ids: map[string]struct{}{}}
	clk := fixedClock{t: time.Date(2025, 6, 4, 0, 0, 0, 0, time.UTC)}
	createAccount, _ := NewCreateAccount(ar, RandomHexID{}, nil)
	createTopUp, _ := NewCreateTopUp(createAccount, ar, tr, stubFakeProvider{}, stubFakeProviderName, RandomHexID{}, clk, "http://localhost")
	confirmTopUp, _ := NewConfirmTopUp(ar, ledger, pt, RandomHexID{}, clk, nil)
	confirmByProv, _ := NewConfirmTopUpByProvider(tr, confirmTopUp, clk)

	ctx := context.Background()
	tu, err := createTopUp.Execute(ctx, "c-mis", 2000, wallet.CurrencyRUB)
	if err != nil {
		t.Fatal(err)
	}
	pid := tu.ProviderPaymentID
	err = confirmByProv.Execute(ctx, stubFakeProviderName, pid, 1999, wallet.CurrencyRUB)
	if err == nil || !errors.Is(err, ErrTopUpAmountMismatch) {
		t.Fatalf("expected ErrTopUpAmountMismatch, got %v", err)
	}
	if ar.accounts["c-mis"].Available() != 0 {
		t.Fatalf("balance changed: %d", ar.accounts["c-mis"].Available())
	}
}

func TestCreateAccountIdempotent(t *testing.T) {
	ar := &memAccountRepo{accounts: map[string]*wallet.Account{}}
	uc, _ := NewCreateAccount(ar, RandomHexID{}, nil)
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
	uc, _ := NewConfirmTopUp(ar, ledger, pt, RandomHexID{}, fixedClock{t: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}, nil)
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
	uc, _ := NewReserveAuctionDeposit(ar, dr, lr, RandomHexID{}, fixedClock{t: time.Now()}, nil)
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

	rel, _ := NewReleaseAuctionDeposit(ar, dr, lr, RandomHexID{}, clk, nil)
	if err := rel.Execute(context.Background(), "c1", "auc1", "r"); err != nil {
		t.Fatal(err)
	}
	if len(lr.entries) == 0 || lr.entries[0].Reason != "r" {
		t.Fatalf("expected ledger reason 'r', got %+v", lr.entries)
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
	cap, _ := NewCaptureAuctionDeposit(ar2, dr2, lr2, RandomHexID{}, clk, nil)
	if err := cap.Execute(context.Background(), "c2", "auc2", "penalty"); err != nil {
		t.Fatal(err)
	}
	if len(lr2.entries) == 0 || lr2.entries[0].Reason != "penalty" {
		t.Fatalf("expected ledger reason 'penalty', got %+v", lr2.entries)
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
	uc, _ := NewCapturePlatformFeeFromDeposit(ar, dr, lr, RandomHexID{}, clk, nil)
	// fee = 3% of 100_000 = 3000, deposit 5000 -> capture 3000, release remainder 2000
	if err := uc.Execute(context.Background(), "c1", "auc1", 100_000); err != nil {
		t.Fatal(err)
	}
	if acc.Held() != 0 || acc.Available() != 17_000 {
		t.Fatalf("avail=%d held=%d", acc.Available(), acc.Held())
	}
	if dep.Status != wallet.DepositSettled {
		t.Fatalf("expected settled deposit, got %s", dep.Status)
	}
	if err := uc.Execute(context.Background(), "c1", "auc1", 100_000); err != nil {
		t.Fatal(err)
	}
}

func TestCapturePlatformFeeDepositEqualsFee(t *testing.T) {
	clk := fixedClock{t: time.Date(2025, 3, 4, 0, 0, 0, 0, time.UTC)}
	ar := &memAccountRepo{accounts: map[string]*wallet.Account{}}
	acc, _ := wallet.NewAccount("a1", "c1", wallet.CurrencyRUB)
	_ = acc.Deposit(5_000)
	_ = acc.Reserve(3_000)
	ar.accounts["c1"] = acc
	dep, _ := wallet.NewAuctionDeposit("auc1", "c1", "a1", 3000, wallet.CurrencyRUB, clk.t)
	dr := &memDepositRepo{deposits: map[string]*wallet.AuctionDeposit{depKey("auc1", "c1"): dep}}
	lr := &memLedgerRepo{}
	uc, _ := NewCapturePlatformFeeFromDeposit(ar, dr, lr, RandomHexID{}, clk, nil)
	if err := uc.Execute(context.Background(), "c1", "auc1", 100_000); err != nil {
		t.Fatal(err)
	}
	if dep.Status != wallet.DepositCaptured {
		t.Fatalf("expected captured, got %s", dep.Status)
	}
	if acc.Available() != 2_000 || acc.Held() != 0 {
		t.Fatalf("avail=%d held=%d", acc.Available(), acc.Held())
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
	uc, _ := NewCapturePlatformFeeFromDeposit(ar, dr, lr, RandomHexID{}, clk, nil)
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

func TestReleaseAfterCaptureAndCaptureAfterRelease(t *testing.T) {
	clk := fixedClock{t: time.Date(2025, 3, 5, 0, 0, 0, 0, time.UTC)}

	ar := &memAccountRepo{accounts: map[string]*wallet.Account{}}
	acc, _ := wallet.NewAccount("a1", "c1", wallet.CurrencyRUB)
	_ = acc.Deposit(10_000)
	_ = acc.Reserve(2_000)
	ar.accounts["c1"] = acc
	dep, _ := wallet.NewAuctionDeposit("auc1", "c1", "a1", 2000, wallet.CurrencyRUB, clk.t)
	dr := &memDepositRepo{deposits: map[string]*wallet.AuctionDeposit{depKey("auc1", "c1"): dep}}
	lr := &memLedgerRepo{}
	capUC, _ := NewCaptureAuctionDeposit(ar, dr, lr, RandomHexID{}, clk, nil)
	relUC, _ := NewReleaseAuctionDeposit(ar, dr, lr, RandomHexID{}, clk, nil)
	if err := capUC.Execute(context.Background(), "c1", "auc1", "capture"); err != nil {
		t.Fatal(err)
	}
	if err := relUC.Execute(context.Background(), "c1", "auc1", "release"); err != ErrDepositNotHeld {
		t.Fatalf("expected ErrDepositNotHeld, got %v", err)
	}

	ar2 := &memAccountRepo{accounts: map[string]*wallet.Account{}}
	acc2, _ := wallet.NewAccount("a2", "c2", wallet.CurrencyRUB)
	_ = acc2.Deposit(10_000)
	_ = acc2.Reserve(2_000)
	ar2.accounts["c2"] = acc2
	dep2, _ := wallet.NewAuctionDeposit("auc2", "c2", "a2", 2000, wallet.CurrencyRUB, clk.t)
	dr2 := &memDepositRepo{deposits: map[string]*wallet.AuctionDeposit{depKey("auc2", "c2"): dep2}}
	lr2 := &memLedgerRepo{}
	relUC2, _ := NewReleaseAuctionDeposit(ar2, dr2, lr2, RandomHexID{}, clk, nil)
	capUC2, _ := NewCaptureAuctionDeposit(ar2, dr2, lr2, RandomHexID{}, clk, nil)
	if err := relUC2.Execute(context.Background(), "c2", "auc2", "release"); err != nil {
		t.Fatal(err)
	}
	if err := capUC2.Execute(context.Background(), "c2", "auc2", "capture"); err != ErrDepositNotHeld {
		t.Fatalf("expected ErrDepositNotHeld, got %v", err)
	}
}

type fixedClock struct {
	t time.Time
}

func (f fixedClock) Now() time.Time { return f.t }
