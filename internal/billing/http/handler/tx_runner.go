package handler

import "context"

type TxRunner interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
