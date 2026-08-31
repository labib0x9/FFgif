package user

import (
	"context"

	"github.com/labib0x9/ffgif/internal/domain/user"
)

func (s *service) UpdateProfile(ctx context.Context, profile user.ProfileResponse, id string) (user.ProfileResponse, error) {
	return s.userRepo.UpdateProfile(ctx, profile, id)
	// if err != nil {
	// 	http.Error(w, "invalid credentials", http.StatusUnauthorized)
	// 	slog.Warn("UpdateProfile: update failed", "error", err)
	// 	return
	// }
}
