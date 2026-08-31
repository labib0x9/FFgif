package media_test

// import (
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"errors"
// 	"net/http"
// 	"net/http/httptest"
// 	"testing"
// 	"time"

// 	"github.com/go-playground/validator/v10"
// 	"github.com/google/uuid"
// 	appmedia "github.com/labib0x9/ffgif/internal/app/media"
// 	domainmedia "github.com/labib0x9/ffgif/internal/domain/media"
// 	"github.com/labib0x9/ffgif/internal/port/queue"
// 	"github.com/labib0x9/ffgif/internal/transport/http/handlers/media"
// 	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
// 	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
// )

// type mockMediaService struct {
// 	uploadFunc                func(rctx context.Context, filename string, claims jwtpkg.Payload) (*domainmedia.UploadResult, error)
// 	downloadFunc              func(ctx context.Context, key string) (string, error)
// 	getByKeyFunc              func(ctx context.Context, key string) (domainmedia.GifResponse, error)
// 	getRecentsFunc            func(ctx context.Context, id string) ([]domainmedia.GifResponse, error)
// 	getGifsFunc               func(ctx context.Context, id string, filter string) (*appmedia.GifResult, error)
// 	lastVideoFunc             func(ctx context.Context, userId string) (domainmedia.LastUploadResponse, error)
// 	deleteFunc                func(ctx context.Context, key string) error
// 	saveFunc                  func(ctx context.Context, key string) error
// 	streamFunc                func(ctx context.Context, key string) (*domainmedia.StreamResult, error)
// 	updateFunc                func(ctx context.Context, key string) error
// 	processAndSaveFunc        func(ctx context.Context, key string) error
// 	updateUploadingStatusFunc func(ctx context.Context, key string, status string) error
// 	statusFunc                func(ctx context.Context, key string) (string, error)

// 	convertFunc          func(ctx context.Context, userId string, key string, start float32, end float32, fps int, width int, loop bool) (*appjob.ConvertResult, error)
// 	conversionStatusFunc func(ctx context.Context, jobId string) (*appmedia.StatusResult, error)
// 	processFunc          func(ctx context.Context, msg queue.VideoMessage) error
// 	saveMetadataFunc     func(ctx context.Context, msg queue.SaveVideoMessage) error
// }

// func (m *mockMediaService) Upload(rctx context.Context, filename string, claims jwtpkg.Payload) (*domainmedia.UploadResult, error) {
// 	if m.uploadFunc != nil {
// 		return m.uploadFunc(rctx, filename, claims)
// 	}
// 	return &domainmedia.UploadResult{Url: "https://storage/presigned-put", Key: "upload-123", ExpireIn: 300}, nil
// }
// func (m *mockMediaService) Download(ctx context.Context, key string) (string, error) {
// 	if m.downloadFunc != nil {
// 		return m.downloadFunc(ctx, key)
// 	}
// 	return "https://storage/presigned-get", nil
// }
// func (m *mockMediaService) GetByKey(ctx context.Context, key string) (domainmedia.GifResponse, error) {
// 	if m.getByKeyFunc != nil {
// 		return m.getByKeyFunc(ctx, key)
// 	}
// 	return domainmedia.GifResponse{Key: key, Url: "https://storage/" + key}, nil
// }
// func (m *mockMediaService) GetRecents(ctx context.Context, id string) ([]domainmedia.GifResponse, error) {
// 	if m.getRecentsFunc != nil {
// 		return m.getRecentsFunc(ctx, id)
// 	}
// 	return []domainmedia.GifResponse{}, nil
// }
// func (m *mockMediaService) GetGifs(ctx context.Context, id string, filter string) (*appmedia.GifResult, error) {
// 	if m.getGifsFunc != nil {
// 		return m.getGifsFunc(ctx, id, filter)
// 	}
// 	return &appmedia.GifResult{Data: []domainmedia.GifResponse{}, Total: 0}, nil
// }
// func (m *mockMediaService) LastVideo(ctx context.Context, userId string) (domainmedia.LastUploadResponse, error) {
// 	if m.lastVideoFunc != nil {
// 		return m.lastVideoFunc(ctx, userId)
// 	}
// 	return domainmedia.LastUploadResponse{UserID: uuid.MustParse(userId), FileKey: "last.mp4"}, nil
// }
// func (m *mockMediaService) Delete(ctx context.Context, key string) error {
// 	if m.deleteFunc != nil {
// 		return m.deleteFunc(ctx, key)
// 	}
// 	return nil
// }
// func (m *mockMediaService) Save(ctx context.Context, key string) error {
// 	if m.saveFunc != nil {
// 		return m.saveFunc(ctx, key)
// 	}
// 	return nil
// }
// func (m *mockMediaService) Stream(ctx context.Context, key string) (*domainmedia.StreamResult, error) {
// 	if m.streamFunc != nil {
// 		return m.streamFunc(ctx, key)
// 	}
// 	return &domainmedia.StreamResult{PresignedUrl: "https://storage/stream/" + key, ExpireIn: 300}, nil
// }
// func (m *mockMediaService) Update(ctx context.Context, key string) error {
// 	if m.updateFunc != nil {
// 		return m.updateFunc(ctx, key)
// 	}
// 	return nil
// }
// func (m *mockMediaService) ProcessAndSave(ctx context.Context, key string) error {
// 	if m.processAndSaveFunc != nil {
// 		return m.processAndSaveFunc(ctx, key)
// 	}
// 	return nil
// }
// func (m *mockMediaService) UpdateUploadingStatus(ctx context.Context, key string, status string) error {
// 	if m.updateUploadingStatusFunc != nil {
// 		return m.updateUploadingStatusFunc(ctx, key, status)
// 	}
// 	return nil
// }
// func (m *mockMediaService) Status(ctx context.Context, key string) (string, error) {
// 	if m.statusFunc != nil {
// 		return m.statusFunc(ctx, key)
// 	}
// 	return "ok", nil
// }

