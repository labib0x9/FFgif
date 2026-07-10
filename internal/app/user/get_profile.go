package user

import (
	"context"

	"github.com/labib0x9/ffgif/internal/domain/user"
)

func (s *service) GetProfile(ctx context.Context, id string) (user.ProfileResp, error) {
	// found, err :=
	// if err != nil {
	// 	http.Error(w, "internal server error", http.StatusInternalServerError)
	// 	slog.Error("GetProfile: user not found", "err", err, "id", id)
	// 	return
	// }
	return s.userRepo.GetProfile(ctx, id)
}
