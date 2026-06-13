package redis

import (
	"context"
	"time"

	"github.com/labib0x9/ProjectUnsafe/config"
	"github.com/redis/go-redis/v9"
)

var luaCode = `
	local key = KEYS[1]
	local capacity = tonumber(ARGV[1])
	local rate = tonumber(ARGV[2])
	local now = tonumber(ARGV[3])

	local data = redis.call("HMGET", key, "token", "last_refill")
	local token = tonumber(data[1])
	local last_refill = tonumber(data[2])

	if token == nil then
		token = rate
		last_refill = now
	end

	local diff = (now - last_refill) / 1000.0
	local add_token = diff * rate
	token = math.min(capacity, token + add_token)
	last_refill = now

	if token >= 1 then
		token = token - 1
		redis.call("HMSET", key, "token", token, "last_refill", last_refill)
		redis.call("PEXPIRE", key, 6000)
		return {1, 0, token}
	end

	local token_need = 1 - token
	local wait_ms = math.ceil((token_need / rate) * 1000) 

	redis.call("HMSET", key, "token", token, "last_refill", last_refill)
	redis.call("PEXPIRE", key, 6000)

	return {0, wait_ms, token}
`

type Redis struct {
	Client *redis.Client
	Script *redis.Script
}

func NewRedis(cnf *config.RedisConfig) *Redis {
	return &Redis{
		Client: redis.NewClient(
			&redis.Options{
				Addr: cnf.Addr,
				// Username: cnf.User,
				// Password: cnf.Pass,
			},
		),
		Script: redis.NewScript(luaCode),
	}
}

func (r *Redis) Close() error {
	return r.Client.Close()
}

func (r *Redis) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return r.Client.Ping(ctx).Err()
}

func Setup(cnf *config.RedisConfig) *Redis {
	r := NewRedis(cnf)
	if err := r.Ping(); err != nil {
		panic(err)
	}
	return r
}
