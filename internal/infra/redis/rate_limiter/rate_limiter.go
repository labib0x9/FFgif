package ratelimitter

import (
	"context"

	"github.com/labib0x9/ffgif/internal/domain/cache"
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

type rateLimiter struct {
	client *redis.Client
	script *redis.Script
}

func NewRateLimiter(client *redis.Client) cache.RateLimiter {
	return &rateLimiter{
		client: client,
		script: redis.NewScript(luaCode),
	}
}

func (r *rateLimiter) RunScript(ctx context.Context, key string, capacity int, rate int, now int64) (interface{}, error) {
	return r.script.Run(
		ctx,
		r.client,
		[]string{key},
		capacity,
		rate,
		now,
	).Result()
}
