package media

import "context"

func (s *service) Save(ctx context.Context, userId, key string) error {
	// To-do, validate user
	return s.gifRepo.SaveRecent(ctx, key)
}
