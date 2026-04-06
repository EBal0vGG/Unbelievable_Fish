package app

import "context"

type noopTransactionManager struct{}

func (noopTransactionManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
