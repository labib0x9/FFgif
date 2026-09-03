package user_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	domainauth "github.com/labib0x9/ffgif/internal/domain/auth"
	domainuser "github.com/labib0x9/ffgif/internal/domain/user"
	"github.com/labib0x9/ffgif/internal/transport/http/handlers/user"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
)

type mockUserService struct {
	getProfileFunc     func(ctx context.Context, id string) (*domainuser.ProfileResponse, error)
	updateProfileFunc  func(ctx context.Context, profile domainuser.ProfileResponse, id string) (domainuser.ProfileResponse, error)
	changePasswordFunc func(ctx context.Context, id string, currentPass string, pass string, confirmPass string) error
	deleteUserFunc     func(ctx context.Context, id string, pass string) error
	getQuotaFunc       func(ctx context.Context, id string) (*domainuser.Quota, error)
}

func (m *mockUserService) GetProfile(ctx context.Context, id string) (*domainuser.ProfileResponse, error) {
	if m.getProfileFunc != nil {
		return m.getProfileFunc(ctx, id)
	}
	return &domainuser.ProfileResponse{Username: "testuser", Email: "test@example.com"}, nil
}
func (m *mockUserService) UpdateProfile(ctx context.Context, profile domainuser.ProfileResponse, id string) (domainuser.ProfileResponse, error) {
	if m.updateProfileFunc != nil {
		return m.updateProfileFunc(ctx, profile, id)
	}
	return profile, nil
}
func (m *mockUserService) ChangePassword(ctx context.Context, id string, currentPass string, pass string, confirmPass string) error {
	if m.changePasswordFunc != nil {
		return m.changePasswordFunc(ctx, id, currentPass, pass, confirmPass)
	}
	return nil
}
func (m *mockUserService) DeleteUser(ctx context.Context, id string, pass string) error {
	if m.deleteUserFunc != nil {
		return m.deleteUserFunc(ctx, id, pass)
	}
	return nil
}
func (m *mockUserService) GetQuota(ctx context.Context, id string) (*domainuser.Quota, error) {
	if m.getQuotaFunc != nil {
		return m.getQuotaFunc(ctx, id)
	}
	return &domainuser.Quota{UserID: uuid.New(), TotalBytes: 1000, GifCount: 10}, nil
}

func withUserAuth(r *http.Request, userId string) *http.Request {
	claims := jwtpkg.Payload{
		Fullname: "Jane Doe",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: userId,
		},
	}
	ctx := httputil.WithAuthContext(r.Context(), claims, "test-token")
	return r.WithContext(ctx)
}

