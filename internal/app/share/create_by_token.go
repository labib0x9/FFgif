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
	if !expiresAt.After(time.Now()) {
		return "", share.ErrInvalidExpiry
	}

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

	rawToken, tokenHash := token.GenerateToken()

	sh := share.ShareByToken{
		GifKey:    gifKey,
		Token:     tokenHash,
		Email:     sharedWithEmail,
		ExpiresAt: &expiresAt,
	}

	if err := s.shareRepo.CreateByToken(ctx, sh); err != nil {
		return "", fmt.Errorf("shareRepo.CreateByToken: %w", err)
	}

	smsg := queue.EmailMessage{
		To:    sharedWithEmail,
		Name:  "share",
		Token: rawToken,
	}

	if err := s.queue.PublishEmail(ctx, smsg); err != nil {
		return rawToken, fmt.Errorf("queue.PublishEmail: %w: %w", apperr.ErrMessageQueueFailed, err)
	}

	return rawToken, nil
}
