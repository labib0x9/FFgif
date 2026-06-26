package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/cache"
)

type RateLimiter struct {
	Client   cache.RateLimiter
	Rate     int
	Capacity int
}

type Result struct {
	allowed     bool
	wait_ms     int64
	token       int
	last_refill int64
}

func NewRateLimiter(
	client cache.RateLimiter,
	rate int,
	capacity int,
) *RateLimiter {
	return &RateLimiter{
		Client:   client,
		Rate:     rate,
		Capacity: capacity,
	}
}

func (rl *RateLimiter) Limit() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := strings.Split(r.RemoteAddr, ":")[0]
			key := "rate_limit:ip:" + ip
			res, err := rl.setLimit(r.Context(), key)
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.Rate))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.token))
			w.Header().Set("X-RateLimit-Reset", strconv.Itoa(int(res.last_refill)))

			if !res.allowed {
				retryAfterSecs := res.wait_ms / 1000
				w.Header().Set("Retry-After", strconv.FormatInt(retryAfterSecs, 10))
				http.Error(w, "too many request", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (rl *RateLimiter) setLimit(ctx context.Context, key string) (Result, error) {
	now := time.Now().UnixMilli()

	res, err := rl.Client.RunScript(ctx, key, rl.Capacity, rl.Rate, now)
	if err != nil {
		return Result{}, err
	}

	data := res.([]interface{})
	allowed := data[0].(int64)
	wait_ms := data[1].(int64)
	token := data[2].(int64)

	return Result{
		allowed:     allowed == 1,
		wait_ms:     wait_ms,
		last_refill: now,
		token:       int(token),
	}, nil
}
