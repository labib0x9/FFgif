package httputil

import (
	"context"
)

type requestIDKey struct{}

var rKey = requestIDKey{}

func WithLoggerContext(ctx context.Context, requestId string) context.Context {
	return context.WithValue(ctx, rKey, requestId)
}

func GetRequestID(ctx context.Context) string {
	id, _ := ctx.Value(rKey).(string)
	return id
}
