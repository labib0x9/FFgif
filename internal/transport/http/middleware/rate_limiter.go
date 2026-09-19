package middleware

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/labib0x9/ffgif/internal/port/cache"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
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
			ip, err := getIp(r.RemoteAddr)
			if err != nil {
				httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
				return
			}
			key := "rate_limit:ip:" + ip
			res, err := rl.setLimit(r.Context(), key)
			if err != nil {
				httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.Rate))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.token))
			w.Header().Set("X-RateLimit-Reset", strconv.Itoa(int(res.last_refill)))

			if !res.allowed {
				retryAfterSecs := (res.wait_ms + 999) / 1000
				if retryAfterSecs < 1 {
					retryAfterSecs = 1
				}
				w.Header().Set("Retry-After", strconv.FormatInt(retryAfterSecs, 10))
				httputil.SendError(w, httputil.RATE_LIMITED, "too many request", http.StatusTooManyRequests)
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

	var data []any
	switch v := res.(type) {
	case []any:
		data = v
	default:
		return Result{}, errors.New("rate limiter: unexpected response type")
	}

	if len(data) < 3 {
		return Result{}, errors.New("rate limiter: response slice has fewer than 3 elements")
	}

	toInt64 := func(v any) (int64, error) {
		switch n := v.(type) {
		case int64:
			return n, nil
		case int:
			return int64(n), nil
		case float64:
			return int64(n), nil
		default:
			return 0, errors.New("rate limiter: unsupported element type")
		}
	}

	allowed, err := toInt64(data[0])
	if err != nil {
		return Result{}, err
	}
	wait_ms, err := toInt64(data[1])
	if err != nil {
		return Result{}, err
	}
	token, err := toInt64(data[2])
	if err != nil {
		return Result{}, err
	}

	return Result{
		allowed:     allowed == 1,
		wait_ms:     wait_ms,
		last_refill: now,
		token:       int(token),
	}, nil
}

func getIp(addr string) (string, error) {
	ip, _, err := net.SplitHostPort(addr)
	return ip, err
}
