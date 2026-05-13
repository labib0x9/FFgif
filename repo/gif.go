package repo

import (
	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ProjectUnsafe/model"
)

type GifRepository interface {
	Create(model.Gif) error
}

type gifRepo struct {
	db *sqlx.DB
}

func NewGifRepository(db *sqlx.DB) GifRepository {
	return &gifRepo{db: db}
}

func (r *gifRepo) Create(model.Gif) error {
	return nil
}
