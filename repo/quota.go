package repo

import (
	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ProjectUnsafe/model"
)

type QuotaRepository interface {
	Create(quota model.Quota) error
	GetById(userId string) (model.Quota, error)
}

type quotaRepo struct {
	dbConn *sqlx.DB
}

func NewQuotaRepository(db *sqlx.DB) QuotaRepository {
	return &quotaRepo{dbConn: db}
}

func (r *quotaRepo) Create(quota model.Quota) error {
	query := `insert into 
		quota(user_id)
		values(:user_id)
	`

	_, err := r.dbConn.NamedExec(query, quota)
	return err
}

func (r *quotaRepo) GetById(userId string) (model.Quota, error) {
	query := `select * from quota where user_id = $1`
	var quota model.Quota
	if err := r.dbConn.Get(&quota, query, userId); err != nil {
		return model.Quota{}, err
	}
	return quota, nil
}
