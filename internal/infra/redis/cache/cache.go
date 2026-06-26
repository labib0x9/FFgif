package cache

import (
	"context"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/cache"
	go_redis "github.com/redis/go-redis/v9"
)

type redis struct {
	client *go_redis.Client
}

func NewCache(client *go_redis.Client) cache.Cache {
	return &redis{
		client: client,
	}
}

func (r *redis) Set(ctx context.Context, key string, value string, expire time.Duration) error {
	return r.client.Set(
		ctx,
		key,
		value,
		expire,
	).Err()
}

func (r *redis) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}
