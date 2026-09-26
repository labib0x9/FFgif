package share_test

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	gojwt "github.com/golang-jwt/jwt/v5"
	"go.uber.org/mock/gomock"

	"github.com/labib0x9/ffgif/config"
	sharesvcmocks "github.com/labib0x9/ffgif/internal/app/share/mocks"
	domainmedia "github.com/labib0x9/ffgif/internal/domain/media"
	domainshare "github.com/labib0x9/ffgif/internal/domain/share"
	"github.com/labib0x9/ffgif/internal/transport/http/handlers/share"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
)

const callerID = "22222222-2222-2222-2222-222222222222"

type shareHarness struct {
	svc *sharesvcmocks.MockService
	h   *share.Handler
}

func newShareHarness(t *testing.T) *shareHarness {
	t.Helper()
	ctrl := gomock.NewController(t)
	svc := sharesvcmocks.NewMockService(ctrl)
	mws := middleware.NewMiddlewares(&config.Config{}, nil, jwtpkg.Jwt{})
	return &shareHarness{svc: svc, h: share.NewHandler(svc, mws, validator.New())}
}

func req(t *testing.T, method, target, body string, withAuth bool) *http.Request {
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
			Fullname: "Caller", Email: "caller@example.com", Role: "user",
			RegisteredClaims: gojwt.RegisteredClaims{
				Subject:   callerID,
				ExpiresAt: gojwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		}
		r = r.WithContext(httputil.WithAuthContext(r.Context(), claims, "tok"))
	}
	return r
}

// ===========================================================================
// Create
// ===========================================================================

// The gif key comes from the path and the sharer id from the token — never
// from the body, which the caller controls.
func TestCreate_UsesThePathKeyAndTheTokenSubject(t *testing.T) {
	h := newShareHarness(t)
	expiry := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)

	h.svc.EXPECT().
		Create(gomock.Any(), gomock.Eq(callerID), gomock.Eq("gif-1"),
			gomock.Eq("friend@example.com"), gomock.Any()).
		DoAndReturn(func(_ context.Context, sharedBy, key, with string, exp time.Time) error {
			if !exp.Equal(expiry) {
				t.Errorf("expiry = %v, want %v", exp, expiry)
			}
			return nil
		}).
		Times(1)

	r := req(t, http.MethodPost, "/gifs/gif-1/shares",
		`{"shared_with":"friend@example.com","expire_at":"`+expiry.Format(time.RFC3339)+`",
		  "owner_id":"attacker-supplied","gif_key":"attacker-supplied"}`, true)
	r.SetPathValue("key", "gif-1")

	rec := httptest.NewRecorder()
	h.h.Create(rec, r)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}
}

func TestCreate_Preconditions(t *testing.T) {
	tests := []struct {
		name     string
		withAuth bool
		key      string
		body     string
		wantCode int
	}{
		{name: "unauthenticated", key: "gif-1", body: `{"shared_with":"f@example.com"}`, wantCode: http.StatusUnauthorized},
		{name: "missing gif key", withAuth: true, body: `{"shared_with":"f@example.com"}`, wantCode: http.StatusBadRequest},
		{name: "malformed json", withAuth: true, key: "gif-1", body: `{"shared_with":`, wantCode: http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newShareHarness(t)
			// the service must not be reached
			r := req(t, http.MethodPost, "/gifs/"+tc.key+"/shares", tc.body, tc.withAuth)
			if tc.key != "" {
				r.SetPathValue("key", tc.key)
			}
			rec := httptest.NewRecorder()
			h.h.Create(rec, r)

			if rec.Code != tc.wantCode {
				t.Errorf("status = %d, want %d; body = %s", rec.Code, tc.wantCode, rec.Body.String())
			}
		})
	}
}