func TestUserHandler_GetProfile_Success(t *testing.T) {
	mockSvc := &mockUserService{
		getProfileFunc: func(ctx context.Context, id string) (*domainuser.ProfileResponse, error) {
			return &domainuser.ProfileResponse{
				Username: "janedoe",
				Email:    "jane@example.com",
				Fullname: "Jane Doe",
			}, nil
		},
	}
	handler := user.NewHandler(mockSvc, nil, validator.New())

	req := httptest.NewRequest(http.MethodGet, "/users/profile/me", nil)
	req = withUserAuth(req, "user-uuid-1")
	rec := httptest.NewRecorder()

	handler.GetProfile(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUserHandler_GetProfile_Unauthenticated(t *testing.T) {
	handler := user.NewHandler(&mockUserService{}, nil, validator.New())

	req := httptest.NewRequest(http.MethodGet, "/users/profile/me", nil)
	rec := httptest.NewRecorder()

	handler.GetProfile(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized, got %d", rec.Code)
	}
}

func TestUserHandler_GetProfile_NotFound(t *testing.T) {
	mockSvc := &mockUserService{
		getProfileFunc: func(ctx context.Context, id string) (*domainuser.ProfileResponse, error) {
			return nil, domainauth.ErrUserNotFound
		},
	}
	handler := user.NewHandler(mockSvc, nil, validator.New())

	req := httptest.NewRequest(http.MethodGet, "/users/profile/me", nil)
	req = withUserAuth(req, "missing-user")
	rec := httptest.NewRecorder()

	handler.GetProfile(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404 Not Found, got %d", rec.Code)
	}
}

func TestUserHandler_UpdateProfile_Success(t *testing.T) {
	mockSvc := &mockUserService{
		updateProfileFunc: func(ctx context.Context, profile domainuser.ProfileResponse, id string) (domainuser.ProfileResponse, error) {
			return profile, nil
		},
	}
	handler := user.NewHandler(mockSvc, nil, validator.New())

	body, _ := json.Marshal(map[string]string{
		"username":  "janedoe_new",
		"full_name": "Jane New",
	})
	req := httptest.NewRequest(http.MethodPatch, "/users/profile/me", bytes.NewReader(body))
	req = withUserAuth(req, "user-uuid-1")
	rec := httptest.NewRecorder()

	handler.UpdateProfile(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestUserHandler_UpdateProfile_BadJSON(t *testing.T) {
	handler := user.NewHandler(&mockUserService{}, nil, validator.New())

	req := httptest.NewRequest(http.MethodPatch, "/users/profile/me", bytes.NewReader([]byte("{bad")))
	req = withUserAuth(req, "user-uuid-1")
	rec := httptest.NewRecorder()

	handler.UpdateProfile(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 Bad Request, got %d", rec.Code)
	}
}

func TestUserHandler_GetQuota_Success(t *testing.T) {
	mockSvc := &mockUserService{
		getQuotaFunc: func(ctx context.Context, id string) (*domainuser.Quota, error) {
			return &domainuser.Quota{
				GifCount:   25,
				TotalBytes: 1024 * 1024,
			}, nil
		},
	}
	handler := user.NewHandler(mockSvc, nil, validator.New())

	req := httptest.NewRequest(http.MethodGet, "/users/me/quota", nil)
	req = withUserAuth(req, "user-uuid-1")
	rec := httptest.NewRecorder()

	handler.GetQuota(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestUserHandler_GetQuota_Unauthenticated(t *testing.T) {
	handler := user.NewHandler(&mockUserService{}, nil, validator.New())

	req := httptest.NewRequest(http.MethodGet, "/users/me/quota", nil)
	rec := httptest.NewRecorder()

	handler.GetQuota(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized, got %d", rec.Code)
	}
}

func TestUserHandler_ChangePassword_Success(t *testing.T) {
	mockSvc := &mockUserService{
		changePasswordFunc: func(ctx context.Context, id string, currentPass string, pass string, confirmPass string) error {
			return nil
		},
	}
	handler := user.NewHandler(mockSvc, nil, validator.New())

	body, _ := json.Marshal(map[string]string{
		"current_password": "OldPassword123!",
		"password":         "NewPassword123!",
		"confirm_password": "NewPassword123!",
	})
	req := httptest.NewRequest(http.MethodPatch, "/users/change-password", bytes.NewReader(body))
	req = withUserAuth(req, "user-uuid-1")
	rec := httptest.NewRecorder()

	handler.ChangePassword(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestUserHandler_ChangePassword_Mismatch(t *testing.T) {
	handler := user.NewHandler(&mockUserService{}, nil, validator.New())

	body, _ := json.Marshal(map[string]string{
		"current_password": "OldPassword123!",
		"password":         "NewPassword123!",
		"confirm_password": "DifferentPassword123!",
	})
	req := httptest.NewRequest(http.MethodPatch, "/users/change-password", bytes.NewReader(body))
	req = withUserAuth(req, "user-uuid-1")
	rec := httptest.NewRecorder()

	handler.ChangePassword(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422 Unprocessable Entity, got %d", rec.Code)
	}
}

func TestUserHandler_ChangePassword_WrongCurrentPassword(t *testing.T) {
	mockSvc := &mockUserService{
		changePasswordFunc: func(ctx context.Context, id string, currentPass string, pass string, confirmPass string) error {
			return domainauth.ErrInvalidCredential
		},
	}
	handler := user.NewHandler(mockSvc, nil, validator.New())

	body, _ := json.Marshal(map[string]string{
		"current_password": "WrongPassword123!",
		"password":         "NewPassword123!",
		"confirm_password": "NewPassword123!",
	})
	req := httptest.NewRequest(http.MethodPatch, "/users/change-password", bytes.NewReader(body))
	req = withUserAuth(req, "user-uuid-1")
	rec := httptest.NewRecorder()

	handler.ChangePassword(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized on invalid credential, got %d", rec.Code)
	}
}

func TestUserHandler_DeleteUser_Success(t *testing.T) {
	mockSvc := &mockUserService{
		deleteUserFunc: func(ctx context.Context, id string, pass string) error {
			return nil
		},
	}
	handler := user.NewHandler(mockSvc, nil, validator.New())

	body, _ := json.Marshal(map[string]string{
		"password": "Password123!",
	})
	req := httptest.NewRequest(http.MethodDelete, "/users/me", bytes.NewReader(body))
	req = withUserAuth(req, "user-uuid-1")
	rec := httptest.NewRecorder()

	handler.DeleteUser(rec, req)
	if rec.Code != http.StatusGone {
		t.Errorf("expected status 410 Gone, got %d", rec.Code)
	}
}

func TestUserHandler_DeleteUser_InvalidPassword(t *testing.T) {
	mockSvc := &mockUserService{
		deleteUserFunc: func(ctx context.Context, id string, pass string) error {
			return domainauth.ErrInvalidCredential
		},
	}
	handler := user.NewHandler(mockSvc, nil, validator.New())

	body, _ := json.Marshal(map[string]string{
		"password": "WrongPassword123!",
	})
	req := httptest.NewRequest(http.MethodDelete, "/users/me", bytes.NewReader(body))
	req = withUserAuth(req, "user-uuid-1")
	rec := httptest.NewRecorder()

	handler.DeleteUser(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized, got %d", rec.Code)
	}
}

func TestUserHandler_DeleteUser_UserNotFound(t *testing.T) {
	mockSvc := &mockUserService{
		deleteUserFunc: func(ctx context.Context, id string, pass string) error {
			return domainauth.ErrUserNotFound
		},
	}
	handler := user.NewHandler(mockSvc, nil, validator.New())

	body, _ := json.Marshal(map[string]string{
		"password": "Password123!",
	})
	req := httptest.NewRequest(http.MethodDelete, "/users/me", bytes.NewReader(body))
	req = withUserAuth(req, "user-uuid-1")
	rec := httptest.NewRecorder()

	handler.DeleteUser(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404 Not Found, got %d", rec.Code)
	}
}
