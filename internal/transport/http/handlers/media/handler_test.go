package media_test

import (
	"bytes"
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
	"go.uber.org/mock/gomock"

	"github.com/labib0x9/ffgif/config"
	appmedia "github.com/labib0x9/ffgif/internal/app/media"
	mediasvcmocks "github.com/labib0x9/ffgif/internal/app/media/mocks"
	domainmedia "github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/transport/http/handlers/media"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
)

const testUserID = "11111111-1111-1111-1111-111111111111"

type handlerHarness struct {
	svc *mediasvcmocks.MockService
	h   *media.Handler
}

func newHandlerHarness(t *testing.T) *handlerHarness {
	t.Helper()
	ctrl := gomock.NewController(t)
	svc := mediasvcmocks.NewMockService(ctrl)
	mws := middleware.NewMiddlewares(&config.Config{}, nil, jwtpkg.Jwt{})
	return &handlerHarness{svc: svc, h: media.NewHandler(svc, mws, validator.New())}
}

// authed returns a request carrying a populated auth context, as the Auth
// middleware would leave it.
func authed(t *testing.T, method, target string, body any) *http.Request {
	t.Helper()
	var r *http.Request
	if body == nil {
		r = httptest.NewRequest(method, target, nil)
	} else {
		var buf bytes.Buffer
		switch b := body.(type) {
		case string:
			buf.WriteString(b)
		default:
			if err := json.NewEncoder(&buf).Encode(b); err != nil {
				t.Fatalf("encode body: %v", err)
			}
		}
		r = httptest.NewRequest(method, target, &buf)
		r.Header.Set("Content-Type", "application/json")
	}

	claims := jwtpkg.Payload{
		Fullname: "Test User",
		Email:    "test@example.com",
		Role:     "user",
		RegisteredClaims: gojwt.RegisteredClaims{
			Subject:   testUserID,
			ExpiresAt: gojwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	return r.WithContext(httputil.WithAuthContext(r.Context(), claims, "test.jwt.token"))
}

func anonymous(method, target string, body string) *http.Request {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

// ===========================================================================
// Update — does the request body actually reach the service?
// ===========================================================================

// The brief lists this as broken ("handler never decodes the request body").
// It is not: internal/transport/http/handlers/media/update.go:32 decodes into a
// media.GifUpdateRequest. This test asserts the decoded fields with a gomock
// argument matcher so a regression to a zero-value struct is caught.
func TestUpdate_DecodedBodyReachesTheService(t *testing.T) {
	h := newHandlerHarness(t)
	updatedAt := time.Now().UTC()

	h.svc.EXPECT().
		Update(gomock.Any(), gomock.Eq(testUserID), gomock.Eq("gif-1"), gomock.Any(), gomock.Eq("etag-value")).
		DoAndReturn(func(_ context.Context, _, _ string, req domainmedia.GifUpdateRequest, _ string) (*domainmedia.GifResponse, error) {
			if req.Name == nil {
				t.Fatal("the request body was discarded: Name is nil at the service boundary")
			}
			if *req.Name != "new.gif" {
				t.Errorf("Name = %q, want new.gif", *req.Name)
			}
			if req.Status == nil || *req.Status != "ready" {
				t.Errorf("Status = %v, want ready", req.Status)
			}
			if req.Persist == nil || *req.Persist != true {
				t.Errorf("Persist = %v, want true", req.Persist)
			}
			return &domainmedia.GifResponse{
				Key: "gif-1", Name: *req.Name, Status: *req.Status,
				Persist: *req.Persist, UpdatedAt: updatedAt,
			}, nil
		}).
		Times(1)

	req := authed(t, http.MethodPatch, "/gifs/gif-1", map[string]any{
		"name": "new.gif", "status": "ready", "persist": true,
	})
	req.SetPathValue("key", "gif-1")
	req.Header.Set("If-Match", "etag-value")

	rec := httptest.NewRecorder()
	h.h.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}

	var got domainmedia.GifResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v — body = %s", err, rec.Body.String())
	}
	if got.Name != "new.gif" {
		t.Errorf("response Name = %q, want new.gif", got.Name)
	}
}

// A partial PATCH must leave the untouched fields nil so the repository's
// COALESCE keeps their current values.
func TestUpdate_PartialBodyLeavesOtherFieldsNil(t *testing.T) {
	h := newHandlerHarness(t)

	h.svc.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _ string, req domainmedia.GifUpdateRequest, _ string) (*domainmedia.GifResponse, error) {
			if req.Name == nil || *req.Name != "only-name.gif" {
				t.Errorf("Name = %v, want only-name.gif", req.Name)
			}
			if req.Status != nil {
				t.Errorf("Status = %q, want nil for a field the client did not send", *req.Status)
			}
			if req.Persist != nil {
				t.Errorf("Persist = %v, want nil for a field the client did not send", *req.Persist)
			}
			return &domainmedia.GifResponse{Key: "gif-1"}, nil
		}).
		Times(1)

	req := authed(t, http.MethodPatch, "/gifs/gif-1", `{"name":"only-name.gif"}`)
	req.SetPathValue("key", "gif-1")
	req.Header.Set("If-Match", "etag")

	rec := httptest.NewRecorder()
	h.h.Update(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}

