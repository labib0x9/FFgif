package db

import "context"

type TxManager interface {
	With(ctx context.Context, fn func(ctx context.Context) (any, error)) (any, error)
}
