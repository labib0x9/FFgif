package cache

import "errors"

// ErrCacheMiss reports that a key is absent from the cache.
//
// It exists so callers can tell "there is no such entry" apart from "the cache
// backend is unreachable". Those two must never collapse into one branch: a
// miss is a normal, expected outcome, while a backend fault has to surface as
// an error rather than being reported to the user as an absent/failed job.
//
// Cache implementations are expected to wrap their driver's not-found value
// (for go-redis, redis.Nil) in this sentinel.
var ErrCacheMiss = errors.New("cache: key not found")
