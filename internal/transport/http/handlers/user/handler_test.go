package user_test

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
	usersvcmocks "github.com/labib0x9/ffgif/internal/app/user/mocks"
	domainauth "github.com/labib0x9/ffgif/internal/domain/auth"
	domainmedia "github.com/labib0x9/ffgif/internal/domain/media"
	domainuser "github.com/labib0x9/ffgif/internal/domain/user"
	"github.com/labib0x9/ffgif/internal/transport/http/handlers/user"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
	"github.com/labib0x9/ffgif/pkg/apperr"
	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
)

const userID = "33333333-3333-3333-3333-333333333333"

type userHarness struct {
	svc *usersvcmocks.MockService
	h   *user.Handler
}

func newUserHarness(t *testing.T) *userHarness {
	t.Helper()
	ctrl := gomock.NewController(t)
	svc := usersvcmocks.NewMockService(ctrl)
	mws := middleware.NewMiddlewares(&config.Config{}, nil, jwtpkg.Jwt{})
	return &userHarness{svc: svc, h: user.NewHandler(svc, mws, validator.New())}
}

func request(t *testing.T, method, target, body string, withAuth bool) *http.Request {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	if withAuth {
		claims := jwtpkg.Payload{
			Fullname: "Alice", Email: "alice@example.com", Role: "user",
			RegisteredClaims: gojwt.RegisteredClaims{
				Subject:   userID,
				ExpiresAt: gojwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		}
		r = r.WithContext(httputil.WithAuthContext(r.Context(), claims, "tok"))
	}
	return r
}

// ===========================================================================
// ChangePassword
// ===========================================================================

func TestChangePassword_ForwardsAllThreePasswordsAndTheTokenSubject(t *testing.T) {
	h := newUserHarness(t)

	h.svc.EXPECT().
		ChangePassword(gomock.Any(),
			gomock.Eq(userID),
			gomock.Eq("old!pass"),
			gomock.Eq("new!pass"),
			gomock.Eq("new!pass")).
		Return(nil).
		Times(1)

	rec := httptest.NewRecorder()
	h.h.ChangePassword(rec, request(t, http.MethodPost, "/me/password",
		`{"current_password":"old!pass","password":"new!pass","confirm_password":"new!pass"}`, true))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}

// The user id must come from the verified token, never from the request body —
// otherwise anyone can change anyone's password.
func TestChangePassword_IgnoresAUserIdInTheBody(t *testing.T) {
	h := newUserHarness(t)

	h.svc.EXPECT().
		ChangePassword(gomock.Any(), gomock.Eq(userID), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id, _, _, _ string) error {
			if id != userID {
				t.Errorf("the handler used id %q from the request body instead of the token subject %q", id, userID)
			}
			return nil
		}).
		Times(1)

	rec := httptest.NewRecorder()
	h.h.ChangePassword(rec, request(t, http.MethodPost, "/me/password",
		`{"user_id":"victim-user-id","id":"victim-user-id",
		  "current_password":"old!pass","password":"new!pass","confirm_password":"new!pass"}`, true))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}

func TestChangePassword_Validation(t *testing.T) {
	tests := []struct {
		name     string
		withAuth bool
		body     string
		wantCode int
	}{
		{name: "unauthenticated", body: `{"current_password":"old!pass","password":"new!pass","confirm_password":"new!pass"}`, wantCode: http.StatusUnauthorized},
		{name: "confirmation mismatch", withAuth: true, body: `{"current_password":"old!pass","password":"new!pass","confirm_password":"typo!pass"}`, wantCode: http.StatusUnprocessableEntity},
		{name: "new password has no special character", withAuth: true, body: `{"current_password":"old!pass","password":"plainpass","confirm_password":"plainpass"}`, wantCode: http.StatusUnprocessableEntity},
		{name: "missing current password", withAuth: true, body: `{"password":"new!pass","confirm_password":"new!pass"}`, wantCode: http.StatusUnprocessableEntity},
		{name: "new password too short", withAuth: true, body: `{"current_password":"old!pass","password":"a!","confirm_password":"a!"}`, wantCode: http.StatusUnprocessableEntity},
		{name: "malformed json", withAuth: true, body: `{"current_password":`, wantCode: http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newUserHarness(t)
			// the service must not be reached
			rec := httptest.NewRecorder()
			h.h.ChangePassword(rec, request(t, http.MethodPost, "/me/password", tc.body, tc.withAuth))
			if rec.Code != tc.wantCode {
				t.Errorf("status = %d, want %d; body = %s", rec.Code, tc.wantCode, rec.Body.String())
			}
		})
	}
}

