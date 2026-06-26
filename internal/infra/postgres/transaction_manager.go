package postgres

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ffgif/internal/domain/db"
)

type txKey struct{}

type txManager struct {
	db *sqlx.DB
}

func NewTxManager(db *sqlx.DB) db.TxManager {
	return &txManager{
		db: db,
	}
}

func (t *txManager) With(ctx context.Context, fn func(ctx context.Context) (any, error)) (any, error) {
	tx, err := t.db.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
	})
	if err != nil {
		return nil, err
	}

	txCtx := context.WithValue(ctx, txKey{}, tx)
	result, err := fn(txCtx)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	return result, tx.Commit()
}

// Get db connection from context, if absent fallback to db
func getDBFromCtx(ctx context.Context, db *sqlx.DB) sqlx.ExtContext {
	if tx, ok := ctx.Value(txKey{}).(*sqlx.Tx); ok {
		return tx
	}
	return db
}
