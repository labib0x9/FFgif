package user_test

// import (
// 	"context"
// 	"encoding/json"
// 	"errors"
// 	"net/http"
// 	"net/http/httptest"
// 	"testing"
// 	"time"

// 	"github.com/go-playground/validator/v10"
// 	"github.com/google/uuid"
// 	domainuser "github.com/labib0x9/ffgif/internal/domain/user"
// 	"github.com/labib0x9/ffgif/internal/transport/http/handlers/user"
// 	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
// 	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
// )

// type mockUserService struct {
// 	getProfileFunc     func(ctx context.Context, id string) (domainuser.ProfileResp, error)
// 	updateProfileFunc  func(ctx context.Context, profile domainuser.ProfileResp, id string) (domainuser.ProfileResp, error)
// 	changePasswordFunc func(ctx context.Context, id string, currentPass string, pass string, confirmPass string) error
// 	deleteUserFunc     func(ctx context.Context, id string, pass string) error
// 	getQuotaFunc       func(ctx context.Context, id string) (*domainuser.Quota, error)
// }

// func (m *mockUserService) GetProfile(ctx context.Context, id string) (domainuser.ProfileResp, error) {
// 	if m.getProfileFunc != nil {
// 		return m.getProfileFunc(ctx, id)
// 	}
// 	return domainuser.ProfileResp{Username: "testuser", Email: "test@example.com"}, nil
// }
// func (m *mockUserService) UpdateProfile(ctx context.Context, profile domainuser.ProfileResp, id string) (domainuser.ProfileResp, error) {
// 	if m.updateProfileFunc != nil {
// 		return m.updateProfileFunc(ctx, profile, id)
// 	}
// 	return profile, nil
// }
// func (m *mockUserService) ChangePassword(ctx context.Context, id string, currentPass string, pass string, confirmPass string) error {
// 	if m.changePasswordFunc != nil {
// 		return m.changePasswordFunc(ctx, id, currentPass, pass, confirmPass)
// 	}
// 	return nil
// }
// func (m *mockUserService) DeleteUser(ctx context.Context, id string, pass string) error {
// 	if m.deleteUserFunc != nil {
// 		return m.deleteUserFunc(ctx, id, pass)
// 	}
// 	return nil
// }
// func (m *mockUserService) GetQuota(ctx context.Context, id string) (*domainuser.Quota, error) {
// 	if m.getQuotaFunc != nil {
// 		return m.getQuotaFunc(ctx, id)
// 	}
// 	return &domainuser.Quota{UserID: uuid.New(), TotalBytes: 1000, GifCount: 10}, nil
// }

// type mockCache struct{}

// func (m *mockCache) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
// 	return nil
// }
// func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
// 	return "", errors.New("not found")
// }

// func TestUserHandler_GetProfile_Success(t *testing.T) {
// 	mockSvc := &mockUserService{
// 		getProfileFunc: func(ctx context.Context, id string) (domainuser.ProfileResp, error) {
// 			return domainuser.ProfileResp{
// 				Username: "janedoe",
// 				Email:    "jane@example.com",
// 				Fullname: "Jane Doe",
// 			}, nil
// 		},
// 	}

// 	jwtProvider := jwtpkg.NewJwt([]byte("test-secret"))
// 	middlewares := middleware.NewMiddlewares(nil, &mockCache{}, *jwtProvider)
// 	val := validator.New()

// 	handler := user.NewHandler(mockSvc, middlewares, val)

// 	tokenStr, _ := jwtProvider.Create("Jane", "user-uuid-1", "jane@example.com", "user")

// 	req := httptest.NewRequest(http.MethodGet, "/users/profile/me", nil)
// 	req.Header.Set("Authorization", "Bearer "+tokenStr)
// 	rec := httptest.NewRecorder()

// 	middlewares.Auth(http.HandlerFunc(handler.GetProfile)).ServeHTTP(rec, req)

// 	if rec.Code != http.StatusOK {
// 		t.Errorf("expected status 200 OK, got %d. Body: %s", rec.Code, rec.Body.String())
// 	}

// 	var resp domainuser.ProfileResp
// 	json.NewDecoder(rec.Body).Decode(&resp)
// 	if resp.Username != "janedoe" || resp.Email != "jane@example.com" {
// 		t.Errorf("unexpected profile: %+v", resp)
// 	}
// }

// func TestUserHandler_GetQuota_Success(t *testing.T) {
// 	mockSvc := &mockUserService{
// 		getQuotaFunc: func(ctx context.Context, id string) (*domainuser.Quota, error) {
// 			return &domainuser.Quota{
// 				GifCount:   25,
// 				TotalBytes: 1024 * 1024,
// 			}, nil
// 		},
// 	}

// 	jwtProvider := jwtpkg.NewJwt([]byte("test-secret"))
// 	middlewares := middleware.NewMiddlewares(nil, &mockCache{}, *jwtProvider)
// 	val := validator.New()

// 	handler := user.NewHandler(mockSvc, middlewares, val)
// 	tokenStr, _ := jwtProvider.Create("Jane", "user-uuid-1", "jane@example.com", "user")

// 	req := httptest.NewRequest(http.MethodGet, "/users/me/quota", nil)
// 	req.Header.Set("Authorization", "Bearer "+tokenStr)
// 	rec := httptest.NewRecorder()

// 	middlewares.Auth(http.HandlerFunc(handler.GetQuota)).ServeHTTP(rec, req)

// 	if rec.Code != http.StatusOK {
// 		t.Errorf("expected status 200 OK, got %d", rec.Code)
// 	}
// }
