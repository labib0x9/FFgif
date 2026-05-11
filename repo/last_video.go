package repo

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ProjectUnsafe/model"
)

type LastVideoRepository interface {
	Create(ctx context.Context, upload model.LastUpload) error
	GetLastVideo(user_id string) (model.LastUploadResp, error)
}

type lastVideoRepo struct {
	db *sqlx.DB
}

func NewLastVideoRepository(db *sqlx.DB) LastVideoRepository {
	return &lastVideoRepo{db: db}
}

func (l *lastVideoRepo) Create(ctx context.Context, upload model.LastUpload) error {
	// query := `
	// INSERT INTO last_uploaded (
	// 	user_id,
	// 	file_key,
	// 	filename
	// )
	// VALUES ($1, $2, $3)
	// `

	// _, err := r.db.ExecContext(
	// 	ctx,
	// 	query,
	// 	upload.UserID,
	// 	upload.FileKey,
	// 	upload.Filename,
	// )

	// return err

	query := `
        INSERT INTO last_upload
            (user_id, file_key, file_name, content_type, size_bytes, uploaded_at)
        VALUES
            (:user_id, :file_key, :file_name, :content_type, :size_bytes, :uploaded_at)
    `
	_, err := l.db.NamedExecContext(ctx, query, upload)
	if err != nil {
		return fmt.Errorf("SaveMetadata: %w", err)
	}
	return nil
}

func (l *lastVideoRepo) GetLastVideo(user_id string) (model.LastUploadResp, error) {
	query := `
		select
			user_id, file_key, file_name, content_type, size_bytes, duration_sec, uploaded_at, thumbnail_url
		from last_upload
		where user_id = $1
	`
	var value model.LastUploadResp
	if err := l.db.Get(&value, query, user_id); err != nil {
		return model.LastUploadResp{}, err
	}
	return value, nil
}
