package repo

import "github.com/jmoiron/sqlx"

type ShareRepository interface {
}

type shareRepo struct {
	db *sqlx.DB
	// cnf *config.MinioConfig
}

func NewShareRepository(db *sqlx.DB) ShareRepository {
	return &shareRepo{
		db: db,
	}
}