// type mockCache struct{}

// func (m *mockCache) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
// 	return nil
// }
// func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
// 	return "", errors.New("not found")
// }

// func TestMediaHandler_Upload_Success(t *testing.T) {
// 	mockSvc := &mockMediaService{
// 		uploadFunc: func(rctx context.Context, filename string, claims jwtpkg.Payload) (*domainmedia.UploadResult, error) {
// 			if filename != "my-video.mp4" {
// 				t.Errorf("expected filename my-video.mp4, got %s", filename)
// 			}
// 			return &domainmedia.UploadResult{
// 				Url:      "https://minio.local/presigned-upload",
// 				Key:      "user-123:video-uuid.mp4",
// 				ExpireIn: 300,
// 			}, nil
// 		},
// 	}

// 	jwtProvider := jwtpkg.NewJwt([]byte("test-secret"))
// 	middlewares := middleware.NewMiddlewares(nil, &mockCache{}, *jwtProvider)
// 	val := validator.New()

// 	handler := media.NewHandler(mockSvc, middlewares, val)

// 	tokenStr, _ := jwtProvider.Create("John", "user-uuid-1", "john@example.com", "user")

// 	body, _ := json.Marshal(map[string]string{
// 		"filename": "my-video.mp4",
// 	})

// 	req := httptest.NewRequest(http.MethodPost, "/uploads", bytes.NewReader(body))
// 	req.Header.Set("Authorization", "Bearer "+tokenStr)
// 	rec := httptest.NewRecorder()

// 	middlewares.Auth(http.HandlerFunc(handler.Upload)).ServeHTTP(rec, req)

// 	if rec.Code != http.StatusCreated {
// 		t.Errorf("expected status 201 Created, got %d. Body: %s", rec.Code, rec.Body.String())
// 	}

// 	var resp domainmedia.UploadResult
// 	json.NewDecoder(rec.Body).Decode(&resp)
// 	if resp.Url != "https://minio.local/presigned-upload" {
// 		t.Errorf("unexpected upload URL: %s", resp.Url)
// 	}
// }

// func TestMediaHandler_Status_Success(t *testing.T) {
// 	mockSvc := &mockMediaService{
// 		statusFunc: func(ctx context.Context, key string) (string, error) {
// 			return "ok", nil
// 		},
// 	}

// 	handler := media.NewHandler(mockSvc, nil, validator.New())

// 	mux := http.NewServeMux()
// 	mux.HandleFunc("GET /uploads/{key}/status", handler.Status)

// 	req := httptest.NewRequest(http.MethodGet, "/uploads/user-1:test.mp4/status", nil)
// 	rec := httptest.NewRecorder()

// 	mux.ServeHTTP(rec, req)

// 	if rec.Code != http.StatusOK {
// 		t.Errorf("expected status 200 OK, got %d", rec.Code)
// 	}
// }

// // import (
// // 	"bytes"
// // 	"context"
// // 	"encoding/json"
// // 	"errors"
// // 	"net/http"
// // 	"net/http/httptest"
// // 	"testing"
// // 	"time"

// // 	"github.com/go-playground/validator/v10"
// // 	"github.com/labib0x9/ffgif/config"
// // 	appjob "github.com/labib0x9/ffgif/internal/app/job"
// // 	domainqueue "github.com/labib0x9/ffgif/internal/domain/queue"
// // 	"github.com/labib0x9/ffgif/internal/transport/http/handlers/job"
// // 	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
// // 	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
// // )

// type mockJobService struct {
// }

