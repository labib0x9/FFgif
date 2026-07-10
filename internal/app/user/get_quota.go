package user

import (
	"context"

	"github.com/labib0x9/ffgif/internal/domain/user"
)

func (s *service) GetQuota(ctx context.Context, id string) (*user.Quota, error) {

	// quota, err :=
	// if err != nil {
	// 	http.Error(w, "internal server error", http.StatusInternalServerError)
	// 	slog.Error("GetQuota: quota not found", "err", err, "id", id)
	// 	return
	// }

	return s.quotaRepo.GetById(ctx, id)
}
