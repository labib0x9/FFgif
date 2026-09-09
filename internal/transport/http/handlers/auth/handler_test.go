package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/labib0x9/ffgif/config"
	appauth "github.com/labib0x9/ffgif/internal/app/auth"
	authsvcmocks "github.com/labib0x9/ffgif/internal/app/auth/mocks"
	domainauth "github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/transport/http/handlers/auth"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
	"github.com/labib0x9/ffgif/pkg/apperr"
	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
)

type authHarness struct {
	svc *authsvcmocks.MockService
	h   *auth.Handler
}

func newAuthHarness(t *testing.T) *authHarness {
	t.Helper()
	ctrl := gomock.NewController(t)
	svc := authsvcmocks.NewMockService(ctrl)
	mws := middleware.NewMiddlewares(&config.Config{}, nil, jwtpkg.Jwt{})
	return &authHarness{svc: svc, h: auth.NewHandler(svc, mws, validator.New())}
}

func post(target, body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

// ===========================================================================
// Signup
// ===========================================================================

func TestSignup_ForwardsTheDecodedFieldsInTheRightOrder(t *testing.T) {
	h := newAuthHarness(t)

	// Signup's parameter list is (email, username, fullname, password) — an
	// easy pair to transpose. Match each one exactly.
	h.svc.EXPECT().
		Signup(gomock.Any(),
			gomock.Eq("alice@example.com"),
			gomock.Eq("alice99"),
			gomock.Eq("Alice Anderson"),
			gomock.Eq("s3cret!pass")).
		Return(&appauth.SignupResult{}, nil).
		Times(1)

	rec := httptest.NewRecorder()
	h.h.Signup(rec, post("/signup", `{
		"username":"alice99","fullname":"Alice Anderson",
		"email":"alice@example.com","password":"s3cret!pass",
		"confirm_password":"s3cret!pass"}`))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}
}

func TestSignup_Validation(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{"confirmation does not match", `{"username":"alice99","fullname":"Alice A","email":"a@example.com","password":"s3cret!pass","confirm_password":"different!"}`, http.StatusUnprocessableEntity},
		{"password with no special character", `{"username":"alice99","fullname":"Alice A","email":"a@example.com","password":"plainpassword","confirm_password":"plainpassword"}`, http.StatusUnprocessableEntity},
		{"password too short", `{"username":"alice99","fullname":"Alice A","email":"a@example.com","password":"a!","confirm_password":"a!"}`, http.StatusUnprocessableEntity},
		{"malformed email", `{"username":"alice99","fullname":"Alice A","email":"not-an-email","password":"s3cret!pass","confirm_password":"s3cret!pass"}`, http.StatusUnprocessableEntity},
		{"non-alphanumeric username", `{"username":"alice 99!","fullname":"Alice A","email":"a@example.com","password":"s3cret!pass","confirm_password":"s3cret!pass"}`, http.StatusUnprocessableEntity},
		{"username too short", `{"username":"ab","fullname":"Alice A","email":"a@example.com","password":"s3cret!pass","confirm_password":"s3cret!pass"}`, http.StatusUnprocessableEntity},
		{"empty body", ``, http.StatusBadRequest},
		{"malformed json", `{"username":`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newAuthHarness(t)
			// the service must not be reached
			rec := httptest.NewRecorder()
			h.h.Signup(rec, post("/signup", tc.body))
			if rec.Code != tc.wantCode {
				t.Errorf("status = %d, want %d; body = %s", rec.Code, tc.wantCode, rec.Body.String())
			}
		})
	}
}

func TestSignup_DuplicateEmailIsConflict(t *testing.T) {
	h := newAuthHarness(t)
	h.svc.EXPECT().
		Signup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, domainauth.ErrUserExists).
		Times(1)

	rec := httptest.NewRecorder()
	h.h.Signup(rec, post("/signup", `{"username":"alice99","fullname":"Alice A","email":"a@example.com","password":"s3cret!pass","confirm_password":"s3cret!pass"}`))

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", rec.Code)
	}
}