func TestChangePassword_ErrorMapping(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{"wrong current password", domainauth.ErrInvalidCredential, http.StatusUnauthorized},
		{"passwords do not match", apperr.ErrPasswordMismatched, http.StatusUnprocessableEntity},
		{"user gone", domainauth.ErrUserNotFound, http.StatusNotFound},
		{"unexpected failure", errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newUserHarness(t)
			h.svc.EXPECT().
				ChangePassword(gomock.Any(), gomock.Eq(userID), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(tc.err).
				Times(1)

			rec := httptest.NewRecorder()
			h.h.ChangePassword(rec, request(t, http.MethodPost, "/me/password",
				`{"current_password":"old!pass","password":"new!pass","confirm_password":"new!pass"}`, true))

			if rec.Code != tc.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantCode)
			}
		})
	}
}

// ===========================================================================
// DeleteUser
// ===========================================================================

func TestDeleteUser_RequiresThePasswordAndUsesTheTokenSubject(t *testing.T) {
	h := newUserHarness(t)

	h.svc.EXPECT().
		DeleteUser(gomock.Any(), gomock.Eq(userID), gomock.Eq("my!password")).
		Return(nil).
		Times(1)

	rec := httptest.NewRecorder()
	h.h.DeleteUser(rec, request(t, http.MethodDelete, "/me", `{"password":"my!password"}`, true))

	if rec.Code != http.StatusOK && rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 200 or 204; body = %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteUser_Rejections(t *testing.T) {
	tests := []struct {
		name     string
		withAuth bool
		body     string
		wantCode int
	}{
		{name: "unauthenticated", body: `{"password":"my!password"}`, wantCode: http.StatusUnauthorized},
		{name: "no password supplied", withAuth: true, body: `{}`, wantCode: http.StatusUnprocessableEntity},
		{name: "empty password", withAuth: true, body: `{"password":""}`, wantCode: http.StatusUnprocessableEntity},
		{name: "malformed json", withAuth: true, body: `{"password":`, wantCode: http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newUserHarness(t)
			// the service must not be reached: deleting an account is irreversible
			rec := httptest.NewRecorder()
			h.h.DeleteUser(rec, request(t, http.MethodDelete, "/me", tc.body, tc.withAuth))
			if rec.Code != tc.wantCode {
				t.Errorf("status = %d, want %d; body = %s", rec.Code, tc.wantCode, rec.Body.String())
			}
		})
	}
}

