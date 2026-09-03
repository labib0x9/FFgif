package user

import (
	"context"

	"github.com/labib0x9/ffgif/internal/domain/user"
)

func (s *service) GetQuota(ctx context.Context, id string) (*user.Quota, error) {
	return s.quotaRepo.GetById(ctx, id)
}
