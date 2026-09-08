package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/labib0x9/ffgif/config"
	cacheMocks "github.com/labib0x9/ffgif/internal/port/cache/mocks"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockCache := cacheMocks.NewMockCache(ctrl)

	secret := []byte("test-secret-key-1234-5678-901234")
	jwtProvider := jwtpkg.NewJwt(secret)

	tokenStr, err := jwtProvider.Create("John Doe", "user-uuid-1", "john@example.com", "user")
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// Cache miss means token is not blocklisted
	mockCache.EXPECT().Get(gomock.Any(), gomock.Eq("token_blocklist:"+tokenStr)).
		Return("", errors.New("key not found in cache")).Times(1)

	middlewares := middleware.NewMiddlewares(&config.Config{}, mockCache, *jwtProvider)

	nextHandlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextHandlerCalled = true
		userId := httputil.GetUserId(r.Context())
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
	ctrl := gomock.NewController(t)
	mockCache := cacheMocks.NewMockCache(ctrl)

	secret := []byte("test-secret-key-1234-5678-901234")
	jwtProvider := jwtpkg.NewJwt(secret)
	middlewares := middleware.NewMiddlewares(&config.Config{}, mockCache, *jwtProvider)

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
	ctrl := gomock.NewController(t)
	mockCache := cacheMocks.NewMockCache(ctrl)

	secret := []byte("test-secret-key-1234-5678-901234")
	jwtProvider := jwtpkg.NewJwt(secret)

	tokenStr, _ := jwtProvider.Create("John Doe", "user-uuid-1", "john@example.com", "user")

	mockCache.EXPECT().Get(gomock.Any(), gomock.Eq("token_blocklist:"+tokenStr)).
		Return("blocked", nil).Times(1)

	middlewares := middleware.NewMiddlewares(&config.Config{}, mockCache, *jwtProvider)

	nextCalled := false
	handler := middlewares.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized for blocklisted token, got %d", rec.Code)
	}
	if nextCalled {
		t.Errorf("next handler must not be called for blocklisted token")
	}
}

// EXPECTED TO FAIL: Auth blocklist fail-open vulnerability in internal/transport/http/middleware/auth.go.
// When Redis fails (returns a backend connection error), the middleware treats all errors as
// "token not blocklisted" and accepts potentially revoked tokens rather than failing safe.
func TestAuthMiddleware_Blocklist_FailOpen_Adversarial(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockCache := cacheMocks.NewMockCache(ctrl)

	secret := []byte("test-secret-key-1234-5678-901234")
	jwtProvider := jwtpkg.NewJwt(secret)

	tokenStr, _ := jwtProvider.Create("Revoked User", "revoked-user-uuid", "revoked@example.com", "user")

	// Simulate Redis outage (connection error / timeout)
	mockCache.EXPECT().Get(gomock.Any(), gomock.Eq("token_blocklist:"+tokenStr)).
		Return("", errors.New("dial tcp 127.0.0.1:6379: connect: connection refused")).Times(1)

	middlewares := middleware.NewMiddlewares(&config.Config{}, mockCache, *jwtProvider)

	nextCalled := false
	handler := middlewares.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if nextCalled || rec.Code == http.StatusOK {
		t.Errorf("SECURITY VULNERABILITY DETECTED: Auth middleware failed open on Redis outage — accepted token and invoked protected handler (status %d)", rec.Code)
	}
}

func TestAuthMiddleware_TamperedSignature(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockCache := cacheMocks.NewMockCache(ctrl)

	serverSecret := []byte("server-secret-key-1234-5678-901234")
	attackerSecret := []byte("attacker-secret-key-1234-5678-9012")

	serverJwt := jwtpkg.NewJwt(serverSecret)
	attackerJwt := jwtpkg.NewJwt(attackerSecret)

	forgedToken, _ := attackerJwt.Create("Attacker", "admin-uuid", "admin@example.com", "admin")

	middlewares := middleware.NewMiddlewares(&config.Config{}, mockCache, *serverJwt)

	handler := middlewares.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+forgedToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized for forged token, got %d", rec.Code)
	}
}