func TestDeleteUser_WrongPasswordIsUnauthorized(t *testing.T) {
	h := newUserHarness(t)
	h.svc.EXPECT().
		DeleteUser(gomock.Any(), gomock.Eq(userID), gomock.Eq("wrong!pass")).
		Return(domainauth.ErrInvalidCredential).
		Times(1)

	rec := httptest.NewRecorder()
	h.h.DeleteUser(rec, request(t, http.MethodDelete, "/me", `{"password":"wrong!pass"}`, true))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

// ===========================================================================
// GetProfile / UpdateProfile
// ===========================================================================

func TestGetProfile_ScopesToTheCallerAndOmitsSecrets(t *testing.T) {
	h := newUserHarness(t)

	h.svc.EXPECT().
		GetProfile(gomock.Any(), gomock.Eq(userID)).
		Return(&domainuser.ProfileResponse{
			Username: "alice", Fullname: "Alice A", Email: "alice@example.com",
			IsVerified: true, UpdatedAt: time.Now().UTC(),
		}, nil).
		Times(1)

	rec := httptest.NewRecorder()
	h.h.GetProfile(rec, request(t, http.MethodGet, "/me", "", true))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	for _, secret := range []string{"password", "password_hash", "PasswordHash"} {
		if strings.Contains(strings.ToLower(body), strings.ToLower(secret)) {
			t.Errorf("the profile response mentions %q: %s", secret, body)
		}
	}

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["username"] != "alice" {
		t.Errorf("username = %v, want alice", got["username"])
	}
}

func TestGetProfile_Unauthenticated(t *testing.T) {
	h := newUserHarness(t)
	// the service must not be reached
	rec := httptest.NewRecorder()
	h.h.GetProfile(rec, request(t, http.MethodGet, "/me", "", false))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestUpdateProfile_RequiresIfMatchAndForwardsIt(t *testing.T) {
	t.Run("with If-Match", func(t *testing.T) {
		h := newUserHarness(t)
		h.svc.EXPECT().
			UpdateProfile(gomock.Any(), gomock.Any(), gomock.Eq(userID), gomock.Eq("etag-value")).
			DoAndReturn(func(_ context.Context, req domainuser.ProfileUpdateRequest, _, _ string) (*domainuser.ProfileResponse, error) {
				if req.Fullname == nil || *req.Fullname != "New Name" {
					t.Errorf("Fullname = %v, want New Name", req.Fullname)
				}
				return &domainuser.ProfileResponse{Fullname: "New Name"}, nil
			}).
			Times(1)

		r := request(t, http.MethodPatch, "/me", `{"fullname":"New Name"}`, true)
		r.Header.Set("If-Match", "etag-value")
		rec := httptest.NewRecorder()
		h.h.UpdateProfile(rec, r)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("without If-Match", func(t *testing.T) {
		h := newUserHarness(t)
		// the service must not be reached: a blind write could clobber a concurrent edit
		rec := httptest.NewRecorder()
		h.h.UpdateProfile(rec, request(t, http.MethodPatch, "/me", `{"fullname":"New Name"}`, true))

		if rec.Code == http.StatusOK {
			t.Errorf("status = %d; a profile update without If-Match was accepted", rec.Code)
		}
	})
}

func TestUpdateProfile_StaleETagIsPreconditionFailed(t *testing.T) {
	h := newUserHarness(t)
	h.svc.EXPECT().
		UpdateProfile(gomock.Any(), gomock.Any(), gomock.Eq(userID), gomock.Any()).
		Return(nil, domainmedia.ErrETagValidationFailed).
		Times(1)

	r := request(t, http.MethodPatch, "/me", `{"fullname":"New Name"}`, true)
	r.Header.Set("If-Match", "stale")
	rec := httptest.NewRecorder()
	h.h.UpdateProfile(rec, r)

	if rec.Code != http.StatusPreconditionFailed {
		t.Errorf("status = %d, want 412", rec.Code)
	}
}

// ===========================================================================
// GetQuota
// ===========================================================================

func TestGetQuota_ReturnsTheCallersOwnQuota(t *testing.T) {
	h := newUserHarness(t)
	id := uuid.MustParse(userID)

	h.svc.EXPECT().
		GetQuota(gomock.Any(), gomock.Eq(userID)).
		Return(&domainuser.Quota{
			UserID: id, UsedBytes: 2048, TotalBytes: 8192, GifCount: 4, GitCount: 50,
		}, nil).
		Times(1)

	rec := httptest.NewRecorder()
	h.h.GetQuota(rec, request(t, http.MethodGet, "/me/quota", "", true))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["used_bytes"] != float64(2048) {
		t.Errorf("used_bytes = %v, want 2048", got["used_bytes"])
	}
	if got["total_bytes"] != float64(8192) {
		t.Errorf("total_bytes = %v, want 8192", got["total_bytes"])
	}
}

func TestGetQuota_Unauthenticated(t *testing.T) {
	h := newUserHarness(t)
	// the service must not be reached
	rec := httptest.NewRecorder()
	h.h.GetQuota(rec, request(t, http.MethodGet, "/me/quota", "", false))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestGetQuota_ServiceFailureIsAServerError(t *testing.T) {
	h := newUserHarness(t)
	h.svc.EXPECT().
		GetQuota(gomock.Any(), gomock.Eq(userID)).
		Return(nil, errors.New("db down")).
		Times(1)

	rec := httptest.NewRecorder()
	h.h.GetQuota(rec, request(t, http.MethodGet, "/me/quota", "", true))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}