func TestUpdate_RequestPreconditions(t *testing.T) {
	tests := []struct {
		name     string
		authed   bool
		key      string
		ifMatch  string
		body     string
		wantCode int
	}{
		{name: "unauthenticated", key: "gif-1", ifMatch: "etag", body: `{"name":"x"}`, wantCode: http.StatusUnauthorized},
		{name: "missing path key", authed: true, ifMatch: "etag", body: `{"name":"x"}`, wantCode: http.StatusBadRequest},
		{name: "malformed json", authed: true, key: "gif-1", ifMatch: "etag", body: `{"name":`, wantCode: http.StatusBadRequest},
		{name: "missing If-Match", authed: true, key: "gif-1", body: `{"name":"x"}`, wantCode: http.StatusPreconditionFailed},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHandlerHarness(t)
			// the service must never be reached for any of these
			var req *http.Request
			if tc.authed {
				req = authed(t, http.MethodPatch, "/gifs/"+tc.key, tc.body)
			} else {
				req = anonymous(http.MethodPatch, "/gifs/"+tc.key, tc.body)
			}
			if tc.key != "" {
				req.SetPathValue("key", tc.key)
			}
			if tc.ifMatch != "" {
				req.Header.Set("If-Match", tc.ifMatch)
			}

			rec := httptest.NewRecorder()
			h.h.Update(rec, req)

			if rec.Code != tc.wantCode {
				t.Errorf("status = %d, want %d; body = %s", rec.Code, tc.wantCode, rec.Body.String())
			}
		})
	}
}

func TestUpdate_ServiceErrorsMapToStatusCodes(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{"gif not found", domainmedia.ErrGifNotFound, http.StatusNotFound},
		{"not the owner", domainmedia.ErrGifOwnerMismatch, http.StatusForbidden},
		{"stale etag", domainmedia.ErrETagValidationFailed, http.StatusPreconditionFailed},
		{"unexpected failure", errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHandlerHarness(t)
			h.svc.EXPECT().
				Update(gomock.Any(), gomock.Eq(testUserID), gomock.Eq("gif-1"), gomock.Any(), gomock.Any()).
				Return(nil, tc.err).
				Times(1)

			req := authed(t, http.MethodPatch, "/gifs/gif-1", `{"name":"x"}`)
			req.SetPathValue("key", "gif-1")
			req.Header.Set("If-Match", "etag")

			rec := httptest.NewRecorder()
			h.h.Update(rec, req)

			if rec.Code != tc.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantCode)
			}
		})
	}
}

// ===========================================================================
// Convert — request validation
// ===========================================================================

