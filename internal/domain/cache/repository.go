package cache

import (
	"context"
	"time"
)

type Cache interface {
	Set(ctx context.Context, key string, value string, expire time.Duration) error
	Get(ctx context.Context, key string) (string, error)
}