// // func (m *mockJobService) Convert(ctx context.Context, userId string, key string, start float32, end float32, fps int, width int, loop bool) (*appjob.ConvertResult, error) {
// // 	if m.convertFunc != nil {
// // 		return m.convertFunc(ctx, userId, key, start, end, fps, width, loop)
// // 	}
// // 	return &appjob.ConvertResult{Id: "job-123", Status: "queued"}, nil
// // }
// // func (m *mockJobService) Status(ctx context.Context, jobId string) (*appjob.StatusResult, error) {
// // 	if m.statusFunc != nil {
// // 		return m.statusFunc(ctx, jobId)
// // 	}
// // 	return &appjob.StatusResult{JobId: jobId, Status: "completed", GifId: "gif-123"}, nil
// // }
// // func (m *mockJobService) Process(ctx context.Context, msg domainqueue.VideoMessage) error {
// // 	if m.processFunc != nil {
// // 		return m.processFunc(ctx, msg)
// // 	}
// // 	return nil
// // }
// // func (m *mockJobService) SaveMetadata(ctx context.Context, msg domainqueue.SaveVideoMessage) error {
// // 	if m.saveMetadataFunc != nil {
// // 		return m.saveMetadataFunc(ctx, msg)
// // 	}
// // 	return nil
// // }

// // type mockCache struct{}

// // func (m *mockCache) Set(ctx context.Context, key string, value string, expire time.Duration) error {
// // 	return nil
// // }
// // func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
// // 	return "", errors.New("not found")
// // }

// // func TestJobHandler_Convert_Success(t *testing.T) {
// // 	mockSvc := &mockJobService{
// // 		convertFunc: func(ctx context.Context, userId string, key string, start float32, end float32, fps int, width int, loop bool) (*appjob.ConvertResult, error) {
// // 			if userId != "user-uuid-123" {
// // 				t.Errorf("expected userId user-uuid-123, got %s", userId)
// // 			}
// // 			return &appjob.ConvertResult{Id: "job-abc", Status: "queued"}, nil
// // 		},
// // 	}

// // 	jwtProvider := jwtpkg.NewJwt([]byte("test-secret"))
// // 	middlewares := middleware.NewMiddlewares(&config.Config{}, &mockCache{}, *jwtProvider)
// // 	val := validator.New()

// // 	handler := job.NewHandler(mockSvc, middlewares, val)

// // 	tokenStr, err := jwtProvider.Create("John", "user-uuid-123", "john@example.com", "user")
// // 	if err != nil {
// // 		t.Fatalf("failed to create jwt: %v", err)
// // 	}

// // 	body, _ := json.Marshal(map[string]any{
// // 		"upload_key": "video_raw.mp4",
// // 		"start_time": 0.0,
// // 		"end_time":   5.0,
// // 		"fps":        10,
// // 		"width":      480,
// // 		"loop":       true,
// // 	})

// // 	req := httptest.NewRequest(http.MethodPost, "/convert", bytes.NewReader(body))
// // 	req.Header.Set("Authorization", "Bearer "+tokenStr)
// // 	rec := httptest.NewRecorder()

// // 	// Wrap with Auth middleware to populate context
// // 	middlewares.Auth(http.HandlerFunc(handler.Convert)).ServeHTTP(rec, req)

// // 	if rec.Code != http.StatusOK {
// // 		t.Errorf("expected status 200 OK, got %d. Body: %s", rec.Code, rec.Body.String())
// // 	}

// // 	var resp map[string]string
// // 	json.NewDecoder(rec.Body).Decode(&resp)
// // 	if resp["job_id"] != "job-abc" || resp["status"] != "queued" {
// // 		t.Errorf("unexpected response: %v", resp)
// // 	}
// // }

// // func TestJobHandler_Status_Success(t *testing.T) {
// // 	mockSvc := &mockJobService{
// // 		statusFunc: func(ctx context.Context, jobId string) (*appjob.StatusResult, error) {
// // 			return &appjob.StatusResult{JobId: jobId, Status: "completed", GifId: "gif-final.gif"}, nil
// // 		},
// // 	}

// // 	handler := job.NewHandler(mockSvc, nil, validator.New())

// // 	mux := http.NewServeMux()
// // 	mux.HandleFunc("GET /convert/{jobId}/status", handler.Status)

// // 	req := httptest.NewRequest(http.MethodGet, "/convert/job-xyz/status", nil)
// // 	rec := httptest.NewRecorder()

// // 	mux.ServeHTTP(rec, req)

// // 	if rec.Code != http.StatusOK {
// // 		t.Errorf("expected status 200 OK, got %d", rec.Code)
// // 	}

// // 	var resp appjob.StatusResult
// // 	json.NewDecoder(rec.Body).Decode(&resp)
// // 	if resp.Status != "completed" || resp.GifId != "gif-final.gif" {
// // 		t.Errorf("unexpected response: %+v", resp)
// // 	}
// // }