// The account exists even if the verification email could not be queued, so a
// queue failure must still report success and let ResendVerify recover.
func TestSignup_QueueFailureStillReportsCreated(t *testing.T) {
	h := newAuthHarness(t)
	h.svc.EXPECT().
		Signup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, apperr.ErrMessageQueueFailed).
		Times(1)

	rec := httptest.NewRecorder()
	h.h.Signup(rec, post("/signup", `{"username":"alice99","fullname":"Alice A","email":"a@example.com","password":"s3cret!pass","confirm_password":"s3cret!pass"}`))

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", rec.Code)
	}
}

// EXPECTED TO FAIL: internal/transport/http/handlers/auth/signup.go:50 builds
// the header as
//
//	w.Header().Set("Location", "/users/"+"res.Id")
//
// concatenating the literal string "res.Id" instead of the created user's id.
// Every signup returns `Location: /users/res.Id`, which points at nothing.
func TestSignup_LocationHeaderPointsAtTheCreatedUser(t *testing.T) {
	h := newAuthHarness(t)
	h.svc.EXPECT().
		Signup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&appauth.SignupResult{}, nil).
		Times(1)

	rec := httptest.NewRecorder()
	h.h.Signup(rec, post("/signup", `{"username":"alice99","fullname":"Alice A","email":"a@example.com","password":"s3cret!pass","confirm_password":"s3cret!pass"}`))

	loc := rec.Header().Get("Location")
	if loc == "/users/res.Id" {
		t.Errorf(`Location = %q — the handler concatenates the literal string "res.Id" `+
			`instead of interpolating the new user's identifier`, loc)
	}
	if strings.Contains(loc, "res.Id") {
		t.Errorf("Location = %q contains a Go expression that was never evaluated", loc)
	}
}

// ===========================================================================
// Login
// ===========================================================================

func TestLogin_SuccessReturnsTokenAndId(t *testing.T) {
	h := newAuthHarness(t)
	id := uuid.New()

	h.svc.EXPECT().
		Login(gomock.Any(), gomock.Eq("alice@example.com"), gomock.Eq("s3cret!pass")).
		Return(&appauth.Result{Token: "the.jwt.token", Id: id}, nil).
		Times(1)

	rec := httptest.NewRecorder()
	h.h.Login(rec, post("/login", `{"email":"alice@example.com","password":"s3cret!pass"}`))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["token"] != "the.jwt.token" {
		t.Errorf("token = %v, want the.jwt.token", body["token"])
	}
	if body["id"] != id.String() {
		t.Errorf("id = %v, want %s", body["id"], id)
	}
}

func TestLogin_ErrorMapping(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{"bad credentials", domainauth.ErrInvalidCredential, http.StatusUnauthorized},
		{"unverified account", domainauth.ErrUserNotVerified, http.StatusForbidden},
		{"unexpected failure", errors.New("db down"), http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newAuthHarness(t)
			h.svc.EXPECT().
				Login(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, tc.err).
				Times(1)

			rec := httptest.NewRecorder()
			h.h.Login(rec, post("/login", `{"email":"a@example.com","password":"s3cret!pass"}`))

			if rec.Code != tc.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantCode)
			}
			if strings.Contains(rec.Body.String(), "the.jwt.token") {
				t.Error("a token was leaked in an error response")
			}
		})
	}
}

// A failed login must never disclose whether the account exists.
func TestLogin_FailureDoesNotDiscloseAccountExistence(t *testing.T) {
	h := newAuthHarness(t)
	h.svc.EXPECT().
		Login(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, domainauth.ErrInvalidCredential).
		Times(1)

	rec := httptest.NewRecorder()
	h.h.Login(rec, post("/login", `{"email":"ghost@example.com","password":"s3cret!pass"}`))

	body := strings.ToLower(rec.Body.String())
	for _, leak := range []string{"not found", "no such user", "unknown email", "does not exist"} {
		if strings.Contains(body, leak) {
			t.Errorf("the response distinguishes a missing account: %q appears in %s", leak, rec.Body.String())
		}
	}
}

