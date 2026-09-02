package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ffgif/config"
	"github.com/labib0x9/ffgif/internal/domain/media"
)

type gifRepo struct {
	db *sqlx.DB
}

func NewGifRepository(db *sqlx.DB, cnf *config.Minio) media.GifRepository {
	return &gifRepo{
		db: db,
	}
}

func (r *gifRepo) Create(ctx context.Context, gif media.Gif) error {
	db := getDBFromCtx(ctx, r.db)
	query := `insert into 
		gifs(user_id, key, thumbnail_url, url, name)
		values(:user_id, :key, :thumbnail_url, :url, :name)
	`

	_, err := sqlx.NamedExecContext(ctx, db, query, gif)
	return err
}

func (r *gifRepo) Get(ctx context.Context, user_id string, status string) ([]media.GifResponse, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `
		select
			name, thumbnail_url, url, key, status, persist, download, created_at
		from
			gifs
		where user_id = $1`

	var val []media.GifResponse
	if status != "all" {
		query += ` and status = $2`
		if err := sqlx.SelectContext(ctx, db, &val, query, user_id, status); err != nil {
			return []media.GifResponse{}, err
		}
	} else {
		if err := sqlx.SelectContext(ctx, db, &val, query, user_id); err != nil {
			return []media.GifResponse{}, err
		}
	}
	return val, nil
}

func (r *gifRepo) GetOwner(ctx context.Context, key string) (string, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `select user_id from gifs where key = $1`
	var userId string
	if err := sqlx.GetContext(ctx, db, &userId, query, key); err != nil {
		return "", err
	}
	return userId, nil
}

func (r *gifRepo) GetByKey(ctx context.Context, key string) (media.GifResponse, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `
		select
			name, thumbnail_url, url,key, status, persist, download, created_at
		from
			gifs
		where key = $1`
	var val media.GifResponse
	if err := sqlx.GetContext(ctx, db, &val, query, key); err != nil {
		return media.GifResponse{}, err
	}
	return val, nil
}

func (r *gifRepo) GetRecents(ctx context.Context, user_id string) ([]media.GifResponse, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `
		select
			name, thumbnail_url, url,key, status, persist, download, created_at
		from
			gifs
		where user_id = $1
		order by created_at desc
        limit 20`

	var val []media.GifResponse
	if err := sqlx.SelectContext(ctx, db, &val, query, user_id); err != nil {
		return []media.GifResponse{}, err
	}
	return val, nil
}

func (r *gifRepo) Delete(ctx context.Context, key string) error {
	db := getDBFromCtx(ctx, r.db)
	query := `delete from gifs where key = $1`
	_, err := db.ExecContext(ctx, query, key)
	return err
}

func (r *gifRepo) Update(ctx context.Context, key string, gif media.GifResponse) error {
	db := getDBFromCtx(ctx, r.db)
	_ = db
	return nil
}

func (r *gifRepo) SaveRecent(ctx context.Context, key string) error {
	db := getDBFromCtx(ctx, r.db)
	_ = db
	return nil
}

type lastVideoRepo struct {
	db *sqlx.DB
}

func NewLastVideoRepository(db *sqlx.DB) media.LastVideoRepository {
	return &lastVideoRepo{db: db}
}

func (l *lastVideoRepo) Create(ctx context.Context, upload media.LastUpload) error {
	db := getDBFromCtx(ctx, l.db)
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
	_, err := sqlx.NamedExecContext(ctx, db, query, upload)
	if err != nil {
		return fmt.Errorf("SaveMetadata: %w", err)
	}
	return nil
}

func (l *lastVideoRepo) GetLastVideo(ctx context.Context, user_id string) (media.LastUploadResponse, error) {
	db := getDBFromCtx(ctx, l.db)
	query := `
		select
			user_id, file_key, file_name, content_type, size_bytes, duration_sec, uploaded_at, thumbnail_url
		from last_upload
		where user_id = $1
	`
	var value media.LastUploadResponse
	if err := sqlx.GetContext(ctx, db, &value, query, user_id); err != nil {
		return media.LastUploadResponse{}, err
	}
	return value, nil
}
