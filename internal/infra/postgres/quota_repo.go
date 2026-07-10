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
