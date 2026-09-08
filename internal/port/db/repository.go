//go:generate mockgen -package=mocks -destination=mocks/mock_tx_manager.go github.com/labib0x9/ffgif/internal/port/db TxManager

package db

import "context"

type TxManager interface {
	With(ctx context.Context, fn func(ctx context.Context) (any, error)) (any, error)
	WithRC(ctx context.Context, fn func(ctx context.Context) (any, error)) (any, error)
}
