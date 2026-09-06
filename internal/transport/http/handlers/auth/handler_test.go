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
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
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

func TestAuthHandler_Signup_BadJSON(t *testing.T) {
	handler := authhandler.NewHandler(&mockAuthService{}, nil, validator.New())
	req := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader([]byte("{invalid-json")))
	rec := httptest.NewRecorder()

	handler.Signup(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 Bad Request, got %d", rec.Code)
	}
}

func TestAuthHandler_Signup_ValidationFailed(t *testing.T) {
	handler := authhandler.NewHandler(&mockAuthService{}, nil, validator.New())

	// password mismatch
	body, _ := json.Marshal(map[string]string{
		"username":         "user1",
		"fullname":         "User One",
		"email":            "invalid-email",
		"password":         "Pass1",
		"confirm_password": "Pass2",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Signup(rec, req)
	if rec.Code != 422 {
		t.Errorf("expected status 422 Unprocessable Entity, got %d", rec.Code)
	}
}

func TestAuthHandler_Signup_Conflict(t *testing.T) {
	mockSvc := &mockAuthService{
		signupFunc: func(ctx context.Context, email string, username string, fullname string, password string) (*appauth.SignupResult, error) {
			return nil, domainauth.ErrUserExists
		},
	}
	handler := authhandler.NewHandler(mockSvc, nil, validator.New())

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
	handler := authhandler.NewHandler(mockSvc, nil, validator.New())

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

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	mockSvc := &mockAuthService{
		loginFunc: func(ctx context.Context, email string, password string) (*appauth.Result, error) {
			return nil, domainauth.ErrInvalidCredential
		},
	}
	handler := authhandler.NewHandler(mockSvc, nil, validator.New())

	body, _ := json.Marshal(map[string]string{
		"email":    "user@example.com",
		"password": "WrongPassword123!",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Login(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized, got %d", rec.Code)
	}
}

func TestAuthHandler_Login_Unverified(t *testing.T) {
	mockSvc := &mockAuthService{
		loginFunc: func(ctx context.Context, email string, password string) (*appauth.Result, error) {
			return nil, domainauth.ErrUserNotVerified
		},
	}
	handler := authhandler.NewHandler(mockSvc, nil, validator.New())

	body, _ := json.Marshal(map[string]string{
		"email":    "unverified@example.com",
		"password": "Password123!",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Login(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status 403 Forbidden for unverified user, got %d", rec.Code)
	}
}

func TestAuthHandler_Logout_Success(t *testing.T) {
	mockSvc := &mockAuthService{
		logoutFunc: func(ctx context.Context, token string, claims jwtpkg.Payload) error {
			return nil
		},
	}
	handler := authhandler.NewHandler(mockSvc, nil, validator.New())

	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	ctx := httputil.WithAuthContext(req.Context(), jwtpkg.Payload{Fullname: "User"}, "token-xyz")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.Logout(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestAuthHandler_Logout_Unauthenticated(t *testing.T) {
	handler := authhandler.NewHandler(&mockAuthService{}, nil, validator.New())
	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	rec := httptest.NewRecorder()

	handler.Logout(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 when missing auth context, got %d", rec.Code)
	}
}

func TestAuthHandler_Verify_Success(t *testing.T) {
	mockSvc := &mockAuthService{
		verifyFunc: func(ctx context.Context, token string) error {
			return nil
		},
	}
	handler := authhandler.NewHandler(mockSvc, nil, validator.New())

	req := httptest.NewRequest(http.MethodGet, "/auth/verify?token=valid-token", nil)
	rec := httptest.NewRecorder()

	handler.Verify(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestAuthHandler_Verify_MissingToken(t *testing.T) {
	handler := authhandler.NewHandler(&mockAuthService{}, nil, validator.New())
	req := httptest.NewRequest(http.MethodGet, "/auth/verify", nil)
	rec := httptest.NewRecorder()

	handler.Verify(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 Bad Request on missing token query, got %d", rec.Code)
	}
}

func TestAuthHandler_Verify_InvalidToken(t *testing.T) {
	mockSvc := &mockAuthService{
		verifyFunc: func(ctx context.Context, token string) error {
			return domainauth.ErrInvalidToken
		},
	}
	handler := authhandler.NewHandler(mockSvc, nil, validator.New())

	req := httptest.NewRequest(http.MethodGet, "/auth/verify?token=bad-token", nil)
	rec := httptest.NewRecorder()

	handler.Verify(rec, req)
	if rec.Code != http.StatusGone {
		t.Errorf("expected status 410 Gone on invalid token, got %d", rec.Code)
	}
}

func TestAuthHandler_ForgotPassword_Success(t *testing.T) {
	mockSvc := &mockAuthService{
		forgotPasswordFunc: func(ctx context.Context, email string) error {
			return nil
		},
	}
	handler := authhandler.NewHandler(mockSvc, nil, validator.New())

	body, _ := json.Marshal(map[string]string{"email": "user@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ForgotPassword(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestAuthHandler_ForgotPassword_UserNotFound(t *testing.T) {
	mockSvc := &mockAuthService{
		forgotPasswordFunc: func(ctx context.Context, email string) error {
			return domainauth.ErrUserNotFound
		},
	}
	handler := authhandler.NewHandler(mockSvc, nil, validator.New())

	body, _ := json.Marshal(map[string]string{"email": "missing@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ForgotPassword(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404 Not Found, got %d", rec.Code)
	}
}

func TestAuthHandler_ResendVerify_Success(t *testing.T) {
	mockSvc := &mockAuthService{
		resendVerifyFunc: func(ctx context.Context, email string) error {
			return nil
		},
	}
	handler := authhandler.NewHandler(mockSvc, nil, validator.New())

	body, _ := json.Marshal(map[string]string{"email": "unverified@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/auth/verify/resend", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ResendVerify(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestAuthHandler_ResendVerify_UserNotFound(t *testing.T) {
	mockSvc := &mockAuthService{
		resendVerifyFunc: func(ctx context.Context, email string) error {
			return domainauth.ErrUserNotFound
		},
	}
	handler := authhandler.NewHandler(mockSvc, nil, validator.New())

	body, _ := json.Marshal(map[string]string{"email": "missing@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/auth/verify/resend", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ResendVerify(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404 Not Found for missing user, got %d", rec.Code)
	}
}

func TestAuthHandler_ResetPasswordGet_Success(t *testing.T) {
	mockSvc := &mockAuthService{
		resetPasswordGetFunc: func(ctx context.Context, token string) (string, error) {
			return "reset-tok-123", nil
		},
	}
	handler := authhandler.NewHandler(mockSvc, nil, validator.New())

	req := httptest.NewRequest(http.MethodGet, "/auth/reset?token=reset-tok-123", nil)
	rec := httptest.NewRecorder()

	handler.ResetPasswordGet(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestAuthHandler_ResetPasswordGet_NotFound(t *testing.T) {
	mockSvc := &mockAuthService{
		resetPasswordGetFunc: func(ctx context.Context, token string) (string, error) {
			return "", domainauth.ErrReseterTokenFatchFailed
		},
	}
	handler := authhandler.NewHandler(mockSvc, nil, validator.New())

	req := httptest.NewRequest(http.MethodGet, "/auth/reset?token=invalid-tok", nil)
	rec := httptest.NewRecorder()

	handler.ResetPasswordGet(rec, req)
	if rec.Code != http.StatusGone {
		t.Errorf("expected status 410 Gone, got %d", rec.Code)
	}
}

func TestAuthHandler_ResetPasswordPost_Success(t *testing.T) {
	mockSvc := &mockAuthService{
		resetPasswordPostFunc: func(ctx context.Context, token string, pass string, confirmPass string) error {
			return nil
		},
	}
	handler := authhandler.NewHandler(mockSvc, nil, validator.New())

	body, _ := json.Marshal(map[string]string{
		"token":            "valid-reset-token",
		"password":         "NewPass123!",
		"confirm_password": "NewPass123!",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/reset", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ResetPasswordPost(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestAuthHandler_ResetPasswordPost_Mismatch(t *testing.T) {
	handler := authhandler.NewHandler(&mockAuthService{}, nil, validator.New())

	body, _ := json.Marshal(map[string]string{
		"token":            "valid-token",
		"password":         "Password123!",
		"confirm_password": "PasswordDifferent!",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/reset", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ResetPasswordPost(rec, req)
	if rec.Code != 422 {
		t.Errorf("expected status 422 Unprocessable Entity, got %d", rec.Code)
	}
}
