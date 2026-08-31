package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labib0x9/ffgif/config"
	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
)

type mockCache struct {
	store map[string]string
}

func (m *mockCache) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	m.store[key] = value
	return nil
}
func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
	if v, ok := m.store[key]; ok {
		return v, nil
	}
	return "", errors.New("key not found in cache")
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	secret := []byte("test-secret-key-1234")
	jwtProvider := jwtpkg.NewJwt(secret)
	cache := &mockCache{store: make(map[string]string)}

	middlewares := middleware.NewMiddlewares(&config.Config{}, cache, *jwtProvider)

	tokenStr, err := jwtProvider.Create("John Doe", "user-uuid-1", "john@example.com", "user")
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	nextHandlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextHandlerCalled = true
		userId := middleware.GetUserId(r)
		if userId != "user-uuid-1" {
			t.Errorf("expected userId user-uuid-1 in context, got %s", userId)
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := middlewares.Auth(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if !nextHandlerCalled {
		t.Errorf("expected next handler to be called")
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	secret := []byte("test-secret-key-1234")
	jwtProvider := jwtpkg.NewJwt(secret)
	cache := &mockCache{store: make(map[string]string)}
	middlewares := middleware.NewMiddlewares(&config.Config{}, cache, *jwtProvider)

	handler := middlewares.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized for missing header, got %d", rec.Code)
	}
}

func TestAuthMiddleware_BlocklistedToken(t *testing.T) {
	secret := []byte("test-secret-key-1234")
	jwtProvider := jwtpkg.NewJwt(secret)
	cache := &mockCache{store: make(map[string]string)}

	tokenStr, _ := jwtProvider.Create("John Doe", "user-uuid-1", "john@example.com", "user")
	// Put token on blocklist
	cache.store["token_blocklist:"+tokenStr] = "blocked"

	middlewares := middleware.NewMiddlewares(&config.Config{}, cache, *jwtProvider)

	handler := middlewares.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized for blocklisted token, got %d", rec.Code)
	}
}
