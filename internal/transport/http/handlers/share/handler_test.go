package share_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	domainshare "github.com/labib0x9/ffgif/internal/domain/share"
	"github.com/labib0x9/ffgif/internal/transport/http/handlers/share"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
)

type mockShareService struct {
	createFunc func(ctx context.Context, sharedBy string, gifKey string, sharedWith string, expiresAt time.Time) error
	deleteFunc func(ctx context.Context, userId, gifKey, shareWithId string) error
	getFunc    func(ctx context.Context, user string) ([]domainshare.GifResponse, error)
}

func (m *mockShareService) Create(ctx context.Context, sharedBy string, gifKey string, sharedWith string, expiresAt time.Time) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, sharedBy, gifKey, sharedWith, expiresAt)
	}
	return nil
}
func (m *mockShareService) Delete(ctx context.Context, userId, gifKey, shareWithId string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, userId, gifKey, shareWithId)
	}
	return nil
}
func (m *mockShareService) Get(ctx context.Context, user string) ([]domainshare.GifResponse, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, user)
	}
	return []domainshare.GifResponse{}, nil
}

func withUserAuth(r *http.Request, userId string) *http.Request {
	claims := jwtpkg.Payload{
		Fullname: "Test User",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: userId,
		},
	}
	ctx := httputil.WithAuthContext(r.Context(), claims, "test-token")
	return r.WithContext(ctx)
}

