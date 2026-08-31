package postgres

import (
	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ffgif/internal/domain/share"
)

type shareRepo struct {
	db *sqlx.DB
	// cnf *config.MinioConfig
}

func NewShareRepository(db *sqlx.DB) share.ShareRepository {
	return &shareRepo{
		db: db,
	}
}
