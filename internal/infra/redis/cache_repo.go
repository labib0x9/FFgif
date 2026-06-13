package redis

import (
	"context"
	"time"

	"github.com/labib0x9/ProjectUnsafe/internal/infra/cache"
)

type cacheRepo struct {
	cache *Redis
}

func NewCacheRepo(
	cache *Redis,
) cache.CacheRepo {
	return &cacheRepo{
		cache: cache,
	}
}

func (r *cacheRepo) Set(ctx context.Context, key string, value string, expire time.Duration) error {
	return r.cache.Client.Set(
		ctx,
		key,
		value,
		expire,
	).Err()
}

func (r *cacheRepo) Get(ctx context.Context, key string) (string, error) {
	return r.cache.Client.Get(ctx, key).Result()
}