type convertBody struct {
	Key   string  `json:"upload_key"`
	Start float32 `json:"start_time"`
	End   float32 `json:"end_time"`
	Width int     `json:"width"`
	FPS   int     `json:"fps"`
	Loop  bool    `json:"loop"`
}

func validConvert() convertBody {
	return convertBody{Key: "upload-1", Start: 0, End: 5, Width: 480, FPS: 15}
}

// EXPECTED TO FAIL: convertRequ in
// internal/transport/http/handlers/media/convert.go validates Start and End
// independently (`gte=0` and `gt=0`) and never relates them. A trim window
// where end <= start is accepted and forwarded to the worker, which hands
// ffmpeg a negative or zero duration.
func TestConvert_RejectsTrimWindowWhereEndIsNotAfterStart(t *testing.T) {
	tests := []struct {
		name  string
		start float32
		end   float32
	}{
		{"end before start", 5, 2},
		{"end equals start", 5, 5},
		{"zero-length window at the origin", 0, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHandlerHarness(t)

			h.svc.EXPECT().
				Convert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _, key string, start, end float32, fps, width int, loop bool) (*appmedia.ConvertResult, error) {
					t.Errorf("a conversion with start=%v end=%v was accepted and queued; "+
						"ffmpeg receives a %v-second window", start, end, end-start)
					return &appmedia.ConvertResult{Id: "job-1", Status: "queued"}, nil
				}).
				AnyTimes()

			body := validConvert()
			body.Start, body.End = tc.start, tc.end

			req := authed(t, http.MethodPost, "/convert", body)
			rec := httptest.NewRecorder()
			h.h.Convert(rec, req)

			if rec.Code != http.StatusUnprocessableEntity {
				t.Errorf("status = %d, want 422 for start=%v end=%v",
					rec.Code, tc.start, tc.end)
			}
		})
	}
}

// The declared width/fps bounds must hold exactly at the edges.
func TestConvert_WidthAndFpsBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		fps      int
		wantCode int
	}{
		{"width at lower bound", 100, 15, http.StatusAccepted},
		{"width at upper bound", 1920, 15, http.StatusAccepted},
		{"width one below lower bound", 99, 15, http.StatusUnprocessableEntity},
		{"width one above upper bound", 1921, 15, http.StatusUnprocessableEntity},
		{"width zero", 0, 15, http.StatusUnprocessableEntity},
		{"fps at lower bound", 480, 1, http.StatusAccepted},
		{"fps at upper bound", 480, 30, http.StatusAccepted},
		{"fps zero", 480, 0, http.StatusUnprocessableEntity},
		{"fps above upper bound", 480, 31, http.StatusUnprocessableEntity},
		{"fps negative", 480, -1, http.StatusUnprocessableEntity},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHandlerHarness(t)

			if tc.wantCode == http.StatusAccepted {
				h.svc.EXPECT().
					Convert(gomock.Any(), gomock.Eq(testUserID), gomock.Eq("upload-1"),
						gomock.Any(), gomock.Any(), gomock.Eq(tc.fps), gomock.Eq(tc.width), gomock.Eq(false)).
					Return(&appmedia.ConvertResult{Id: "job-1", Status: "queued"}, nil).
					Times(1)
			}
			// otherwise the service must not be called at all

			body := validConvert()
			body.Width, body.FPS = tc.width, tc.fps

			req := authed(t, http.MethodPost, "/convert", body)
			rec := httptest.NewRecorder()
			h.h.Convert(rec, req)

			if rec.Code != tc.wantCode {
				t.Errorf("status = %d, want %d; body = %s", rec.Code, tc.wantCode, rec.Body.String())
			}
		})
	}
}

