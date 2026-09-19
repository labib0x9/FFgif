package share

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/share"
)

func (s *service) DownloadByToken(ctx context.Context, token string) (string, error) {
	resp, err := s.shareRepo.GetByToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", share.ErrNotFound
		}
		return "", fmt.Errorf("shareRepo.GetByToken: %w", err)
	}

	if err := s.gifRepo.IncrementDownload(ctx, resp.GifKey); err != nil {
		return "", fmt.Errorf("gifRepo.IncrementDownload: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	u, err := s.storage.Download(timeoutCtx, resp.GifKey, 5*time.Minute)
	if err != nil {
		return "", fmt.Errorf("storage.Download: %w", err)
	}
	return u.String(), nil
}
