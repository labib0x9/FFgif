package postgres

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ffgif/internal/domain/user"
)

type quotaRepo struct {
	db *sqlx.DB
}

func NewQuotaRepository(db *sqlx.DB) user.QuotaRepository {
	return &quotaRepo{db: db}
}

func (r *quotaRepo) Create(ctx context.Context, quota user.Quota) error {
	db := getDBFromCtx(ctx, r.db)
	query := `insert into 
		quota(user_id)
		values(:user_id)
	`

	_, err := sqlx.NamedExecContext(ctx, db, query, quota)
	return err
}

func (r *quotaRepo) GetById(ctx context.Context, userId string) (*user.Quota, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `select * from quota where user_id = $1`
	var quota user.Quota
	if err := sqlx.GetContext(ctx, db, &quota, query, userId); err != nil {
		return nil, err
	}
	return &quota, nil
}

func (r *quotaRepo) IncrementUsage(ctx context.Context, userId string, addBytes int, addGifCount int) (bool, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `
		UPDATE quota
		SET used_bytes = used_bytes + $1, gif_count = gif_count + $2
		WHERE user_id = $3
		  AND used_bytes + $1 <= total_bytes
		  AND gif_count + $2 <= gif_limit
	`
	res, err := db.ExecContext(ctx, query, addBytes, addGifCount, userId)
	if err != nil {
		return false, err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

type anonQuotaRepo struct {
	db *sqlx.DB
}

func NewAnonQuotaRepository(db *sqlx.DB) user.AnonQuotaRepository {
	return &anonQuotaRepo{db: db}
}

func (r *anonQuotaRepo) Create(ctx context.Context, quota user.Quota) error {
	db := getDBFromCtx(ctx, r.db)
	query := `insert into 
		anon_quota(user_id)
		values(:user_id)
	`

	_, err := sqlx.NamedExecContext(ctx, db, query, quota)
	return err
}

func (r *anonQuotaRepo) GetById(ctx context.Context, userId string) (*user.Quota, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `select * from anon_quota where user_id = $1`
	var quota user.Quota
	if err := sqlx.GetContext(ctx, db, &quota, query, userId); err != nil {
		return nil, err
	}
	return &quota, nil
}
