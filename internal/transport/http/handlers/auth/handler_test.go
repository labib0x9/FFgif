package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	appauth "github.com/labib0x9/ffgif/internal/app/auth"
	domainauth "github.com/labib0x9/ffgif/internal/domain/auth"
	authhandler "github.com/labib0x9/ffgif/internal/transport/http/handlers/auth"
	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
)

type mockAuthService struct {
	signupFunc            func(ctx context.Context, email string, username string, fullname string, password string) (*appauth.SignupResult, error)
	loginFunc             func(ctx context.Context, email string, password string) (*appauth.Result, error)
	verifyFunc            func(ctx context.Context, token string) error
	forgotPasswordFunc    func(ctx context.Context, email string) error
	resendVerifyFunc      func(ctx context.Context, email string) error
	resetPasswordGetFunc  func(ctx context.Context, token string) (string, error)
	resetPasswordPostFunc func(ctx context.Context, token string, pass string, confirmPass string) error
	logoutFunc            func(ctx context.Context, jwt string, claims jwtpkg.Payload) error
}

func (m *mockAuthService) Signup(ctx context.Context, email string, username string, fullname string, password string) (*appauth.SignupResult, error) {
	if m.signupFunc != nil {
		return m.signupFunc(ctx, email, username, fullname, password)
	}
	return &appauth.SignupResult{}, nil
}
func (m *mockAuthService) Login(ctx context.Context, email string, password string) (*appauth.Result, error) {
	if m.loginFunc != nil {
		return m.loginFunc(ctx, email, password)
	}
	return &appauth.Result{Token: "test.jwt.token", Id: uuid.New()}, nil
}
func (m *mockAuthService) Verify(ctx context.Context, token string) error {
	if m.verifyFunc != nil {
		return m.verifyFunc(ctx, token)
	}
	return nil
}
func (m *mockAuthService) ForgotPassword(ctx context.Context, email string) error {
	if m.forgotPasswordFunc != nil {
		return m.forgotPasswordFunc(ctx, email)
	}
	return nil
}
func (m *mockAuthService) ResendVerify(ctx context.Context, email string) error {
	if m.resendVerifyFunc != nil {
		return m.resendVerifyFunc(ctx, email)
	}
	return nil
}
func (m *mockAuthService) ResetPasswordGet(ctx context.Context, token string) (string, error) {
	if m.resetPasswordGetFunc != nil {
		return m.resetPasswordGetFunc(ctx, token)
	}
	return "user@example.com", nil
}
func (m *mockAuthService) ResetPasswordPost(ctx context.Context, token string, pass string, confirmPass string) error {
	if m.resetPasswordPostFunc != nil {
		return m.resetPasswordPostFunc(ctx, token, pass, confirmPass)
	}
	return nil
}
func (m *mockAuthService) Logout(ctx context.Context, jwt string, claims jwtpkg.Payload) error {
	if m.logoutFunc != nil {
		return m.logoutFunc(ctx, jwt, claims)
	}
	return nil
}

func TestAuthHandler_Signup_Success(t *testing.T) {
	mockSvc := &mockAuthService{}
	val := validator.New()
	handler := authhandler.NewHandler(mockSvc, nil, val)

	body, _ := json.Marshal(map[string]string{
		"username":         "validuser",
		"fullname":         "Valid User",
		"email":            "user@example.com",
		"password":         "Password123!",
		"confirm_password": "Password123!",
	})

	req := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Signup(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201 Created, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthHandler_Signup_Conflict(t *testing.T) {
	mockSvc := &mockAuthService{
		signupFunc: func(ctx context.Context, email string, username string, fullname string, password string) (*appauth.SignupResult, error) {
			return nil, domainauth.ErrUserExits
		},
	}
	val := validator.New()
	handler := authhandler.NewHandler(mockSvc, nil, val)

	body, _ := json.Marshal(map[string]string{
		"username":         "existinguser",
		"fullname":         "Existing User",
		"email":            "exists@example.com",
		"password":         "Password123!",
		"confirm_password": "Password123!",
	})

	req := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Signup(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected status 409 Conflict, got %d", rec.Code)
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	mockSvc := &mockAuthService{
		loginFunc: func(ctx context.Context, email string, password string) (*appauth.Result, error) {
			return &appauth.Result{Token: "sample.jwt.token"}, nil
		},
	}
	val := validator.New()
	handler := authhandler.NewHandler(mockSvc, nil, val)

	body, _ := json.Marshal(map[string]string{
		"email":    "user@example.com",
		"password": "Password123!",
	})

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}
