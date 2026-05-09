package app

import (
	"context"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

type AccountRepository interface {
	Create(ctx context.Context, account *wallet.Account) error
	LoadByCompany(ctx context.Context, companyID string) (*wallet.Account, error)
	LoadByCompanyForUpdate(ctx context.Context, companyID string) (*wallet.Account, error)
	Save(ctx context.Context, account *wallet.Account) error
	ExistsByCompany(ctx context.Context, companyID string) (bool, error)
}

type AuctionDepositRepository interface {
	Find(ctx context.Context, auctionID, companyID string) (*wallet.AuctionDeposit, error)
	Create(ctx context.Context, deposit *wallet.AuctionDeposit) error
	Save(ctx context.Context, deposit *wallet.AuctionDeposit) error
	ListByAuction(ctx context.Context, auctionID string) ([]*wallet.AuctionDeposit, error)
	ListByCompany(ctx context.Context, companyID string, limit int) ([]*wallet.AuctionDeposit, error)
}

type LedgerRepository interface {
	Append(ctx context.Context, entry wallet.LedgerEntry) error
	ExistsByReference(ctx context.Context, companyID, referenceType, referenceID string, typ wallet.LedgerEntryType) (bool, error)
}

// LedgerQuery read-side for HTTP (implemented by postgres.LedgerLister).
type LedgerQuery interface {
	ListByCompany(ctx context.Context, companyID string, limit int) ([]wallet.LedgerEntry, error)
}

type DepositQuery interface {
	ListByCompany(ctx context.Context, companyID string, limit int) ([]*wallet.AuctionDeposit, error)
}

type ProcessedTopUpRepository interface {
	InsertIfNew(ctx context.Context, externalPaymentID, companyID, accountID string, amount int64) (inserted bool, err error)
}

// DealInvoiceRepository persists full-payment invoices for deals (Stage 9+).
type DealInvoiceRepository interface {
	Create(ctx context.Context, inv *wallet.DealInvoice) error
	Save(ctx context.Context, inv *wallet.DealInvoice) error
	LoadByDealID(ctx context.Context, dealID string) (*wallet.DealInvoice, error)
	LoadByDealIDForUpdate(ctx context.Context, dealID string) (*wallet.DealInvoice, error)
	LoadByID(ctx context.Context, id string) (*wallet.DealInvoice, error)
	LoadByIDForUpdate(ctx context.Context, id string) (*wallet.DealInvoice, error)
	ListByBuyerCompany(ctx context.Context, buyerCompanyID string, limit int) ([]*wallet.DealInvoice, error)
}

// SellerPayoutRepository persists seller receivable rows (Stage 12+).
type SellerPayoutRepository interface {
	Create(ctx context.Context, payout *wallet.SellerPayout) error
	Save(ctx context.Context, payout *wallet.SellerPayout) error
	LoadByID(ctx context.Context, id string) (*wallet.SellerPayout, error)
	LoadByDealID(ctx context.Context, dealID string) (*wallet.SellerPayout, error)
	LoadByDealIDForUpdate(ctx context.Context, dealID string) (*wallet.SellerPayout, error)
	ListBySellerCompany(ctx context.Context, sellerCompanyID string, limit int) ([]*wallet.SellerPayout, error)
}

type TopUpRepository interface {
	Create(ctx context.Context, topUp *wallet.TopUp) error
	Save(ctx context.Context, topUp *wallet.TopUp) error
	Load(ctx context.Context, id string) (*wallet.TopUp, error)
	LoadForUpdate(ctx context.Context, id string) (*wallet.TopUp, error)
	LoadByProviderPayment(ctx context.Context, provider, providerPaymentID string) (*wallet.TopUp, error)
	LoadByProviderPaymentForUpdate(ctx context.Context, provider, providerPaymentID string) (*wallet.TopUp, error)
	ListByCompany(ctx context.Context, companyID string, limit int) ([]*wallet.TopUp, error)
}

type IDGenerator interface {
	NewID() string
}

type Clock interface {
	Now() time.Time
}

type UnitOfWork interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// ExpiredDealInvoiceLister returns invoice IDs due for expiry (PAYMENT_PENDING, due_at <= now).
// Intended for use inside BillingTx with FOR UPDATE SKIP LOCKED.
type ExpiredDealInvoiceLister interface {
	ListExpired(ctx context.Context, now time.Time, limit int) ([]string, error)
}

type DomainEventPublisher interface {
	Publish(ctx context.Context, aggregateID, companyID string, event any) error
}
