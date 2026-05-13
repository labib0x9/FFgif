package repo

import (
	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ProjectUnsafe/model"
)

type GifRepository interface {
	Create(gif model.Gif) error
	Get(user_id string, status string) ([]model.GifResp, error)
}

type gifRepo struct {
	db *sqlx.DB
}

func NewGifRepository(db *sqlx.DB) GifRepository {
	return &gifRepo{db: db}
}

func (r *gifRepo) Create(gif model.Gif) error {
	query := `insert into 
		gifs(user_id, key)
		values(:user_id, :key)
	`

	_, err := r.db.NamedExec(query, gif)
	return err
}

func (r *gifRepo) Get(user_id string, status string) ([]model.GifResp, error) {
	query := `
		select
			key, status, persist, download, created_at
		from
			gifs
		where user_id = $1`

	var val []model.GifResp
	if status != "all" {
		query += `and status = $2`
		if err := r.db.Get(&val, query, user_id, status); err != nil {
			return []model.GifResp{}, err
		}
	} else {
		if err := r.db.Get(&val, query, user_id); err != nil {
			return []model.GifResp{}, err
		}
	}
	return val, nil
}
