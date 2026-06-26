package redis

import (
	"context"
	"log/slog"
	"time"

	"github.com/labib0x9/ffgif/config"
	"github.com/redis/go-redis/v9"
)

func ping(client *redis.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return client.Ping(ctx).Err()
}

func Setup(cnf *config.RedisConfig) *redis.Client {
	client := redis.NewClient(
		&redis.Options{
			Addr: cnf.Addr,
			// Username: cnf.User,
			// Password: cnf.Pass,
		},
	)
	if err := ping(client); err != nil {
		panic(err)
	}

	slog.Info("Redis connection complete")
	return client
}
