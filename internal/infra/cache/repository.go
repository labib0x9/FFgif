package cache

import (
	"context"
	"time"
)

type CacheRepo interface {
	Set(ctx context.Context, key string, value string, expire time.Duration) error
	Get(ctx context.Context, key string) (string, error)
}