func TestShareHandler_Create_Success(t *testing.T) {
	mockSvc := &mockShareService{
		createFunc: func(ctx context.Context, sharedBy string, gifKey string, sharedWith string, expiresAt time.Time) error {
			return nil
		},
	}
	handler := share.NewHandler(mockSvc, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("POST /gifs/me/{key}/shares", handler.Create)

	body, _ := json.Marshal(map[string]any{
		"shared_with": "friend@example.com",
		"expire_at":   time.Now().Add(24 * time.Hour),
	})

	req := httptest.NewRequest(http.MethodPost, "/gifs/me/my-gif.gif/shares", bytes.NewReader(body))
	req = withUserAuth(req, "owner-1")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201 Created, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestShareHandler_Create_Unauthenticated(t *testing.T) {
	handler := share.NewHandler(&mockShareService{}, nil, validator.New())

	req := httptest.NewRequest(http.MethodPost, "/gifs/me/key-1/shares", bytes.NewReader([]byte("{}")))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized, got %d", rec.Code)
	}
}

func TestShareHandler_Create_MissingKey(t *testing.T) {
	handler := share.NewHandler(&mockShareService{}, nil, validator.New())

	req := httptest.NewRequest(http.MethodPost, "/gifs/me//shares", bytes.NewReader([]byte("{}")))
	req = withUserAuth(req, "owner-1")
	rec := httptest.NewRecorder()

	handler.Create(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 Bad Request on missing key, got %d", rec.Code)
	}
}

func TestShareHandler_Create_BadJSON(t *testing.T) {
	handler := share.NewHandler(&mockShareService{}, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("POST /gifs/me/{key}/shares", handler.Create)

	req := httptest.NewRequest(http.MethodPost, "/gifs/me/gif-1/shares", bytes.NewReader([]byte("{bad")))
	req = withUserAuth(req, "owner-1")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 Bad Request on malformed JSON, got %d", rec.Code)
	}
}

func TestShareHandler_Create_ServiceError(t *testing.T) {
	mockSvc := &mockShareService{
		createFunc: func(ctx context.Context, sharedBy string, gifKey string, sharedWith string, expiresAt time.Time) error {
			return errors.New("db error")
		},
	}
	handler := share.NewHandler(mockSvc, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("POST /gifs/me/{key}/shares", handler.Create)

	body, _ := json.Marshal(map[string]any{
		"shared_with": "friend@example.com",
		"expire_at":   time.Now().Add(24 * time.Hour),
	})
	req := httptest.NewRequest(http.MethodPost, "/gifs/me/gif-1/shares", bytes.NewReader(body))
	req = withUserAuth(req, "owner-1")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 Internal Server Error, got %d", rec.Code)
	}
}

func TestShareHandler_Get_Success(t *testing.T) {
	mockSvc := &mockShareService{
		getFunc: func(ctx context.Context, user string) ([]domainshare.GifResponse, error) {
			return []domainshare.GifResponse{
				{Name: "Gif 1", GifKey: "k1", SharedWith: "user-2"},
			}, nil
		},
	}
	handler := share.NewHandler(mockSvc, nil, validator.New())

	req := httptest.NewRequest(http.MethodGet, "/gifs/me/shares", nil)
	req = withUserAuth(req, "user-uuid-1")
	rec := httptest.NewRecorder()

	handler.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestShareHandler_Get_Unauthenticated(t *testing.T) {
	handler := share.NewHandler(&mockShareService{}, nil, validator.New())

	req := httptest.NewRequest(http.MethodGet, "/gifs/me/shares", nil)
	rec := httptest.NewRecorder()

	handler.Get(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized, got %d", rec.Code)
	}
}

func TestShareHandler_Get_ServiceError(t *testing.T) {
	mockSvc := &mockShareService{
		getFunc: func(ctx context.Context, user string) ([]domainshare.GifResponse, error) {
			return nil, errors.New("db query error")
		},
	}
	handler := share.NewHandler(mockSvc, nil, validator.New())

	req := httptest.NewRequest(http.MethodGet, "/gifs/me/shares", nil)
	req = withUserAuth(req, "user-uuid-1")
	rec := httptest.NewRecorder()

	handler.Get(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 Internal Server Error, got %d", rec.Code)
	}
}

func TestShareHandler_Delete_Success(t *testing.T) {
	mockSvc := &mockShareService{
		deleteFunc: func(ctx context.Context, userId, gifKey, shareWithId string) error {
			return nil
		},
	}
	handler := share.NewHandler(mockSvc, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /gifs/me/{key}/shares/{shareWithId}", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/gifs/me/gif-123/shares/user-456", nil)
	req = withUserAuth(req, "owner-1")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestShareHandler_Delete_Unauthenticated(t *testing.T) {
	handler := share.NewHandler(&mockShareService{}, nil, validator.New())

	req := httptest.NewRequest(http.MethodDelete, "/gifs/me/gif-123/shares/user-456", nil)
	rec := httptest.NewRecorder()

	handler.Delete(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized, got %d", rec.Code)
	}
}

func TestShareHandler_Delete_MissingKey(t *testing.T) {
	handler := share.NewHandler(&mockShareService{}, nil, validator.New())

	req := httptest.NewRequest(http.MethodDelete, "/gifs/me//shares/user-456", nil)
	req = withUserAuth(req, "owner-1")
	rec := httptest.NewRecorder()

	handler.Delete(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 Bad Request on missing key, got %d", rec.Code)
	}
}

func TestShareHandler_Delete_NotAuthorized(t *testing.T) {
	mockSvc := &mockShareService{
		deleteFunc: func(ctx context.Context, userId, gifKey, shareWithId string) error {
			return domainshare.ErrNotAuthorized
		},
	}
	handler := share.NewHandler(mockSvc, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /gifs/me/{key}/shares/{shareWithId}", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/gifs/me/gif-123/shares/user-456", nil)
	req = withUserAuth(req, "attacker")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized for unauthorized delete, got %d", rec.Code)
	}
}

func TestShareHandler_Delete_NotFound(t *testing.T) {
	mockSvc := &mockShareService{
		deleteFunc: func(ctx context.Context, userId, gifKey, shareWithId string) error {
			return domainshare.ErrNotFound
		},
	}
	handler := share.NewHandler(mockSvc, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /gifs/me/{key}/shares/{shareWithId}", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/gifs/me/missing-gif/shares/user-456", nil)
	req = withUserAuth(req, "owner-1")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404 Not Found, got %d", rec.Code)
	}
}

func TestShareHandler_Delete_ServiceError(t *testing.T) {
	mockSvc := &mockShareService{
		deleteFunc: func(ctx context.Context, userId, gifKey, shareWithId string) error {
			return errors.New("db error")
		},
	}
	handler := share.NewHandler(mockSvc, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /gifs/me/{key}/shares/{shareWithId}", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/gifs/me/gif-123/shares/user-456", nil)
	req = withUserAuth(req, "owner-1")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 Internal Server Error, got %d", rec.Code)
	}
}
