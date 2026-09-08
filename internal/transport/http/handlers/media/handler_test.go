package media_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	appmedia "github.com/labib0x9/ffgif/internal/app/media"
	domainmedia "github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/port/queue"
	"github.com/labib0x9/ffgif/internal/transport/http/handlers/media"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
)

type mockMediaService struct {
	uploadFunc                func(rctx context.Context, filename string, claims string) (*domainmedia.UploadResult, error)
	downloadFunc              func(ctx context.Context, userId, key string) (string, error)
	getByKeyFunc              func(ctx context.Context, userId string, key string) (*domainmedia.GifResponse, error)
	getRecentsFunc            func(ctx context.Context, id string) ([]domainmedia.GifResponse, error)
	getGifsFunc               func(ctx context.Context, id string, filter string) (*appmedia.GifResult, error)
	lastVideoFunc             func(ctx context.Context, userId string) (domainmedia.LastUploadResponse, error)
	deleteFunc                func(ctx context.Context, userId string, key string) error
	saveFunc                  func(ctx context.Context, userId, key string) error
	streamFunc                func(ctx context.Context, userId string, key string) (*domainmedia.StreamResult, error)
	updateFunc                func(ctx context.Context, userId string, key string, _gif domainmedia.GifUpdateRequest, lastUpdatedAt string) (*domainmedia.GifResponse, error)
	processAndSaveFunc        func(ctx context.Context, key string) error
	updateUploadingStatusFunc func(ctx context.Context, key string, status string) error
	statusFunc                func(ctx context.Context, userId, key string) (string, string, error)

	convertFunc          func(ctx context.Context, userId string, key string, start float32, end float32, fps int, width int, loop bool) (*appmedia.ConvertResult, error)
	conversionStatusFunc func(ctx context.Context, jobId string) (*appmedia.StatusResult, error)
	processFunc          func(ctx context.Context, msg queue.VideoMessage) error
	saveMetadataFunc     func(ctx context.Context, msg queue.SaveVideoMessage) error

	getGifThumbnailFunc func(ctx context.Context, userId string, key string) (string, error)
}

