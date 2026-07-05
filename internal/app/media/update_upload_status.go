package media

import (
	"context"
	"time"
)

// key = filename, lookup key is generated here
func (s *service) UpdateUploadingStatus(ctx context.Context, key string, status string) error {
	lookupKey := "uploading:" + key
	return s.cache.Set(ctx, lookupKey, status, 5*time.Minute)
}
