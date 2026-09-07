package share

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/domain/share"
	"github.com/labib0x9/ffgif/internal/port/queue"
	"github.com/labib0x9/ffgif/pkg/apperr"
	"github.com/labib0x9/ffgif/pkg/token"
)

func (s *service) CreateByToken(ctx context.Context, sharedBy string, gifKey string, sharedWithEmail string, expiresAt time.Time) (string, error) {
	owner, err := s.gifRepo.GetOwner(ctx, gifKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("gifRepo.GetOwner: %w: %w", media.ErrGifNotFound, err)
		}
		return "", fmt.Errorf("gifRepo.GetOwner: %w", err)
	}

	if owner != sharedBy {
		return "", media.ErrGifOwnerMismatch
	}

	_token, _ := token.GenerateToken()

	share := share.ShareByToken{
		GifKey:    gifKey,
		Token:     _token,
		Email:     sharedWithEmail,
		ExpiresAt: &expiresAt,
	}

	if err := s.shareRepo.CreateByToken(ctx, share); err != nil {
		return "", fmt.Errorf("shareRepo.CreateByToken: %w", err)
	}

	smsg := queue.EmailMessage{
		To:    sharedWithEmail,
		Name:  "share",
		Token: _token,
	}

	if err := s.queue.PublishEmail(ctx, smsg); err != nil {
		return _token, fmt.Errorf("queue.PublishEmail: %w: %w", apperr.ErrMessageQueueFailed, err)
	}

	return _token, nil
}
