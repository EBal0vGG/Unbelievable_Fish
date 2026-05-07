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
}

type LedgerRepository interface {
	Append(ctx context.Context, entry wallet.LedgerEntry) error
	ExistsByReference(ctx context.Context, companyID, referenceType, referenceID string, typ wallet.LedgerEntryType) (bool, error)
}

// LedgerQuery read-side for HTTP (implemented by postgres.LedgerLister).
type LedgerQuery interface {
	ListByCompany(ctx context.Context, companyID string, limit int) ([]wallet.LedgerEntry, error)
}

type ProcessedTopUpRepository interface {
	InsertIfNew(ctx context.Context, externalPaymentID, companyID, accountID string, amount int64) (inserted bool, err error)
}

type IDGenerator interface {
	NewID() string
}

type Clock interface {
	Now() time.Time
}