func (m *mockMediaService) Upload(rctx context.Context, filename string, claims string) (*domainmedia.UploadResult, error) {
	if m.uploadFunc != nil {
		return m.uploadFunc(rctx, filename, claims)
	}
	return &domainmedia.UploadResult{Url: "https://storage/presigned-put", Key: "upload-123", ExpireIn: 300}, nil
}
func (m *mockMediaService) Download(ctx context.Context, userId, key string) (string, error) {
	if m.downloadFunc != nil {
		return m.downloadFunc(ctx, userId, key)
	}
	return "https://storage/presigned-get", nil
}
func (m *mockMediaService) GetByKey(ctx context.Context, userId string, key string) (*domainmedia.GifResponse, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, userId, key)
	}
	return &domainmedia.GifResponse{Key: key, Url: "https://storage/" + key}, nil
}
func (m *mockMediaService) GetRecents(ctx context.Context, id string) ([]domainmedia.GifResponse, error) {
	if m.getRecentsFunc != nil {
		return m.getRecentsFunc(ctx, id)
	}
	return []domainmedia.GifResponse{}, nil
}
func (m *mockMediaService) GetGifs(ctx context.Context, id string, filter string) (*appmedia.GifResult, error) {
	if m.getGifsFunc != nil {
		return m.getGifsFunc(ctx, id, filter)
	}
	return &appmedia.GifResult{Data: []domainmedia.GifResponse{}, Total: 0}, nil
}
func (m *mockMediaService) LastVideo(ctx context.Context, userId string) (domainmedia.LastUploadResponse, error) {
	if m.lastVideoFunc != nil {
		return m.lastVideoFunc(ctx, userId)
	}
	uID, _ := uuid.Parse(userId)
	return domainmedia.LastUploadResponse{UserID: uID, FileKey: "last.mp4"}, nil
}
func (m *mockMediaService) Delete(ctx context.Context, userId string, key string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, userId, key)
	}
	return nil
}
func (m *mockMediaService) Save(ctx context.Context, userId, key string) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, userId, key)
	}
	return nil
}
func (m *mockMediaService) Stream(ctx context.Context, userId string, key string) (*domainmedia.StreamResult, error) {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, userId, key)
	}
	return &domainmedia.StreamResult{PresignedUrl: "https://storage/stream/" + key, ExpireIn: 300}, nil
}
func (m *mockMediaService) Update(ctx context.Context, userId string, key string, _gif domainmedia.GifUpdateRequest, lastUpdatedAt string) (*domainmedia.GifResponse, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, userId, key, _gif, lastUpdatedAt)
	}
	return &domainmedia.GifResponse{Key: key}, nil
}
func (m *mockMediaService) ProcessAndSave(ctx context.Context, key string) error {
	if m.processAndSaveFunc != nil {
		return m.processAndSaveFunc(ctx, key)
	}
	return nil
}
func (m *mockMediaService) UpdateUploadingStatus(ctx context.Context, key string, status string) error {
	if m.updateUploadingStatusFunc != nil {
		return m.updateUploadingStatusFunc(ctx, key, status)
	}
	return nil
}
func (m *mockMediaService) Status(ctx context.Context, userId, key string) (string, string, error) {
	if m.statusFunc != nil {
		return m.statusFunc(ctx, userId, key)
	}
	return "stream-key", "ok", nil
}
func (m *mockMediaService) Convert(ctx context.Context, userId string, key string, start float32, end float32, fps int, width int, loop bool) (*appmedia.ConvertResult, error) {
	if m.convertFunc != nil {
		return m.convertFunc(ctx, userId, key, start, end, fps, width, loop)
	}
	return &appmedia.ConvertResult{Id: "job-123", Status: "queued"}, nil
}
func (m *mockMediaService) ConversionStatus(ctx context.Context, jobId string) (*appmedia.StatusResult, error) {
	if m.conversionStatusFunc != nil {
		return m.conversionStatusFunc(ctx, jobId)
	}
	return &appmedia.StatusResult{JobId: jobId, Status: "completed", GifId: "gif-123"}, nil
}
func (m *mockMediaService) Process(ctx context.Context, msg queue.VideoMessage) error {
	if m.processFunc != nil {
		return m.processFunc(ctx, msg)
	}
	return nil
}
func (m *mockMediaService) SaveMetadata(ctx context.Context, msg queue.SaveVideoMessage) error {
	if m.saveMetadataFunc != nil {
		return m.saveMetadataFunc(ctx, msg)
	}
	return nil
}
func (m *mockMediaService) GetGifThumbnail(ctx context.Context, userId string, key string) (string, error) {
	if m.getGifThumbnailFunc != nil {
		return m.getGifThumbnailFunc(ctx, userId, key)
	}
	return "https://storage/thumbnail.jpg", nil
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

func TestMediaHandler_Upload_Success(t *testing.T) {
	mockSvc := &mockMediaService{
		uploadFunc: func(rctx context.Context, filename string, claims string) (*domainmedia.UploadResult, error) {
			return &domainmedia.UploadResult{
				Url:      "https://minio.local/presigned-upload",
				Key:      "user-123:video-uuid.mp4",
				ExpireIn: 300,
			}, nil
		},
	}
	handler := media.NewHandler(mockSvc, nil, validator.New())

	body, _ := json.Marshal(map[string]string{"filename": "my-video.mp4"})
	req := httptest.NewRequest(http.MethodPost, "/uploads", bytes.NewReader(body))
	req = withUserAuth(req, "user-123")

	rec := httptest.NewRecorder()
	handler.Upload(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201 Created, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestMediaHandler_Upload_BadJSON(t *testing.T) {
	handler := media.NewHandler(&mockMediaService{}, nil, validator.New())
	req := httptest.NewRequest(http.MethodPost, "/uploads", bytes.NewReader([]byte("{bad")))
	rec := httptest.NewRecorder()

	handler.Upload(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 Bad Request, got %d", rec.Code)
	}
}

func TestMediaHandler_Status_Success(t *testing.T) {
	mockSvc := &mockMediaService{
		statusFunc: func(ctx context.Context, userId, key string) (string, string, error) {
			return "stream-test.mp4", "ok", nil
		},
	}
	handler := media.NewHandler(mockSvc, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /uploads/{key}/status", handler.Status)

	req := httptest.NewRequest(http.MethodGet, "/uploads/user-1:test.mp4/status", nil)
	req = withUserAuth(req, "user-1")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestMediaHandler_Convert_Success(t *testing.T) {
	mockSvc := &mockMediaService{
		convertFunc: func(ctx context.Context, userId string, key string, start float32, end float32, fps int, width int, loop bool) (*appmedia.ConvertResult, error) {
			return &appmedia.ConvertResult{Id: "job-xyz", Status: "queued"}, nil
		},
	}
	handler := media.NewHandler(mockSvc, nil, validator.New())

	body, _ := json.Marshal(map[string]any{
		"upload_key": "raw_video.mp4",
		"start_time": 0.0,
		"end_time":   5.0,
		"fps":        15,
		"width":      480,
		"loop":       true,
	})
	req := httptest.NewRequest(http.MethodPost, "/convert", bytes.NewReader(body))
	req = withUserAuth(req, "user-123")

	rec := httptest.NewRecorder()
	handler.Convert(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("expected status 202 Accepted, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestMediaHandler_Convert_ValidationFailed(t *testing.T) {
	handler := media.NewHandler(&mockMediaService{}, nil, validator.New())

	body, _ := json.Marshal(map[string]any{
		"upload_key": "",
		"start_time": -1.0,
	})
	req := httptest.NewRequest(http.MethodPost, "/convert", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Convert(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422 Unprocessable Entity, got %d", rec.Code)
	}
}

func TestMediaHandler_ConversionStatus_Success(t *testing.T) {
	mockSvc := &mockMediaService{
		conversionStatusFunc: func(ctx context.Context, jobId string) (*appmedia.StatusResult, error) {
			return &appmedia.StatusResult{JobId: jobId, Status: "completed", GifId: "output.gif"}, nil
		},
	}
	handler := media.NewHandler(mockSvc, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /convert/{jobId}/status", handler.ConversionStatus)

	req := httptest.NewRequest(http.MethodGet, "/convert/job-123/status", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestMediaHandler_GetGifs_Success(t *testing.T) {
	mockSvc := &mockMediaService{
		getGifsFunc: func(ctx context.Context, id string, filter string) (*appmedia.GifResult, error) {
			return &appmedia.GifResult{Data: []domainmedia.GifResponse{}, Total: 0}, nil
		},
	}
	handler := media.NewHandler(mockSvc, nil, validator.New())

	req := httptest.NewRequest(http.MethodGet, "/gifs/me", nil)
	req = withUserAuth(req, "user-123")

	rec := httptest.NewRecorder()
	handler.GetGifs(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestMediaHandler_GetByKey_Success(t *testing.T) {
	mockSvc := &mockMediaService{
		getByKeyFunc: func(ctx context.Context, userId string, key string) (*domainmedia.GifResponse, error) {
			return &domainmedia.GifResponse{Key: key}, nil
		},
	}
	handler := media.NewHandler(mockSvc, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /gifs/me/{key}", handler.GetByKey)

	req := httptest.NewRequest(http.MethodGet, "/gifs/me/sample.gif", nil)
	req = withUserAuth(req, "user-123")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestMediaHandler_GetRecents_Success(t *testing.T) {
	mockSvc := &mockMediaService{
		getRecentsFunc: func(ctx context.Context, id string) ([]domainmedia.GifResponse, error) {
			return []domainmedia.GifResponse{}, nil
		},
	}
	handler := media.NewHandler(mockSvc, nil, validator.New())

	req := httptest.NewRequest(http.MethodGet, "/gifs/me/recents", nil)
	req = withUserAuth(req, "user-123")

	rec := httptest.NewRecorder()
	handler.GetRecents(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestMediaHandler_LastVideo_Success(t *testing.T) {
	mockSvc := &mockMediaService{
		lastVideoFunc: func(ctx context.Context, userId string) (domainmedia.LastUploadResponse, error) {
			return domainmedia.LastUploadResponse{FileKey: "last.mp4"}, nil
		},
	}
	handler := media.NewHandler(mockSvc, nil, validator.New())

	req := httptest.NewRequest(http.MethodGet, "/uploads/last", nil)
	req = withUserAuth(req, "123e4567-e89b-12d3-a456-426614174000")

	rec := httptest.NewRecorder()
	handler.LastVideo(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestMediaHandler_Download_Success(t *testing.T) {
	mockSvc := &mockMediaService{
		downloadFunc: func(ctx context.Context, userId, key string) (string, error) {
			return "https://minio.download/sample.gif", nil
		},
	}
	handler := media.NewHandler(mockSvc, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /gifs/me/{key}/download", handler.Download)

	req := httptest.NewRequest(http.MethodGet, "/gifs/me/sample.gif/download", nil)
	req = withUserAuth(req, "user-123")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestMediaHandler_Download_NotFound(t *testing.T) {
	mockSvc := &mockMediaService{
		downloadFunc: func(ctx context.Context, userId, key string) (string, error) {
			return "", domainmedia.ErrGifNotFound
		},
	}
	handler := media.NewHandler(mockSvc, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /gifs/me/{key}/download", handler.Download)

	req := httptest.NewRequest(http.MethodGet, "/gifs/me/missing.gif/download", nil)
	req = withUserAuth(req, "user-123")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404 Not Found, got %d", rec.Code)
	}
}

func TestMediaHandler_Download_OwnerMismatch(t *testing.T) {
	mockSvc := &mockMediaService{
		downloadFunc: func(ctx context.Context, userId, key string) (string, error) {
			return "", domainmedia.ErrGifOwnerMismatch
		},
	}
	handler := media.NewHandler(mockSvc, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /gifs/me/{key}/download", handler.Download)

	req := httptest.NewRequest(http.MethodGet, "/gifs/me/private.gif/download", nil)
	req = withUserAuth(req, "user-123")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status 403 Forbidden, got %d", rec.Code)
	}
}

func TestMediaHandler_Stream_Success(t *testing.T) {
	mockSvc := &mockMediaService{
		streamFunc: func(ctx context.Context, userId string, key string) (*domainmedia.StreamResult, error) {
			return &domainmedia.StreamResult{PresignedUrl: "https://minio/stream/v.mp4"}, nil
		},
	}
	handler := media.NewHandler(mockSvc, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /uploads/{key}/stream", handler.Stream)

	req := httptest.NewRequest(http.MethodGet, "/uploads/user-1:v.mp4/stream", nil)
	req = withUserAuth(req, "user-1")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestMediaHandler_Save_Success(t *testing.T) {
	mockSvc := &mockMediaService{
		saveFunc: func(ctx context.Context, userId, key string) error {
			return nil
		},
	}
	handler := media.NewHandler(mockSvc, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("POST /gifs/me/recents/{key}/save", handler.Save)

	req := httptest.NewRequest(http.MethodPost, "/gifs/me/recents/key-123/save", nil)
	req = withUserAuth(req, "user-123")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204 No Content, got %d", rec.Code)
	}
}

func TestMediaHandler_Update_Success(t *testing.T) {
	mockSvc := &mockMediaService{
		updateFunc: func(ctx context.Context, userId string, key string, _gif domainmedia.GifUpdateRequest, lastUpdatedAt string) (*domainmedia.GifResponse, error) {
			return &domainmedia.GifResponse{Key: key}, nil
		},
	}
	handler := media.NewHandler(mockSvc, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /gifs/me/{key}", handler.Update)

	name := "updated_name"
	body, _ := json.Marshal(domainmedia.GifUpdateRequest{Name: &name})
	req := httptest.NewRequest(http.MethodPatch, "/gifs/me/key-123", bytes.NewReader(body))
	req.Header.Set("If-Match", "2026-09-07T12:00:00Z")
	req = withUserAuth(req, "user-123")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestMediaHandler_Delete_Success(t *testing.T) {
	mockSvc := &mockMediaService{
		deleteFunc: func(ctx context.Context, userId string, key string) error {
			return nil
		},
	}
	handler := media.NewHandler(mockSvc, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /gifs/me/{key}", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/gifs/me/key-123", nil)
	req = withUserAuth(req, "user-123")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestMediaHandler_Delete_NotFound(t *testing.T) {
	mockSvc := &mockMediaService{
		deleteFunc: func(ctx context.Context, userId string, key string) error {
			return domainmedia.ErrGifNotFound
		},
	}
	handler := media.NewHandler(mockSvc, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /gifs/me/{key}", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/gifs/me/missing.gif", nil)
	req = withUserAuth(req, "user-123")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404 Not Found, got %d", rec.Code)
	}
}

func TestMediaHandler_Delete_OwnerMismatch(t *testing.T) {
	mockSvc := &mockMediaService{
		deleteFunc: func(ctx context.Context, userId string, key string) error {
			return domainmedia.ErrGifOwnerMismatch
		},
	}
	handler := media.NewHandler(mockSvc, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /gifs/me/{key}", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/gifs/me/other.gif", nil)
	req = withUserAuth(req, "user-123")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status 403 Forbidden, got %d", rec.Code)
	}
}

func TestMediaHandler_GetGifThumbnail_Success(t *testing.T) {
	mockSvc := &mockMediaService{
		getGifThumbnailFunc: func(ctx context.Context, userId string, key string) (string, error) {
			return "https://minio/thumbnail.jpg", nil
		},
	}
	handler := media.NewHandler(mockSvc, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /gifs/me/{key}/thumbnail", handler.GetGifThumbnail)

	req := httptest.NewRequest(http.MethodGet, "/gifs/me/sample.gif/thumbnail", nil)
	req = withUserAuth(req, "user-123")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestMediaHandler_GetGifThumbnail_NotFound(t *testing.T) {
	mockSvc := &mockMediaService{
		getGifThumbnailFunc: func(ctx context.Context, userId string, key string) (string, error) {
			return "", domainmedia.ErrGifNotFound
		},
	}
	handler := media.NewHandler(mockSvc, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /gifs/me/{key}/thumbnail", handler.GetGifThumbnail)

	req := httptest.NewRequest(http.MethodGet, "/gifs/me/missing.gif/thumbnail", nil)
	req = withUserAuth(req, "user-123")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404 Not Found, got %d", rec.Code)
	}
}

func TestMediaHandler_GetGifThumbnail_InternalError(t *testing.T) {
	mockSvc := &mockMediaService{
		getGifThumbnailFunc: func(ctx context.Context, userId string, key string) (string, error) {
			return "", errors.New("storage error")
		},
	}
	handler := media.NewHandler(mockSvc, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /gifs/me/{key}/thumbnail", handler.GetGifThumbnail)

	req := httptest.NewRequest(http.MethodGet, "/gifs/me/error.gif/thumbnail", nil)
	req = withUserAuth(req, "user-123")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 Internal Server Error, got %d", rec.Code)
	}
}
