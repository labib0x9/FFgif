package repo

import (
	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ProjectUnsafe/model"
)

type AnonQuotaRepository interface {
	Create(quota model.Quota) error
	GetById(userId string) (model.Quota, error)
}

type anonQuotaRepo struct {
	dbConn *sqlx.DB
}

func NewAnonQuotaRepository(db *sqlx.DB) AnonQuotaRepository {
	return &anonQuotaRepo{dbConn: db}
}

func (r *anonQuotaRepo) Create(quota model.Quota) error {
	query := `insert into 
		anon_quota(user_id)
		values(:user_id)
	`

	_, err := r.dbConn.NamedExec(query, quota)
	return err
}

func (r *anonQuotaRepo) GetById(userId string) (model.Quota, error) {
	query := `select * from anon_quota where user_id = $1`
	var quota model.Quota
	if err := r.dbConn.Get(&quota, query, userId); err != nil {
		return model.Quota{}, err
	}
	return quota, nil
}
