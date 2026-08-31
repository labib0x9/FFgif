package share_test

// import (
// 	"context"
// 	"errors"
// 	"net/http"
// 	"net/http/httptest"
// 	"testing"
// 	"time"

// 	"github.com/go-playground/validator/v10"
// 	domainshare "github.com/labib0x9/ffgif/internal/domain/share"
// 	"github.com/labib0x9/ffgif/internal/transport/http/handlers/share"
// 	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
// 	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
// )

// type mockShareService struct {
// 	createFunc func(ctx context.Context, sharedBy string, gifKey string, sharedWith string, expiresAt time.Time) error
// 	getFunc    func(ctx context.Context, user string) ([]domainshare.GifResp, error)
// }

// func (m *mockShareService) Create(ctx context.Context, sharedBy string, gifKey string, sharedWith string, expiresAt time.Time) error {
// 	if m.createFunc != nil {
// 		return m.createFunc(ctx, sharedBy, gifKey, sharedWith, expiresAt)
// 	}
// 	return nil
// }
// func (m *mockShareService) Delete()   {}
// func (m *mockShareService) Download() {}
// func (m *mockShareService) Get(ctx context.Context, user string) ([]domainshare.GifResp, error) {
// 	if m.getFunc != nil {
// 		return m.getFunc(ctx, user)
// 	}
// 	return []domainshare.GifResp{}, nil
// }
// func (m *mockShareService) Update() {}
// func (m *mockShareService) View()   {}

// type mockCache struct{}

// func (m *mockCache) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
// 	return nil
// }
// func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
// 	return "", errors.New("not found")
// }

// func TestShareHandler_Get_Success(t *testing.T) {
// 	mockSvc := &mockShareService{
// 		getFunc: func(ctx context.Context, user string) ([]domainshare.GifResp, error) {
// 			return []domainshare.GifResp{}, nil
// 		},
// 	}

// 	jwtProvider := jwtpkg.NewJwt([]byte("test-secret"))
// 	middlewares := middleware.NewMiddlewares(nil, &mockCache{}, *jwtProvider)
// 	val := validator.New()

// 	handler := share.NewHandler(mockSvc, middlewares, val)
// 	tokenStr, _ := jwtProvider.Create("Jane", "user-uuid-1", "jane@example.com", "user")

// 	req := httptest.NewRequest(http.MethodGet, "/gifs/me/1/shares", nil)
// 	req.Header.Set("Authorization", "Bearer "+tokenStr)
// 	rec := httptest.NewRecorder()

// 	middlewares.Auth(http.HandlerFunc(handler.Get)).ServeHTTP(rec, req)

// 	if rec.Code != http.StatusOK {
// 		t.Errorf("expected status 200 OK, got %d. Body: %s", rec.Code, rec.Body.String())
// 	}
// }

// func TestShareHandler_Get_Unauthenticated(t *testing.T) {
// 	mockSvc := &mockShareService{}
// 	val := validator.New()
// 	handler := share.NewHandler(mockSvc, nil, val)

// 	req := httptest.NewRequest(http.MethodGet, "/gifs/me/1/shares", nil)
// 	rec := httptest.NewRecorder()

// 	// Calling directly without Auth middleware should return 401
// 	handler.Get(rec, req)

// 	if rec.Code != http.StatusUnauthorized {
// 		t.Errorf("expected status 401 Unauthorized, got %d", rec.Code)
// 	}
// }
