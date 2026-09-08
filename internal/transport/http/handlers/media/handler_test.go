package media_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	appmedia "github.com/labib0x9/ffgif/internal/app/media"
	mediaMocks "github.com/labib0x9/ffgif/internal/app/media/mocks"
	domainmedia "github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/transport/http/handlers/media"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
)

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
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)

	mockSvc.EXPECT().Upload(gomock.Any(), gomock.Eq("my-video.mp4"), gomock.Eq("user-123")).
		Return(&domainmedia.UploadResult{
			Url:      "https://minio.local/presigned-upload",
			Key:      "user-123:video-uuid.mp4",
			ExpireIn: 300,
		}, nil).Times(1)

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
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)

	handler := media.NewHandler(mockSvc, nil, validator.New())
	req := httptest.NewRequest(http.MethodPost, "/uploads", bytes.NewReader([]byte("{bad")))
	rec := httptest.NewRecorder()

	handler.Upload(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 Bad Request, got %d", rec.Code)
	}
}

func TestMediaHandler_Status_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)

	mockSvc.EXPECT().Status(gomock.Any(), gomock.Eq("user-1"), gomock.Eq("user-1:test.mp4")).
		Return("stream-test.mp4", "ok", nil).Times(1)

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
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)

	mockSvc.EXPECT().Convert(gomock.Any(), gomock.Eq("user-123"), gomock.Eq("raw_video.mp4"),
		float32(0.0), float32(5.0), 15, 480, true).
		Return(&appmedia.ConvertResult{Id: "job-xyz", Status: "queued"}, nil).Times(1)

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
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)

	handler := media.NewHandler(mockSvc, nil, validator.New())

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

// EXPECTED TO FAIL: Cross-field validation for convert (start >= end) is missing in convertRequ.
// When start=5 and end=2 or start=5 and end=5, the handler should return 422 Unprocessable Entity.
func TestMediaHandler_Convert_CrossFieldValidation_Adversarial(t *testing.T) {
	testCases := []struct {
		name  string
		start float32
		end   float32
	}{
		{"start > end", 5.0, 2.0},
		{"start == end", 5.0, 5.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockSvc := mediaMocks.NewMockService(ctrl)

			// Handler should NOT call service if validation passes
			mockSvc.EXPECT().Convert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&appmedia.ConvertResult{Id: "job-bad", Status: "queued"}, nil).AnyTimes()

			handler := media.NewHandler(mockSvc, nil, validator.New())

			body, _ := json.Marshal(map[string]any{
				"upload_key": "raw_video.mp4",
				"start_time": tc.start,
				"end_time":   tc.end,
				"fps":        15,
				"width":      480,
				"loop":       true,
			})
			req := httptest.NewRequest(http.MethodPost, "/convert", bytes.NewReader(body))
			req = withUserAuth(req, "user-123")

			rec := httptest.NewRecorder()
			handler.Convert(rec, req)

			if rec.Code != http.StatusUnprocessableEntity && rec.Code != http.StatusBadRequest {
				t.Errorf("BUG DETECTED: Convert accepted invalid time range (start=%f, end=%f), returned status %d",
					tc.start, tc.end, rec.Code)
			}
		})
	}
}