// EXPECTED TO FAIL: reqCreate in
// internal/transport/http/handlers/share/create.go carries no validate tags and
// the handler never calls h.validate.Struct on it — unlike every other write
// handler in the codebase. An empty or malformed recipient, and a zero-value
// expire_at, go straight to the service.
func TestCreate_ValidatesTheRecipientAndExpiry(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"empty recipient", `{"shared_with":"","expire_at":"2030-01-01T00:00:00Z"}`},
		{"recipient is not an email", `{"shared_with":"not-an-email","expire_at":"2030-01-01T00:00:00Z"}`},
		{"missing recipient entirely", `{"expire_at":"2030-01-01T00:00:00Z"}`},
		{"zero-value expiry", `{"shared_with":"f@example.com"}`},
		{"expiry in the past", `{"shared_with":"f@example.com","expire_at":"2000-01-01T00:00:00Z"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newShareHarness(t)

			h.svc.EXPECT().
				Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _, _, with string, exp time.Time) error {
					t.Errorf("the handler forwarded shared_with=%q expire_at=%v without validating it", with, exp)
					return nil
				}).
				AnyTimes()

			r := req(t, http.MethodPost, "/gifs/gif-1/shares", tc.body, true)
			r.SetPathValue("key", "gif-1")
			rec := httptest.NewRecorder()
			h.h.Create(rec, r)

			if rec.Code != http.StatusUnprocessableEntity && rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 422 or 400; body = %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCreate_ErrorMapping(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{"gif not found", domainmedia.ErrGifNotFound, http.StatusNotFound},
		{"caller does not own the gif", domainmedia.ErrGifOwnerMismatch, http.StatusForbidden},
		{"unexpected failure", errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newShareHarness(t)
			h.svc.EXPECT().
				Create(gomock.Any(), gomock.Eq(callerID), gomock.Eq("gif-1"), gomock.Any(), gomock.Any()).
				Return(tc.err).
				Times(1)

			r := req(t, http.MethodPost, "/gifs/gif-1/shares",
				`{"shared_with":"f@example.com","expire_at":"2030-01-01T00:00:00Z"}`, true)
			r.SetPathValue("key", "gif-1")
			rec := httptest.NewRecorder()
			h.h.Create(rec, r)

			if rec.Code != tc.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantCode)
			}
		})
	}
}

// ===========================================================================
// Delete
// ===========================================================================

func TestDelete_ForwardsBothPathSegmentsAndTheCallerId(t *testing.T) {
	h := newShareHarness(t)

	h.svc.EXPECT().
		Delete(gomock.Any(), gomock.Eq(callerID), gomock.Eq("gif-1"), gomock.Eq("recipient-9")).
		Return(nil).
		Times(1)

	r := req(t, http.MethodDelete, "/gifs/gif-1/shares/recipient-9", "", true)
	r.SetPathValue("key", "gif-1")
	r.SetPathValue("shareWithId", "recipient-9")

	rec := httptest.NewRecorder()
	h.h.Delete(rec, r)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}

func TestDelete_ErrorMapping(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{"share belongs to someone else", domainshare.ErrNotAuthorized, http.StatusForbidden},
		{"already deleted", domainshare.ErrNotFound, http.StatusNotFound},
		{"unexpected failure", errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newShareHarness(t)
			h.svc.EXPECT().
				Delete(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(tc.err).
				Times(1)

			r := req(t, http.MethodDelete, "/gifs/gif-1/shares/recipient-9", "", true)
			r.SetPathValue("key", "gif-1")
			r.SetPathValue("shareWithId", "recipient-9")
			rec := httptest.NewRecorder()
			h.h.Delete(rec, r)

			if rec.Code != tc.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantCode)
			}
		})
	}
}

func TestDelete_Preconditions(t *testing.T) {
	tests := []struct {
		name        string
		withAuth    bool
		key         string
		shareWithID string
		wantCode    int
	}{
		{name: "unauthenticated", key: "gif-1", shareWithID: "r-9", wantCode: http.StatusUnauthorized},
		{name: "missing gif key", withAuth: true, shareWithID: "r-9", wantCode: http.StatusBadRequest},
		{name: "missing recipient id", withAuth: true, key: "gif-1", wantCode: http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newShareHarness(t)
			// the service must not be reached
			r := req(t, http.MethodDelete, "/gifs/x/shares/y", "", tc.withAuth)
			if tc.key != "" {
				r.SetPathValue("key", tc.key)
			}
			if tc.shareWithID != "" {
				r.SetPathValue("shareWithId", tc.shareWithID)
			}
			rec := httptest.NewRecorder()
			h.h.Delete(rec, r)

			if rec.Code != tc.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantCode)
			}
		})
	}
}

// ===========================================================================
// Get / GetByToken
// ===========================================================================

func TestGet_ScopesToTheAuthenticatedCaller(t *testing.T) {
	h := newShareHarness(t)

	h.svc.EXPECT().
		Get(gomock.Any(), gomock.Eq(callerID)).
		Return([]domainshare.GifResponse{{GifKey: "k1", Name: "one.gif"}}, nil).
		Times(1)

	r := req(t, http.MethodGet, "/shares", "", true)
	rec := httptest.NewRecorder()
	h.h.Get(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}

func TestGet_UnauthenticatedIsRefused(t *testing.T) {
	h := newShareHarness(t)
	// the service must not be reached
	rec := httptest.NewRecorder()
	h.h.Get(rec, req(t, http.MethodGet, "/shares", "", false))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

// The token link is public by design, so it must not require auth — but an
// unknown or expired token must be a plain 404 that reveals nothing.
func TestGetByToken(t *testing.T) {
	t.Run("valid token", func(t *testing.T) {
		h := newShareHarness(t)
		h.svc.EXPECT().
			GetByToken(gomock.Any(), gomock.Eq("share-token")).
			Return(domainshare.GifTokenResponse{GifKey: "gif-1", Name: "one.gif", Url: "https://storage/one.gif"}, nil).
			Times(1)

		r := req(t, http.MethodGet, "/s/share-token", "", false)
		r.SetPathValue("token", "share-token")
		rec := httptest.NewRecorder()
		h.h.GetByToken(rec, r)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("expired or unknown token", func(t *testing.T) {
		h := newShareHarness(t)
		h.svc.EXPECT().
			GetByToken(gomock.Any(), gomock.Eq("expired")).
			Return(domainshare.GifTokenResponse{}, sql.ErrNoRows).
			Times(1)

		r := req(t, http.MethodGet, "/s/expired", "", false)
		r.SetPathValue("token", "expired")
		rec := httptest.NewRecorder()
		h.h.GetByToken(rec, r)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
		if strings.Contains(rec.Body.String(), "storage") {
			t.Errorf("a storage URL leaked in the 404 body: %s", rec.Body.String())
		}
	})

	t.Run("missing token segment", func(t *testing.T) {
		h := newShareHarness(t)
		// the service must not be reached
		rec := httptest.NewRecorder()
		h.h.GetByToken(rec, req(t, http.MethodGet, "/s/", "", false))

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})
}
