package postgres

import (
	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ffgif/internal/domain/user"
)

type quotaRepo struct {
	dbConn *sqlx.DB
}

func NewQuotaRepository(db *sqlx.DB) user.QuotaRepository {
	return &quotaRepo{dbConn: db}
}

func (r *quotaRepo) Create(quota user.Quota) error {
	query := `insert into 
		quota(user_id)
		values(:user_id)
	`

	_, err := r.dbConn.NamedExec(query, quota)
	return err
}

func (r *quotaRepo) GetById(userId string) (*user.Quota, error) {
	query := `select * from quota where user_id = $1`
	var quota user.Quota
	if err := r.dbConn.Get(&quota, query, userId); err != nil {
		return nil, err
	}
	return &quota, nil
}

type anonQuotaRepo struct {
	dbConn *sqlx.DB
}

func NewAnonQuotaRepository(db *sqlx.DB) user.AnonQuotaRepository {
	return &anonQuotaRepo{dbConn: db}
}

func (r *anonQuotaRepo) Create(quota user.Quota) error {
	query := `insert into 
		anon_quota(user_id)
		values(:user_id)
	`

	_, err := r.dbConn.NamedExec(query, quota)
	return err
}

func (r *anonQuotaRepo) GetById(userId string) (*user.Quota, error) {
	query := `select * from anon_quota where user_id = $1`
	var quota user.Quota
	if err := r.dbConn.Get(&quota, query, userId); err != nil {
		return nil, err
	}
	return &quota, nil
}
