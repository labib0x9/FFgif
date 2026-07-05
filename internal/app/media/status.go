package media

import "context"

func (s *service) Status(ctx context.Context, key string) (string, error) {
	lookupKey := "uploading:" + key
	status, err := s.cache.Get(ctx, lookupKey)
	if err != nil {
		return "", err
	}
	if status == "" {
		status = "failed"
	}
	return status, nil
}
