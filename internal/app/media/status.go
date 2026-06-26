package media

import "context"

func (s *service) Status(ctx context.Context, key string) (bool, error) {
	return s.storage.IsExists(ctx, key)
}
