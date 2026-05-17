package repo

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ProjectUnsafe/config"
	"github.com/labib0x9/ProjectUnsafe/model"
)

type GifRepository interface {
	Create(gif model.Gif) error
	Get(user_id string, status string) ([]model.GifResp, error)
	GetByKey(key string) (model.GifResp, error)
	GetUrl(key string) string
	GetRecents(user_id string) ([]model.GifResp, error)
	Delete(key string) error
	Update(key string, gif model.GifResp) error
	SaveRecent(key string) error
}

type gifRepo struct {
	db  *sqlx.DB
	cnf *config.MinioConfig
}

func NewGifRepository(db *sqlx.DB, cnf *config.MinioConfig) GifRepository {
	return &gifRepo{
		db:  db,
		cnf: cnf,
	}
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
		query += ` and status = $2`
		if err := r.db.Select(&val, query, user_id, status); err != nil {
			return []model.GifResp{}, err
		}
	} else {
		if err := r.db.Select(&val, query, user_id); err != nil {
			return []model.GifResp{}, err
		}
	}
	return val, nil
}

func (r *gifRepo) GetUrl(key string) string {
	return fmt.Sprintf("http://localhost:9000/%s/%s", r.cnf.BucketName, key)
}

func (r *gifRepo) GetByKey(key string) (model.GifResp, error) {
	query := `
		select
			key, status, persist, download, created_at
		from
			gifs
		where key = $1`
	var val model.GifResp
	if err := r.db.Get(&val, query, key); err != nil {
		return model.GifResp{}, err
	}
	return val, nil
}

func (r *gifRepo) GetRecents(user_id string) ([]model.GifResp, error) {
	query := `
		select
			key, status, persist, download, created_at
		from
			gifs
		where user_id = $1
		order by created_at desc
        limit 20`

	var val []model.GifResp
	if err := r.db.Select(&val, query, user_id); err != nil {
		return []model.GifResp{}, err
	}
	return val, nil
}

func (r *gifRepo) Delete(key string) error {
	query := `delete from gifs where key = $1`
	_, err := r.db.Exec(query, key)
	return err
}

func (r *gifRepo) Update(key string, gif model.GifResp) error {
	return nil
}

func (r *gifRepo) SaveRecent(key string) error {
	return nil
}