func TestMediaHandler_ConversionStatus_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)

	mockSvc.EXPECT().ConversionStatus(gomock.Any(), gomock.Eq("job-123")).
		Return(&appmedia.StatusResult{JobId: "job-123", Status: "completed", GifId: "output.gif"}, nil).Times(1)

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
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)

	mockSvc.EXPECT().GetGifs(gomock.Any(), gomock.Eq("user-123"), gomock.Any()).
		Return(&appmedia.GifResult{Data: []domainmedia.GifResponse{}, Total: 0}, nil).Times(1)

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
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)

	mockSvc.EXPECT().GetByKey(gomock.Any(), gomock.Eq("user-123"), gomock.Eq("sample.gif")).
		Return(&domainmedia.GifResponse{Key: "sample.gif"}, nil).Times(1)

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
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)

	mockSvc.EXPECT().GetRecents(gomock.Any(), gomock.Eq("user-123")).
		Return([]domainmedia.GifResponse{}, nil).Times(1)

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
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)

	uID := uuid.New()
	mockSvc.EXPECT().LastVideo(gomock.Any(), gomock.Eq(uID.String())).
		Return(domainmedia.LastUploadResponse{UserID: uID, FileKey: "last.mp4"}, nil).Times(1)

	handler := media.NewHandler(mockSvc, nil, validator.New())

	req := httptest.NewRequest(http.MethodGet, "/uploads/last", nil)
	req = withUserAuth(req, uID.String())

	rec := httptest.NewRecorder()
	handler.LastVideo(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestMediaHandler_Download_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)

	mockSvc.EXPECT().Download(gomock.Any(), gomock.Eq("user-123"), gomock.Eq("sample.gif")).
		Return("https://minio.download/sample.gif", nil).Times(1)

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
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)

	mockSvc.EXPECT().Download(gomock.Any(), gomock.Eq("user-123"), gomock.Eq("missing.gif")).
		Return("", domainmedia.ErrGifNotFound).Times(1)

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
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)

	mockSvc.EXPECT().Download(gomock.Any(), gomock.Eq("user-123"), gomock.Eq("private.gif")).
		Return("", domainmedia.ErrGifOwnerMismatch).Times(1)

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
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)

	mockSvc.EXPECT().Stream(gomock.Any(), gomock.Eq("user-1"), gomock.Eq("user-1:v.mp4")).
		Return(&domainmedia.StreamResult{PresignedUrl: "https://minio/stream/v.mp4"}, nil).Times(1)

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
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)

	mockSvc.EXPECT().Save(gomock.Any(), gomock.Eq("user-123"), gomock.Eq("key-123")).
		Return(nil).Times(1)

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
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)

	name := "updated_name"
	ifMatch := "2026-09-07T12:00:00Z"

	mockSvc.EXPECT().Update(gomock.Any(), gomock.Eq("user-123"), gomock.Eq("key-123"),
		gomock.Cond(func(x any) bool {
			req, ok := x.(domainmedia.GifUpdateRequest)
			return ok && req.Name != nil && *req.Name == "updated_name"
		}), gomock.Eq(ifMatch)).
		Return(&domainmedia.GifResponse{Key: "key-123"}, nil).Times(1)

	handler := media.NewHandler(mockSvc, nil, validator.New())

	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /gifs/me/{key}", handler.Update)

	body, _ := json.Marshal(domainmedia.GifUpdateRequest{Name: &name})
	req := httptest.NewRequest(http.MethodPatch, "/gifs/me/key-123", bytes.NewReader(body))
	req.Header.Set("If-Match", ifMatch)
	req = withUserAuth(req, "user-123")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestMediaHandler_Delete_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)

	mockSvc.EXPECT().Delete(gomock.Any(), gomock.Eq("user-123"), gomock.Eq("key-123")).
		Return(nil).Times(1)

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
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)

	mockSvc.EXPECT().Delete(gomock.Any(), gomock.Eq("user-123"), gomock.Eq("missing.gif")).
		Return(domainmedia.ErrGifNotFound).Times(1)

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
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)

	mockSvc.EXPECT().Delete(gomock.Any(), gomock.Eq("user-123"), gomock.Eq("other.gif")).
		Return(domainmedia.ErrGifOwnerMismatch).Times(1)

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
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)

	mockSvc.EXPECT().GetGifThumbnail(gomock.Any(), gomock.Eq("user-123"), gomock.Eq("sample.gif")).
		Return("https://minio/thumbnail.jpg", nil).Times(1)

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
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)

	mockSvc.EXPECT().GetGifThumbnail(gomock.Any(), gomock.Eq("user-123"), gomock.Eq("missing.gif")).
		Return("", domainmedia.ErrGifNotFound).Times(1)

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
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)

	mockSvc.EXPECT().GetGifThumbnail(gomock.Any(), gomock.Eq("user-123"), gomock.Eq("error.gif")).
		Return("", errors.New("storage error")).Times(1)

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