func TestLogin_Validation(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{"missing password", `{"email":"a@example.com"}`, http.StatusUnprocessableEntity},
		{"missing email", `{"password":"s3cret!pass"}`, http.StatusUnprocessableEntity},
		{"malformed email", `{"email":"nope","password":"s3cret!pass"}`, http.StatusUnprocessableEntity},
		{"malformed json", `{"email":`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newAuthHarness(t)
			// the service must not be reached
			rec := httptest.NewRecorder()
			h.h.Login(rec, post("/login", tc.body))
			if rec.Code != tc.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantCode)
			}
		})
	}
}

// ===========================================================================
// Logout
// ===========================================================================

func TestLogout_PassesTheCallersOwnTokenAndClaims(t *testing.T) {
	h := newAuthHarness(t)
	exp := time.Now().Add(time.Hour)
	claims := jwtpkg.Payload{
		Fullname: "Alice", Email: "a@example.com", Role: "user",
		RegisteredClaims: gojwt.RegisteredClaims{
			Subject: "user-1", ExpiresAt: gojwt.NewNumericDate(exp),
		},
	}

	h.svc.EXPECT().
		Logout(gomock.Any(), gomock.Eq("the.raw.jwt"), gomock.Any()).
		DoAndReturn(func(_ context.Context, token string, got jwtpkg.Payload) error {
			if got.Subject != "user-1" {
				t.Errorf("claims.Subject = %q, want user-1", got.Subject)
			}
			if got.ExpiresAt == nil || !got.ExpiresAt.Time.Equal(exp.Truncate(time.Second)) {
				// jwt.NewNumericDate truncates to the second
				if got.ExpiresAt == nil {
					t.Error("claims.ExpiresAt is nil; the blocklist TTL cannot be computed")
				}
			}
			return nil
		}).
		Times(1)

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req = req.WithContext(httputil.WithAuthContext(req.Context(), claims, "the.raw.jwt"))

	rec := httptest.NewRecorder()
	h.h.Logout(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}

// Without an auth context there is nothing to revoke; the service must not be
// called with a zero-value token, which would blocklist the key
// "token_blocklist:" for every user at once.
func TestLogout_WithoutAuthContextDoesNotCallTheService(t *testing.T) {
	h := newAuthHarness(t)
	// Logout must not be called.
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	rec := httptest.NewRecorder()
	h.h.Logout(rec, req)

	if rec.Code == http.StatusOK {
		t.Errorf("status = %d; an unauthenticated logout was treated as a success", rec.Code)
	}
}

// A failed blocklist write must not be reported as a successful logout.
func TestLogout_ServiceFailureIsNotReportedAsSuccess(t *testing.T) {
	h := newAuthHarness(t)
	claims := jwtpkg.Payload{RegisteredClaims: gojwt.RegisteredClaims{
		Subject: "user-1", ExpiresAt: gojwt.NewNumericDate(time.Now().Add(time.Hour)),
	}}

	h.svc.EXPECT().
		Logout(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errors.New("redis down")).
		Times(1)

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req = req.WithContext(httputil.WithAuthContext(req.Context(), claims, "tok"))
	rec := httptest.NewRecorder()
	h.h.Logout(rec, req)

	if rec.Code == http.StatusOK {
		t.Error("the user was told they were logged out while their token stays valid")
	}
}

// ===========================================================================
// Verify / ResendVerify / ForgotPassword / ResetPassword
// ===========================================================================

func TestVerify_ForwardsTheTokenFromTheQueryString(t *testing.T) {
	h := newAuthHarness(t)

	h.svc.EXPECT().
		Verify(gomock.Any(), gomock.Eq("the-verify-token")).
		Return(nil).
		Times(1)

	req := httptest.NewRequest(http.MethodGet, "/verify?token=the-verify-token", nil)
	rec := httptest.NewRecorder()
	h.h.Verify(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}

func TestVerify_RejectsAMissingOrInvalidToken(t *testing.T) {
	t.Run("no token", func(t *testing.T) {
		h := newAuthHarness(t)
		// the service must not be reached
		rec := httptest.NewRecorder()
		h.h.Verify(rec, httptest.NewRequest(http.MethodGet, "/verify", nil))
		if rec.Code == http.StatusOK {
			t.Errorf("status = %d; an empty verification token was accepted", rec.Code)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		h := newAuthHarness(t)
		h.svc.EXPECT().
			Verify(gomock.Any(), gomock.Eq("bogus")).
			Return(domainauth.ErrInvalidToken).
			Times(1)

		rec := httptest.NewRecorder()
		h.h.Verify(rec, httptest.NewRequest(http.MethodGet, "/verify?token=bogus", nil))
		if rec.Code == http.StatusOK {
			t.Error("an invalid verification token was accepted")
		}
	})
}

// A password-reset request must answer identically whether or not the address
// is registered, or it becomes an account-enumeration oracle.
func TestForgotPassword_ResponseIsIdenticalForKnownAndUnknownAddresses(t *testing.T) {
	capture := func(t *testing.T, svcErr error) (int, string) {
		t.Helper()
		h := newAuthHarness(t)
		h.svc.EXPECT().
			ForgotPassword(gomock.Any(), gomock.Eq("probe@example.com")).
			Return(svcErr).
			Times(1)

		rec := httptest.NewRecorder()
		h.h.ForgotPassword(rec, post("/forgot-password", `{"email":"probe@example.com"}`))
		return rec.Code, rec.Body.String()
	}

	knownCode, knownBody := capture(t, nil)
	unknownCode, unknownBody := capture(t, domainauth.ErrUserNotFound)

	if knownCode != unknownCode || knownBody != unknownBody {
		t.Errorf("the endpoint distinguishes registered from unregistered addresses:\n"+
			"  registered:   %d %s\n  unregistered: %d %s\n"+
			"this lets an attacker enumerate which emails hold accounts",
			knownCode, knownBody, unknownCode, unknownBody)
	}
}

func TestResetPasswordPost_MismatchedConfirmationIsRejectedAtTheEdge(t *testing.T) {
	h := newAuthHarness(t)
	// the service must not be reached: the eqfield tag stops this at the handler
	rec := httptest.NewRecorder()
	h.h.ResetPasswordPost(rec, post("/reset-password",
		`{"token":"tok","password":"newpass!","confirm_password":"different!"}`))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}

func TestResetPasswordPost_ExpiredTokenIsGone(t *testing.T) {
	h := newAuthHarness(t)
	h.svc.EXPECT().
		ResetPasswordPost(gomock.Any(), gomock.Eq("tok"), gomock.Eq("newpass!"), gomock.Eq("newpass!")).
		Return(domainauth.ErrReseterTokenFatchFailed).
		Times(1)

	rec := httptest.NewRecorder()
	h.h.ResetPasswordPost(rec, post("/reset-password",
		`{"token":"tok","password":"newpass!","confirm_password":"newpass!"}`))

	if rec.Code != http.StatusGone {
		t.Errorf("status = %d, want 410", rec.Code)
	}
}

func TestResendVerify_ForwardsTheEmail(t *testing.T) {
	h := newAuthHarness(t)
	h.svc.EXPECT().
		ResendVerify(gomock.Any(), gomock.Eq("alice@example.com")).
		Return(nil).
		Times(1)

	rec := httptest.NewRecorder()
	h.h.ResendVerify(rec, post("/resend-verify", `{"email":"alice@example.com"}`))

	if rec.Code != http.StatusOK && rec.Code != http.StatusAccepted {
		t.Errorf("status = %d, want 200 or 202; body = %s", rec.Code, rec.Body.String())
	}
}
