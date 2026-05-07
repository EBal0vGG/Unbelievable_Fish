package app

import "context"

// CompanyCreatedPublisher enqueues identity.CompanyCreated for integration consumers.
type CompanyCreatedPublisher interface {
	PublishCompanyCreated(ctx context.Context, companyID string) error
}

// UnitOfWork runs a function inside a DB transaction (optional for tests).
type UnitOfWork interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
