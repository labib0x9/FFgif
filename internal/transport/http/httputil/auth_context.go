package httputil

import (
	"context"

	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
)

type contextKey struct{}
type authHeaderKey struct{}

var claimKey = contextKey{}
var jwtKey = authHeaderKey{}

func WithAuthContext(ctx context.Context, claims jwtpkg.Payload, token string) context.Context {
	ctx = context.WithValue(ctx, claimKey, claims)
	return context.WithValue(ctx, jwtKey, token)
}

func GetClaims(ctx context.Context) (jwtpkg.Payload, bool) {
	claims, ok := ctx.Value(claimKey).(jwtpkg.Payload)
	return claims, ok
}

func GetAuthorizationHeader(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(jwtKey).(string)
	return token, ok
}

func GetUserId(ctx context.Context) string {
	claims, ok := GetClaims(ctx)
	if !ok {
		return ""
	}
	return claims.RegisteredClaims.Subject
}
