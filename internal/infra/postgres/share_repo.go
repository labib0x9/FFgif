package postgres

import (
	"context"

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

func (s *shareRepo) Create(ctx context.Context, gif share.Share) error {
	db := getDBFromCtx(ctx, s.db)
	query := `insert into 
		shares(gif_key, owner_id, shared_with, expires_at)
		values(:gif_key, :owner_id, :shared_with, :expires_at)
	`

	_, err := sqlx.NamedExecContext(ctx, db, query, gif)
	return err
}

func (s *shareRepo) Get(ctx context.Context, userID string) ([]share.GifResponse, error) {
	db := getDBFromCtx(ctx, s.db)
	query := `
		select
			s.gif_key, s.owner_id, s.shared_with, s.expires_at, g.name, g.thumbnail_url, g.url, 
		from
			shares s
		join
			gifs g on g.key = s.gif_key
		where
			s.owner_id = $1 OR s.shared_with = $1
		`

	var val []share.GifResponse

	if err := sqlx.SelectContext(ctx, db, &val, query, userID); err != nil {
		return []share.GifResponse{}, err
	}

	return val, nil
}

// func (s *shareRepo) Delete()            {}
// func (s *shareRepo) GetBySharedWithId() {}
