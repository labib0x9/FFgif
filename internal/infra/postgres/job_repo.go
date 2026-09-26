package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ffgif/internal/domain/media"
)

type jobRepo struct {
	db *sqlx.DB
}

func NewJobRepository(db *sqlx.DB) media.JobRepository {
	return &jobRepo{db: db}
}

func (r *jobRepo) Create(ctx context.Context, j media.Job) error {
	db := getDBFromCtx(ctx, r.db)
	query := `
		INSERT INTO jobs (id, user_id, type, status, progress, error_message, result_key, created_at, updated_at)
		VALUES (:id, :user_id, :type, :status, :progress, :error_message, :result_key, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			progress = EXCLUDED.progress,
			error_message = EXCLUDED.error_message,
			result_key = EXCLUDED.result_key,
			updated_at = NOW()
	`
	_, err := sqlx.NamedExecContext(ctx, db, query, j)
	return err
}

func (r *jobRepo) GetByID(ctx context.Context, id string) (*media.Job, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `SELECT id, user_id, type, status, progress, error_message, result_key, created_at, updated_at FROM jobs WHERE id = $1`
	var j media.Job
	if err := sqlx.GetContext(ctx, db, &j, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, media.ErrJobNotFound
		}
		return nil, err
	}
	return &j, nil
}

func (r *jobRepo) UpdateStatus(ctx context.Context, id string, status string, progress int, resultKey string, errMsg string) error {
	db := getDBFromCtx(ctx, r.db)
	query := `
		UPDATE jobs
		SET status = $1, progress = $2, result_key = $3, error_message = $4, updated_at = NOW()
		WHERE id = $5
	`
	_, err := db.ExecContext(ctx, query, status, progress, resultKey, errMsg, id)
	return err
}

func (r *jobRepo) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]media.Job, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `
		SELECT id, user_id, type, status, progress, error_message, result_key, created_at, updated_at
		FROM jobs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	var list []media.Job
	if err := sqlx.SelectContext(ctx, db, &list, query, userID, limit, offset); err != nil {
		return []media.Job{}, err
	}
	return list, nil
}