func TestConvert_MissingOrMalformedRequest(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{"missing upload key", `{"start_time":0,"end_time":5,"width":480,"fps":15}`, http.StatusUnprocessableEntity},
		{"empty upload key", `{"upload_key":"","start_time":0,"end_time":5,"width":480,"fps":15}`, http.StatusUnprocessableEntity},
		{"negative start", `{"upload_key":"u","start_time":-1,"end_time":5,"width":480,"fps":15}`, http.StatusUnprocessableEntity},
		{"malformed json", `{"upload_key":`, http.StatusBadRequest},
		{"empty body", ``, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHandlerHarness(t)
			// the service must not be reached
			req := authed(t, http.MethodPost, "/convert", tc.body)
			rec := httptest.NewRecorder()
			h.h.Convert(rec, req)

			if rec.Code != tc.wantCode {
				t.Errorf("status = %d, want %d; body = %s", rec.Code, tc.wantCode, rec.Body.String())
			}
		})
	}
}

func TestConvert_AcceptedResponseCarriesJobIdAndLocation(t *testing.T) {
	h := newHandlerHarness(t)

	h.svc.EXPECT().
		Convert(gomock.Any(), gomock.Eq(testUserID), gomock.Eq("upload-1"),
			gomock.Eq(float32(1.5)), gomock.Eq(float32(9.25)),
			gomock.Eq(24), gomock.Eq(640), gomock.Eq(true)).
		Return(&appmedia.ConvertResult{Id: "job-abc", Status: "queued"}, nil).
		Times(1)

	body := convertBody{Key: "upload-1", Start: 1.5, End: 9.25, Width: 640, FPS: 24, Loop: true}
	req := authed(t, http.MethodPost, "/convert", body)
	rec := httptest.NewRecorder()
	h.h.Convert(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "/jobs/job-abc/status" {
		t.Errorf("Location = %q, want /jobs/job-abc/status", got)
	}

	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["job_id"] != "job-abc" || payload["status"] != "queued" {
		t.Errorf("body = %v, want job_id=job-abc status=queued", payload)
	}
}

// EXPECTED TO FAIL: Convert's handler validates the body BEFORE it reads the
// user id from the context, and answers 500 when the id is missing. An
// unauthenticated request must be 401, not 500 — and it must not be told
// whether its payload was well-formed.
func TestConvert_UnauthenticatedRequestIsUnauthorizedNotServerError(t *testing.T) {
	h := newHandlerHarness(t)

	buf, err := json.Marshal(validConvert())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := anonymous(http.MethodPost, "/convert", string(buf))
	rec := httptest.NewRecorder()
	h.h.Convert(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for a request with no auth context", rec.Code)
	}
}

// ===========================================================================
// Status
// ===========================================================================

func TestStatus_LocationHeaderOnlyWhenReady(t *testing.T) {
	tests := []struct {
		name         string
		status       string
		streamKey    string
		wantLocation string
	}{
		{"ready", "ok", "user-1:clip.mp4", "/uploads/user-1:clip.mp4/stream"},
		{"still uploading", "uploading", "", ""},
		{"failed", "failed", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHandlerHarness(t)
			h.svc.EXPECT().
				Status(gomock.Any(), gomock.Eq(testUserID), gomock.Eq("key-1")).
				Return(tc.streamKey, tc.status, nil).
				Times(1)

			req := authed(t, http.MethodGet, "/uploads/key-1/status", nil)
			req.SetPathValue("key", "key-1")
			rec := httptest.NewRecorder()
			h.h.Status(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			if got := rec.Header().Get("Location"); got != tc.wantLocation {
				t.Errorf("Location = %q, want %q", got, tc.wantLocation)
			}
		})
	}
}

func TestStatus_RequiresKeyAndAuth(t *testing.T) {
	t.Run("missing key", func(t *testing.T) {
		h := newHandlerHarness(t)
		req := authed(t, http.MethodGet, "/uploads//status", nil)
		rec := httptest.NewRecorder()
		h.h.Status(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		h := newHandlerHarness(t)
		req := anonymous(http.MethodGet, "/uploads/key-1/status", "")
		req.SetPathValue("key", "key-1")
		rec := httptest.NewRecorder()
		h.h.Status(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rec.Code)
		}
	})
}
