package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ProjectUnsafe/config"
	"github.com/labib0x9/ProjectUnsafe/internal/domain/media"
)

type gifRepo struct {
	db  *sqlx.DB
	cnf *config.MinioConfig
}

func NewGifRepository(db *sqlx.DB, cnf *config.MinioConfig) media.GifRepository {
	return &gifRepo{
		db:  db,
		cnf: cnf,
	}
}

func (r *gifRepo) Create(gif media.Gif) error {
	query := `insert into 
		gifs(user_id, key)
		values(:user_id, :key)
	`

	_, err := r.db.NamedExec(query, gif)
	return err
}

func (r *gifRepo) Get(user_id string, status string) ([]media.GifResp, error) {
	query := `
		select
			key, status, persist, download, created_at
		from
			gifs
		where user_id = $1`

	var val []media.GifResp
	if status != "all" {
		query += ` and status = $2`
		if err := r.db.Select(&val, query, user_id, status); err != nil {
			return []media.GifResp{}, err
		}
	} else {
		if err := r.db.Select(&val, query, user_id); err != nil {
			return []media.GifResp{}, err
		}
	}
	return val, nil
}

func (r *gifRepo) GetUrl(key string) string {
	return fmt.Sprintf("http://localhost:9000/%s/%s", r.cnf.BucketName, key)
}

func (r *gifRepo) GetByKey(key string) (media.GifResp, error) {
	query := `
		select
			key, status, persist, download, created_at
		from
			gifs
		where key = $1`
	var val media.GifResp
	if err := r.db.Get(&val, query, key); err != nil {
		return media.GifResp{}, err
	}
	return val, nil
}

func (r *gifRepo) GetRecents(user_id string) ([]media.GifResp, error) {
	query := `
		select
			key, status, persist, download, created_at
		from
			gifs
		where user_id = $1
		order by created_at desc
        limit 20`

	var val []media.GifResp
	if err := r.db.Select(&val, query, user_id); err != nil {
		return []media.GifResp{}, err
	}
	return val, nil
}

func (r *gifRepo) Delete(key string) error {
	query := `delete from gifs where key = $1`
	_, err := r.db.Exec(query, key)
	return err
}

func (r *gifRepo) Update(key string, gif media.GifResp) error {
	return nil
}

func (r *gifRepo) SaveRecent(key string) error {
	return nil
}

type lastVideoRepo struct {
	db *sqlx.DB
}

func NewLastVideoRepository(db *sqlx.DB) media.LastVideoRepository {
	return &lastVideoRepo{db: db}
}

func (l *lastVideoRepo) Create(ctx context.Context, upload media.LastUpload) error {
	query := `
        INSERT INTO last_upload
            (user_id, file_key, file_name, content_type, size_bytes, uploaded_at, updated_at)
        VALUES
            (:user_id, :file_key, :file_name, :content_type, :size_bytes, :uploaded_at, NOW())
        ON CONFLICT (user_id) DO UPDATE SET
            file_key     = EXCLUDED.file_key,
            file_name    = EXCLUDED.file_name,
            content_type = EXCLUDED.content_type,
            size_bytes   = EXCLUDED.size_bytes,
            uploaded_at  = EXCLUDED.uploaded_at,
            updated_at   = NOW()
    `
	_, err := l.db.NamedExecContext(ctx, query, upload)
	if err != nil {
		return fmt.Errorf("SaveMetadata: %w", err)
	}
	return nil
}

func (l *lastVideoRepo) GetLastVideo(user_id string) (media.LastUploadResp, error) {
	query := `
		select
			user_id, file_key, file_name, content_type, size_bytes, duration_sec, uploaded_at, thumbnail_url
		from last_upload
		where user_id = $1
	`
	var value media.LastUploadResp
	if err := l.db.Get(&value, query, user_id); err != nil {
		return media.LastUploadResp{}, err
	}
	return value, nil
}
