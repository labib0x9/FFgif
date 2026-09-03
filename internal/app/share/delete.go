package share

import (
	"context"
	"database/sql"
	"errors"

	"github.com/labib0x9/ffgif/internal/domain/share"
)

func (h *service) Delete(ctx context.Context, userId, gifKey, shareWithId string) error {
	owner, err := h.shareRepo.GetOwner(ctx, shareWithId, gifKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return share.ErrNotFound
		}
		return err
	}
	if owner != userId {
		return share.ErrNotAuthorized
	}

	return h.shareRepo.Delete(ctx, gifKey, shareWithId)
}
