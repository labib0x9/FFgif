package db

import "context"

//go:generate mockgen -source=repository.go -destination=mocks/mock_tx_manager.go -package=mocks
type TxManager interface {
	With(ctx context.Context, fn func(ctx context.Context) (any, error)) (any, error)
	WithRC(ctx context.Context, fn func(ctx context.Context) (any, error)) (any, error)
}
