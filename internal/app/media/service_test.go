package media_test

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labib0x9/ffgif/config"
	appmedia "github.com/labib0x9/ffgif/internal/app/media"
	domainauth "github.com/labib0x9/ffgif/internal/domain/auth"
	domainmedia "github.com/labib0x9/ffgif/internal/domain/media"
	domainshare "github.com/labib0x9/ffgif/internal/domain/share"
	domainuser "github.com/labib0x9/ffgif/internal/domain/user"
	"github.com/labib0x9/ffgif/internal/port/processor"
	"github.com/labib0x9/ffgif/internal/port/queue"
	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
	amqp "github.com/rabbitmq/amqp091-go"
)

// --- Mocks ---

type mockStorageRepo struct {
	createFunc          func(ctx context.Context, key string, expirey time.Duration) (*url.URL, error)
	downloadFunc        func(ctx context.Context, key string, expirey time.Duration) (*url.URL, error)
	getStreamURLFunc    func(ctx context.Context, key string, expiry time.Duration) (*url.URL, error)
	getThumbnailURLFunc func(ctx context.Context, key string) (*url.URL, error)
	statusFunc          func(ctx context.Context, key string) (domainmedia.Info, error)
}

func (m *mockStorageRepo) Create(ctx context.Context, key string, expirey time.Duration) (*url.URL, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, key, expirey)
	}
	u, _ := url.Parse("https://storage.local/upload/" + key)
	return u, nil
}
func (m *mockStorageRepo) Download(ctx context.Context, key string, expirey time.Duration) (*url.URL, error) {
	if m.downloadFunc != nil {
		return m.downloadFunc(ctx, key, expirey)
	}
	u, _ := url.Parse("https://storage.local/download/" + key)
	return u, nil
}
func (m *mockStorageRepo) IsExists(ctx context.Context, key string) (bool, error) { return true, nil }
func (m *mockStorageRepo) Status(ctx context.Context, key string) (domainmedia.Info, error) {
	if m.statusFunc != nil {
		return m.statusFunc(ctx, key)
	}
	return domainmedia.Info{Size: 1024, ContentType: "video/mp4", UploadedAt: time.Now()}, nil
}
func (m *mockStorageRepo) GetObject(ctx context.Context, start, end int64, key string) (domainmedia.Object, error) {
	return domainmedia.Object{}, nil
}
func (m *mockStorageRepo) DownloadLocal(ctx context.Context, key, destPath string) error { return nil }
func (m *mockStorageRepo) DownloadLocalRawVideo(ctx context.Context, key, destPath string) error {
	return nil
}
func (m *mockStorageRepo) Upload(ctx context.Context, key, filePath, contentType string) error {
	return nil
}
func (m *mockStorageRepo) Delete(ctx context.Context, key string) error { return nil }
func (m *mockStorageRepo) GetStreamURL(ctx context.Context, key string, expiry time.Duration) (*url.URL, error) {
	if m.getStreamURLFunc != nil {
		return m.getStreamURLFunc(ctx, key, expiry)
	}
	u, _ := url.Parse("https://storage.local/stream/" + key)
	return u, nil
}
func (m *mockStorageRepo) GetThumbnailURL(ctx context.Context, key string) (*url.URL, error) {
	if m.getThumbnailURLFunc != nil {
		return m.getThumbnailURLFunc(ctx, key)
	}
	u, _ := url.Parse("https://storage.local/thumbnail/" + key)
	return u, nil
}

type mockGifRepo struct {
	getOwnerFunc   func(ctx context.Context, key string) (string, error)
	getByKeyFunc   func(ctx context.Context, key string) (domainmedia.GifResponse, error)
	getRecentsFunc func(ctx context.Context, user_id string) ([]domainmedia.GifResponse, error)
	getFunc        func(ctx context.Context, user_id string, status string) ([]domainmedia.GifResponse, error)
	deleteFunc     func(ctx context.Context, key string) error
	createFunc     func(ctx context.Context, gif domainmedia.Gif) error
	updateFunc     func(ctx context.Context, key string, gif domainmedia.GifResponse) error
	saveRecentFunc func(ctx context.Context, key string) error
}

func (m *mockGifRepo) GetOwner(ctx context.Context, key string) (string, error) {
	if m.getOwnerFunc != nil {
		return m.getOwnerFunc(ctx, key)
	}
	return "owner-uuid-1", nil
}
func (m *mockGifRepo) Create(ctx context.Context, gif domainmedia.Gif) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, gif)
	}
	return nil
}
func (m *mockGifRepo) Get(ctx context.Context, user_id string, status string) ([]domainmedia.GifResponse, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, user_id, status)
	}
	return []domainmedia.GifResponse{}, nil
}
func (m *mockGifRepo) GetByKey(ctx context.Context, key string) (domainmedia.GifResponse, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key)
	}
	return domainmedia.GifResponse{Key: key, Url: "https://storage/gif/" + key}, nil
}
func (m *mockGifRepo) GetRecents(ctx context.Context, user_id string) ([]domainmedia.GifResponse, error) {
	if m.getRecentsFunc != nil {
		return m.getRecentsFunc(ctx, user_id)
	}
	return []domainmedia.GifResponse{}, nil
}
func (m *mockGifRepo) Delete(ctx context.Context, key string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, key)
	}
	return nil
}
func (m *mockGifRepo) Update(ctx context.Context, key string, gif domainmedia.GifResponse) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, key, gif)
	}
	return nil
}
func (m *mockGifRepo) SaveRecent(ctx context.Context, key string) error {
	if m.saveRecentFunc != nil {
		return m.saveRecentFunc(ctx, key)
	}
	return nil
}

type mockShareRepo struct {
	getOwnerFunc func(ctx context.Context, sharedWithUserId string, gifKey string) (string, error)
}

func (m *mockShareRepo) Create(ctx context.Context, gif domainshare.Share) error {
	return nil
}
func (m *mockShareRepo) Get(ctx context.Context, userID string) ([]domainshare.GifResponse, error) {
	return nil, nil
}
func (m *mockShareRepo) GetOwner(ctx context.Context, sharedWithUserId string, gifKey string) (string, error) {
	if m.getOwnerFunc != nil {
		return m.getOwnerFunc(ctx, sharedWithUserId, gifKey)
	}
	return "", sql.ErrNoRows
}
func (m *mockShareRepo) Delete(ctx context.Context, key, shareWithId string) error {
	return nil
}

type mockLastVideoRepo struct {
	createFunc       func(ctx context.Context, upload domainmedia.LastUpload) error
	getLastVideoFunc func(ctx context.Context, user_id string) (domainmedia.LastUploadResponse, error)
}

func (m *mockLastVideoRepo) Create(ctx context.Context, upload domainmedia.LastUpload) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, upload)
	}
	return nil
}
func (m *mockLastVideoRepo) GetLastVideo(ctx context.Context, user_id string) (domainmedia.LastUploadResponse, error) {
	if m.getLastVideoFunc != nil {
		return m.getLastVideoFunc(ctx, user_id)
	}
	return domainmedia.LastUploadResponse{UserID: uuid.MustParse(user_id), FileKey: "last-video.mp4"}, nil
}

type mockProcessor struct {
	preProcessFunc func(ctx context.Context, key string) (*processor.PrePrecessedResult, error)
	processFunc    func(ctx context.Context, JobId string, Key string, Start float32, End float32, Width int, FPS int, Loop bool) (*processor.JobResult, error)
}

func (m *mockProcessor) Process(ctx context.Context, JobId string, Key string, Start float32, End float32, Width int, FPS int, Loop bool) (*processor.JobResult, error) {
	if m.processFunc != nil {
		return m.processFunc(ctx, JobId, Key, Start, End, Width, FPS, Loop)
	}
	return &processor.JobResult{GifKey: "output.gif", ThumbKey: "thumb.jpg"}, nil
}
func (m *mockProcessor) PreProcess(ctx context.Context, key string) (*processor.PrePrecessedResult, error) {
	if m.preProcessFunc != nil {
		return m.preProcessFunc(ctx, key)
	}
	return &processor.PrePrecessedResult{
		VideoKey:     "converted_video.mp4",
		ThumbnailKey: "thumb.jpg",
		ContentType:  "video/mp4",
		Size:         "2048",
		Duration:     "10.0",
	}, nil
}

type mockCache struct {
	store   map[string]string
	setFunc func(ctx context.Context, key string, value string, expiration time.Duration) error
	getFunc func(ctx context.Context, key string) (string, error)
}

func newMockCache() *mockCache {
	return &mockCache{store: make(map[string]string)}
}

func (m *mockCache) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	if m.setFunc != nil {
		return m.setFunc(ctx, key, value, expiration)
	}
	m.store[key] = value
	return nil
}
func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, key)
	}
	if v, ok := m.store[key]; ok {
		return v, nil
	}
	return "", errors.New("not found in cache")
}

type mockQueue struct {
	publishVideoFunc          func(ctx context.Context, msg queue.VideoMessage) error
	publishRetrySaveVideoFunc func(ctx context.Context, msg queue.SaveVideoMessage) error
}

func (m *mockQueue) PublishEmail(ctx context.Context, msg queue.EmailMessage) error { return nil }
func (m *mockQueue) PublishVideo(ctx context.Context, msg queue.VideoMessage) error {
	if m.publishVideoFunc != nil {
		return m.publishVideoFunc(ctx, msg)
	}
	return nil
}
func (m *mockQueue) PublishSaveVideo(ctx context.Context, msg queue.SaveVideoMessage) error {
	return nil
}
func (m *mockQueue) PublishRetrySaveVideo(ctx context.Context, msg queue.SaveVideoMessage) error {
	if m.publishRetrySaveVideoFunc != nil {
		return m.publishRetrySaveVideoFunc(ctx, msg)
	}
	return nil
}
func (m *mockQueue) ConsumeEmail(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error) {
	return nil, nil
}
func (m *mockQueue) ConsumeSave(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error) {
	return nil, nil
}
func (m *mockQueue) ConsumeVideo(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error) {
	return nil, nil
}
func (m *mockQueue) ConsumeRawVideo(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error) {
	return nil, nil
}
func (m *mockQueue) Close() error                           { return nil }
func (m *mockQueue) CloseConsumerChannel(name string) error { return nil }

type mockAuthRepo struct{}

func (m *mockAuthRepo) GetByEmail(ctx context.Context, email string) (domainauth.User, error) {
	return domainauth.User{}, nil
}
func (m *mockAuthRepo) GetById(ctx context.Context, id uuid.UUID) (domainauth.User, error) {
	return domainauth.User{}, nil
}
func (m *mockAuthRepo) Create(ctx context.Context, user domainauth.User) (domainauth.User, error) {
	return user, nil
}
func (m *mockAuthRepo) DeleteById(ctx context.Context, id uuid.UUID) error    { return nil }
func (m *mockAuthRepo) DeleteByEmail(ctx context.Context, email string) error { return nil }
func (m *mockAuthRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passHash string) error {
	return nil
}
func (m *mockAuthRepo) SetVerified(ctx context.Context, userId uuid.UUID) error { return nil }
func (m *mockAuthRepo) Upgrade(ctx context.Context, id string, user domainauth.User) (domainauth.User, error) {
	return user, nil
}

type mockUserRepo struct{}

func (m *mockUserRepo) GetProfile(ctx context.Context, id string) (domainuser.ProfileResponse, error) {
	return domainuser.ProfileResponse{}, nil
}
func (m *mockUserRepo) UpdateProfile(ctx context.Context, profile domainuser.ProfileResponse, id string) (domainuser.ProfileResponse, error) {
	return profile, nil
}
func (m *mockUserRepo) SetProfile(ctx context.Context, profile domainuser.Profile) error { return nil }
func (m *mockUserRepo) ChangePassword(ctx context.Context, userId string, hash string) error {
	return nil
}

type mockQuotaRepo struct{}

func (m *mockQuotaRepo) Create(ctx context.Context, quota domainuser.Quota) error { return nil }
func (m *mockQuotaRepo) GetById(ctx context.Context, id string) (*domainuser.Quota, error) {
	return &domainuser.Quota{}, nil
}

func newTestMediaService(
	gifRepo *mockGifRepo,
	shareRepo *mockShareRepo,
	lastVideoRepo *mockLastVideoRepo,
	storage *mockStorageRepo,
	queueMock *mockQueue,
	cache *mockCache,
	proc *mockProcessor,
) appmedia.Service {
	if gifRepo == nil {
		gifRepo = &mockGifRepo{}
	}
	if shareRepo == nil {
		shareRepo = &mockShareRepo{}
	}
	if lastVideoRepo == nil {
		lastVideoRepo = &mockLastVideoRepo{}
	}
	if storage == nil {
		storage = &mockStorageRepo{}
	}
	if queueMock == nil {
		queueMock = &mockQueue{}
	}
	if cache == nil {
		cache = newMockCache()
	}
	if proc == nil {
		proc = &mockProcessor{}
	}
	return appmedia.NewService(
		&mockAuthRepo{},
		&mockUserRepo{},
		&mockQuotaRepo{},
		gifRepo,
		shareRepo,
		lastVideoRepo,
		storage,
		queueMock,
		cache,
		proc,
		&config.Config{},
	)
}

// --- Tests ---

func TestMediaService_Upload_Success(t *testing.T) {
	cache := newMockCache()
	storage := &mockStorageRepo{
		createFunc: func(ctx context.Context, key string, expirey time.Duration) (*url.URL, error) {
			return url.Parse("https://storage.local/upload/" + key)
		},
	}

	svc := newTestMediaService(nil, nil, nil, storage, nil, cache, nil)

	claims := jwtpkg.Payload{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "user-123",
		},
	}

	result, err := svc.Upload(context.Background(), "sample.mp4", claims)
	if err != nil {
		t.Fatalf("expected Upload to succeed, got error: %v", err)
	}

	if result.Url == "" || result.Key == "" {
		t.Errorf("expected non-empty URL and Key in upload result")
	}
	if status, _ := cache.Get(context.Background(), "uploading:"+result.Key); status != "uploading" {
		t.Errorf("expected uploading status cached, got %s", status)
	}
}

func TestMediaService_Upload_StorageError(t *testing.T) {
	storage := &mockStorageRepo{
		createFunc: func(ctx context.Context, key string, expirey time.Duration) (*url.URL, error) {
			return nil, errors.New("minio down")
		},
	}
	svc := newTestMediaService(nil, nil, nil, storage, nil, nil, nil)

	claims := jwtpkg.Payload{RegisteredClaims: jwt.RegisteredClaims{Subject: "user-123"}}
	_, err := svc.Upload(context.Background(), "video.mp4", claims)
	if err == nil {
		t.Fatal("expected error on storage failure, got nil")
	}
}

func TestMediaService_Status_Success(t *testing.T) {
	cache := newMockCache()
	cache.Set(context.Background(), "uploading:test-key.mp4", "completed", time.Minute)

	svc := newTestMediaService(nil, nil, nil, nil, nil, cache, nil)

	status, err := svc.Status(context.Background(), "test-key.mp4")
	if err != nil {
		t.Fatalf("expected Status to succeed, got %v", err)
	}
	if status != "completed" {
		t.Errorf("expected completed status, got %s", status)
	}
}

func TestMediaService_Status_CacheEmptyDefaultsFailed(t *testing.T) {
	cache := newMockCache()
	cache.Set(context.Background(), "uploading:test-key.mp4", "", time.Minute)

	svc := newTestMediaService(nil, nil, nil, nil, nil, cache, nil)

	status, err := svc.Status(context.Background(), "test-key.mp4")
	if err != nil {
		t.Fatalf("expected Status to succeed, got %v", err)
	}
	if status != "failed" {
		t.Errorf("expected failed status for empty string, got %s", status)
	}
}

func TestMediaService_ProcessAndSave_Success(t *testing.T) {
	userId := uuid.New().String()
	uploadKey := userId + ":video-uuid.mp4"
	saved := false

	lastVideoRepo := &mockLastVideoRepo{
		createFunc: func(ctx context.Context, upload domainmedia.LastUpload) error {
			saved = true
			if upload.UserID.String() != userId {
				t.Errorf("expected userID %s, got %s", userId, upload.UserID)
			}
			return nil
		},
	}

	svc := newTestMediaService(nil, nil, lastVideoRepo, nil, nil, nil, nil)

	err := svc.ProcessAndSave(context.Background(), uploadKey)
	if err != nil {
		t.Fatalf("expected ProcessAndSave to succeed, got: %v", err)
	}
	if !saved {
		t.Errorf("expected LastUpload to be created")
	}
}

func TestMediaService_ProcessAndSave_EmptyKey(t *testing.T) {
	svc := newTestMediaService(nil, nil, nil, nil, nil, nil, nil)

	err := svc.ProcessAndSave(context.Background(), "")
	if !errors.Is(err, domainmedia.ErrEmptyKey) {
		t.Errorf("expected ErrEmptyKey, got %v", err)
	}
}

func TestMediaService_ProcessAndSave_PreprocessError(t *testing.T) {
	proc := &mockProcessor{
		preProcessFunc: func(ctx context.Context, key string) (*processor.PrePrecessedResult, error) {
			return nil, errors.New("corrupt video")
		},
	}
	svc := newTestMediaService(nil, nil, nil, nil, nil, nil, proc)

	err := svc.ProcessAndSave(context.Background(), uuid.New().String()+":file.mp4")
	if !errors.Is(err, domainmedia.ErrInvalidFiletype) {
		t.Errorf("expected ErrInvalidFiletype, got %v", err)
	}
}

func TestMediaService_SaveMetadata_Success(t *testing.T) {
	saved := false
	validUUID := uuid.New().String()

	lastVideoRepo := &mockLastVideoRepo{
		createFunc: func(ctx context.Context, upload domainmedia.LastUpload) error {
			saved = true
			if upload.FileKey != "valid-key.mp4" {
				t.Errorf("expected file key 'valid-key.mp4', got %s", upload.FileKey)
			}
			return nil
		},
	}

	svc := newTestMediaService(nil, nil, lastVideoRepo, nil, nil, nil, nil)

	msg := queue.SaveVideoMessage{
		UserID:   validUUID,
		Key:      "valid-key.mp4",
		Filename: "video.mp4",
	}

	err := svc.SaveMetadata(context.Background(), msg)
	if err != nil {
		t.Fatalf("expected SaveMetadata to succeed, got: %v", err)
	}
	if !saved {
		t.Errorf("expected last video metadata to be saved")
	}
}

func TestMediaService_SaveMetadata_InvalidUserID(t *testing.T) {
	svc := newTestMediaService(nil, nil, nil, nil, nil, nil, nil)

	msg := queue.SaveVideoMessage{
		UserID:   "invalid-uuid",
		Key:      "key-123",
		Filename: "test.mp4",
	}

	err := svc.SaveMetadata(context.Background(), msg)
	if !errors.Is(err, domainmedia.ErrInvalidUserID) {
		t.Errorf("expected ErrInvalidUserID, got %v", err)
	}
}

func TestMediaService_SaveMetadata_StorageStatusRetry(t *testing.T) {
	retried := false
	storage := &mockStorageRepo{
		statusFunc: func(ctx context.Context, key string) (domainmedia.Info, error) {
			return domainmedia.Info{}, errors.New("object not found yet")
		},
	}
	queueMock := &mockQueue{
		publishRetrySaveVideoFunc: func(ctx context.Context, msg queue.SaveVideoMessage) error {
			retried = true
			return nil
		},
	}

	svc := newTestMediaService(nil, nil, nil, storage, queueMock, nil, nil)

	msg := queue.SaveVideoMessage{
		UserID:   uuid.New().String(),
		Key:      "raw.mp4",
		Filename: "raw.mp4",
	}

	err := svc.SaveMetadata(context.Background(), msg)
	if err == nil {
		t.Fatal("expected error from status fetch, got nil")
	}
	if !retried {
		t.Errorf("expected retry message to be published to queue")
	}
}

func TestMediaService_Convert_Success(t *testing.T) {
	cache := newMockCache()
	videoPublished := false
	queueMock := &mockQueue{
		publishVideoFunc: func(ctx context.Context, msg queue.VideoMessage) error {
			videoPublished = true
			if msg.Key != "test_key.mp4" {
				t.Errorf("expected key test_key.mp4, got %s", msg.Key)
			}
			return nil
		},
	}

	svc := newTestMediaService(nil, nil, nil, nil, queueMock, cache, nil)

	res, err := svc.Convert(context.Background(), "user-123", "test_key.mp4", 0.0, 5.0, 15, 480, true)
	if err != nil {
		t.Fatalf("expected Convert to succeed, got error: %v", err)
	}

	if res.Id == "" {
		t.Errorf("expected non-empty JobId")
	}
	if res.Status != "queued" {
		t.Errorf("expected status 'queued', got %s", res.Status)
	}
	if !videoPublished {
		t.Errorf("expected video message to be published to queue")
	}
}

func TestMediaService_Convert_QueueFailure(t *testing.T) {
	queueMock := &mockQueue{
		publishVideoFunc: func(ctx context.Context, msg queue.VideoMessage) error {
			return errors.New("amqp connection closed")
		},
	}
	svc := newTestMediaService(nil, nil, nil, nil, queueMock, nil, nil)

	_, err := svc.Convert(context.Background(), "user-1", "test.mp4", 0, 5, 10, 320, true)
	if !errors.Is(err, domainmedia.ErrMessageQueueFailed) {
		t.Errorf("expected ErrMessageQueueFailed, got %v", err)
	}
}

func TestMediaService_ConversionStatus_Success(t *testing.T) {
	cache := newMockCache()
	cache.Set(context.Background(), "messaage_queue:job_id:job-789", "completed", time.Minute)
	cache.Set(context.Background(), "messaage_queue_gif:job_id:job-789", "gif-result-key.gif", time.Minute)

	svc := newTestMediaService(nil, nil, nil, nil, nil, cache, nil)

	result, err := svc.ConversionStatus(context.Background(), "job-789")
	if err != nil {
		t.Fatalf("expected ConversionStatus to succeed, got: %v", err)
	}

	if result.Status != "completed" {
		t.Errorf("expected status 'completed', got %s", result.Status)
	}
	if result.GifId != "gif-result-key.gif" {
		t.Errorf("expected gifId 'gif-result-key.gif', got %s", result.GifId)
	}
}

func TestMediaService_ConversionStatus_NotFound(t *testing.T) {
	cache := newMockCache()
	svc := newTestMediaService(nil, nil, nil, nil, nil, cache, nil)

	_, err := svc.ConversionStatus(context.Background(), "non-existent-job")
	if !errors.Is(err, domainmedia.ErrCacheGetFailed) {
		t.Errorf("expected ErrCacheGetFailed, got %v", err)
	}
}

func TestMediaService_Process_Success(t *testing.T) {
	cache := newMockCache()
	createdGif := false
	gifRepo := &mockGifRepo{
		createFunc: func(ctx context.Context, gif domainmedia.Gif) error {
			createdGif = true
			if gif.Key != "output.gif" {
				t.Errorf("expected gif key output.gif, got %s", gif.Key)
			}
			return nil
		},
	}
	proc := &mockProcessor{
		processFunc: func(ctx context.Context, JobId string, Key string, Start float32, End float32, Width int, FPS int, Loop bool) (*processor.JobResult, error) {
			return &processor.JobResult{GifKey: "output.gif", ThumbKey: "thumb.jpg"}, nil
		},
	}

	svc := newTestMediaService(gifRepo, nil, nil, nil, nil, cache, proc)

	msg := queue.VideoMessage{
		UserID: "user-123",
		JobId:  "job-456",
		Key:    "raw.mp4",
		Start:  0,
		End:    3,
		FPS:    10,
		Width:  320,
		Loop:   true,
	}

	err := svc.Process(context.Background(), msg)
	if err != nil {
		t.Fatalf("expected Process to succeed, got: %v", err)
	}

	if !createdGif {
		t.Errorf("expected gif record to be created in repository")
	}

	status, _ := cache.Get(context.Background(), "messaage_queue:job_id:job-456")
	if status != "completed" {
		t.Errorf("expected status completed, got %s", status)
	}
}

func TestMediaService_Process_ProcessorFailure(t *testing.T) {
	cache := newMockCache()
	proc := &mockProcessor{
		processFunc: func(ctx context.Context, JobId string, Key string, Start float32, End float32, Width int, FPS int, Loop bool) (*processor.JobResult, error) {
			return nil, errors.New("ffmpeg transcoding failed")
		},
	}

	svc := newTestMediaService(nil, nil, nil, nil, nil, cache, proc)

	msg := queue.VideoMessage{
		UserID: "user-1",
		JobId:  "job-fail-1",
		Key:    "video.mp4",
	}

	err := svc.Process(context.Background(), msg)
	if err == nil {
		t.Fatal("expected error on ffmpeg failure, got nil")
	}

	status, _ := cache.Get(context.Background(), "messaage_queue:job_id:job-fail-1")
	if status != "failed" {
		t.Errorf("expected status failed in cache, got %s", status)
	}
}

func TestMediaService_GetGifs_Success(t *testing.T) {
	gifRepo := &mockGifRepo{
		getFunc: func(ctx context.Context, user_id string, status string) ([]domainmedia.GifResponse, error) {
			return []domainmedia.GifResponse{
				{Key: "gif-1.gif", Url: "https://minio/1.gif"},
				{Key: "gif-2.gif", Url: "https://minio/2.gif"},
			}, nil
		},
	}
	svc := newTestMediaService(gifRepo, nil, nil, nil, nil, nil, nil)

	res, err := svc.GetGifs(context.Background(), "user-1", "all")
	if err != nil {
		t.Fatalf("expected GetGifs to succeed, got %v", err)
	}
	if res.Total != 2 {
		t.Errorf("expected 2 gifs, got %d", res.Total)
	}
}

func TestMediaService_GetGifs_RepoError(t *testing.T) {
	gifRepo := &mockGifRepo{
		getFunc: func(ctx context.Context, user_id string, status string) ([]domainmedia.GifResponse, error) {
			return nil, errors.New("db error")
		},
	}
	svc := newTestMediaService(gifRepo, nil, nil, nil, nil, nil, nil)

	_, err := svc.GetGifs(context.Background(), "user-1", "all")
	if !errors.Is(err, domainmedia.ErrGifFetchFailed) {
		t.Errorf("expected ErrGifFetchFailed, got %v", err)
	}
}

func TestMediaService_GetByKey_Success(t *testing.T) {
	gifRepo := &mockGifRepo{
		getByKeyFunc: func(ctx context.Context, key string) (domainmedia.GifResponse, error) {
			return domainmedia.GifResponse{Key: key, Url: "https://minio/" + key}, nil
		},
	}
	svc := newTestMediaService(gifRepo, nil, nil, nil, nil, nil, nil)

	gif, err := svc.GetByKey(context.Background(), "my-gif.gif")
	if err != nil {
		t.Fatalf("expected GetByKey to succeed, got %v", err)
	}
	if gif.Key != "my-gif.gif" {
		t.Errorf("expected key my-gif.gif, got %s", gif.Key)
	}
}

func TestMediaService_GetRecents_Success(t *testing.T) {
	gifRepo := &mockGifRepo{
		getRecentsFunc: func(ctx context.Context, user_id string) ([]domainmedia.GifResponse, error) {
			return []domainmedia.GifResponse{{Key: "recent.gif"}}, nil
		},
	}
	svc := newTestMediaService(gifRepo, nil, nil, nil, nil, nil, nil)

	recents, err := svc.GetRecents(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("expected GetRecents to succeed, got %v", err)
	}
	if len(recents) != 1 {
		t.Errorf("expected 1 recent gif, got %d", len(recents))
	}
}

func TestMediaService_LastVideo_Success(t *testing.T) {
	userId := uuid.New().String()
	lastVideoRepo := &mockLastVideoRepo{
		getLastVideoFunc: func(ctx context.Context, uid string) (domainmedia.LastUploadResponse, error) {
			return domainmedia.LastUploadResponse{
				UserID:  uuid.MustParse(uid),
				FileKey: "last-upload.mp4",
			}, nil
		},
	}
	svc := newTestMediaService(nil, nil, lastVideoRepo, nil, nil, nil, nil)

	last, err := svc.LastVideo(context.Background(), userId)
	if err != nil {
		t.Fatalf("expected LastVideo to succeed, got %v", err)
	}
	if last.FileKey != "last-upload.mp4" {
		t.Errorf("expected last-upload.mp4, got %s", last.FileKey)
	}
}

func TestMediaService_LastVideo_NotFound(t *testing.T) {
	lastVideoRepo := &mockLastVideoRepo{
		getLastVideoFunc: func(ctx context.Context, uid string) (domainmedia.LastUploadResponse, error) {
			return domainmedia.LastUploadResponse{}, sql.ErrNoRows
		},
	}
	svc := newTestMediaService(nil, nil, lastVideoRepo, nil, nil, nil, nil)

	_, err := svc.LastVideo(context.Background(), uuid.New().String())
	if !errors.Is(err, domainmedia.ErrLastVideoNotFound) {
		t.Errorf("expected ErrLastVideoNotFound, got %v", err)
	}
}

func TestMediaService_Download_SuccessAsOwner(t *testing.T) {
	gifRepo := &mockGifRepo{
		getOwnerFunc: func(ctx context.Context, key string) (string, error) {
			return "user-owner-1", nil
		},
	}
	storage := &mockStorageRepo{
		downloadFunc: func(ctx context.Context, key string, expirey time.Duration) (*url.URL, error) {
			return url.Parse("https://minio.download/" + key)
		},
	}
	svc := newTestMediaService(gifRepo, nil, nil, storage, nil, nil, nil)

	downloadUrl, err := svc.Download(context.Background(), "user-owner-1", "my-gif.gif")
	if err != nil {
		t.Fatalf("expected Download to succeed, got %v", err)
	}
	if downloadUrl != "https://minio.download/my-gif.gif" {
		t.Errorf("unexpected download URL: %s", downloadUrl)
	}
}

func TestMediaService_Download_SuccessAsSharedUser(t *testing.T) {
	gifRepo := &mockGifRepo{
		getOwnerFunc: func(ctx context.Context, key string) (string, error) {
			return "user-owner-1", nil
		},
	}
	shareRepo := &mockShareRepo{
		getOwnerFunc: func(ctx context.Context, sharedWithUserId string, gifKey string) (string, error) {
			if sharedWithUserId == "user-shared-2" && gifKey == "my-gif.gif" {
				return "user-owner-1", nil
			}
			return "", sql.ErrNoRows
		},
	}
	storage := &mockStorageRepo{
		downloadFunc: func(ctx context.Context, key string, expirey time.Duration) (*url.URL, error) {
			return url.Parse("https://minio.download/" + key)
		},
	}
	svc := newTestMediaService(gifRepo, shareRepo, nil, storage, nil, nil, nil)

	downloadUrl, err := svc.Download(context.Background(), "user-shared-2", "my-gif.gif")
	if err != nil {
		t.Fatalf("expected Download to succeed for shared user, got %v", err)
	}
	if downloadUrl != "https://minio.download/my-gif.gif" {
		t.Errorf("unexpected download URL: %s", downloadUrl)
	}
}

func TestMediaService_Download_NotFound(t *testing.T) {
	gifRepo := &mockGifRepo{
		getOwnerFunc: func(ctx context.Context, key string) (string, error) {
			return "", sql.ErrNoRows
		},
	}
	svc := newTestMediaService(gifRepo, nil, nil, nil, nil, nil, nil)

	_, err := svc.Download(context.Background(), "user-1", "non-existent.gif")
	if !errors.Is(err, domainmedia.ErrGifNotFound) {
		t.Errorf("expected ErrGifNotFound, got %v", err)
	}
}

func TestMediaService_Download_OwnerMismatch(t *testing.T) {
	gifRepo := &mockGifRepo{
		getOwnerFunc: func(ctx context.Context, key string) (string, error) {
			return "user-owner-1", nil
		},
	}
	shareRepo := &mockShareRepo{
		getOwnerFunc: func(ctx context.Context, sharedWithUserId string, gifKey string) (string, error) {
			return "", sql.ErrNoRows
		},
	}
	svc := newTestMediaService(gifRepo, shareRepo, nil, nil, nil, nil, nil)

	_, err := svc.Download(context.Background(), "unauthorized-user-3", "my-gif.gif")
	if !errors.Is(err, domainmedia.ErrGifOwnerMismatch) {
		t.Errorf("expected ErrGifOwnerMismatch, got %v", err)
	}
}

func TestMediaService_Stream_Success(t *testing.T) {
	storage := &mockStorageRepo{
		getStreamURLFunc: func(ctx context.Context, key string, expiry time.Duration) (*url.URL, error) {
			return url.Parse("https://minio.stream/" + key)
		},
	}
	svc := newTestMediaService(nil, nil, nil, storage, nil, nil, nil)

	res, err := svc.Stream(context.Background(), "video.mp4")
	if err != nil {
		t.Fatalf("expected Stream to succeed, got %v", err)
	}
	if res.PresignedUrl != "https://minio.stream/video.mp4" {
		t.Errorf("unexpected presigned URL: %s", res.PresignedUrl)
	}
}

func TestMediaService_Save_Success(t *testing.T) {
	saved := false
	gifRepo := &mockGifRepo{
		saveRecentFunc: func(ctx context.Context, key string) error {
			saved = true
			return nil
		},
	}
	svc := newTestMediaService(gifRepo, nil, nil, nil, nil, nil, nil)

	err := svc.Save(context.Background(), "user-1", "gif-key-1")
	if err != nil {
		t.Fatalf("expected Save to succeed, got %v", err)
	}
	if !saved {
		t.Errorf("expected SaveRecent to be called")
	}
}

func TestMediaService_Update_Success(t *testing.T) {
	updated := false
	gifRepo := &mockGifRepo{
		updateFunc: func(ctx context.Context, key string, gif domainmedia.GifResponse) error {
			updated = true
			return nil
		},
	}
	svc := newTestMediaService(gifRepo, nil, nil, nil, nil, nil, nil)

	err := svc.Update(context.Background(), "user-1", "gif-key-1")
	if err != nil {
		t.Fatalf("expected Update to succeed, got %v", err)
	}
	if !updated {
		t.Errorf("expected Update to be called")
	}
}

func TestMediaService_Delete_Success(t *testing.T) {
	deleted := false
	gifRepo := &mockGifRepo{
		getOwnerFunc: func(ctx context.Context, key string) (string, error) {
			return "user-1", nil
		},
		deleteFunc: func(ctx context.Context, key string) error {
			deleted = true
			return nil
		},
	}
	svc := newTestMediaService(gifRepo, nil, nil, nil, nil, nil, nil)

	err := svc.Delete(context.Background(), "user-1", "my-gif.gif")
	if err != nil {
		t.Fatalf("expected Delete to succeed, got %v", err)
	}
	if !deleted {
		t.Errorf("expected gif to be deleted")
	}
}

func TestMediaService_Delete_NotFound(t *testing.T) {
	gifRepo := &mockGifRepo{
		getOwnerFunc: func(ctx context.Context, key string) (string, error) {
			return "", sql.ErrNoRows
		},
	}
	svc := newTestMediaService(gifRepo, nil, nil, nil, nil, nil, nil)

	err := svc.Delete(context.Background(), "user-1", "missing.gif")
	if !errors.Is(err, domainmedia.ErrGifNotFound) {
		t.Errorf("expected ErrGifNotFound, got %v", err)
	}
}

func TestMediaService_Delete_OwnerMismatch(t *testing.T) {
	gifRepo := &mockGifRepo{
		getOwnerFunc: func(ctx context.Context, key string) (string, error) {
			return "another-user", nil
		},
	}
	svc := newTestMediaService(gifRepo, nil, nil, nil, nil, nil, nil)

	err := svc.Delete(context.Background(), "user-1", "my-gif.gif")
	if !errors.Is(err, domainmedia.ErrGifOwnerMismatch) {
		t.Errorf("expected ErrGifOwnerMismatch, got %v", err)
	}
}

func TestMediaService_GetGifThumbnail_Success(t *testing.T) {
	gifRepo := &mockGifRepo{
		getByKeyFunc: func(ctx context.Context, key string) (domainmedia.GifResponse, error) {
			return domainmedia.GifResponse{
				Key:          key,
				ThumbnailUrl: "thumb_" + key + ".jpg",
			}, nil
		},
	}
	storage := &mockStorageRepo{
		getThumbnailURLFunc: func(ctx context.Context, key string) (*url.URL, error) {
			return url.Parse("https://minio.local/thumbnails/" + key)
		},
	}
	svc := newTestMediaService(gifRepo, nil, nil, storage, nil, nil, nil)

	thumbUrl, err := svc.GetGifThumbnail(context.Background(), "sample.gif")
	if err != nil {
		t.Fatalf("expected GetGifThumbnail to succeed, got %v", err)
	}
	if thumbUrl != "https://minio.local/thumbnails/thumb_sample.gif.jpg" {
		t.Errorf("unexpected thumbnail URL: %s", thumbUrl)
	}
}

func TestMediaService_GetGifThumbnail_NotFound(t *testing.T) {
	gifRepo := &mockGifRepo{
		getByKeyFunc: func(ctx context.Context, key string) (domainmedia.GifResponse, error) {
			return domainmedia.GifResponse{}, sql.ErrNoRows
		},
	}
	svc := newTestMediaService(gifRepo, nil, nil, nil, nil, nil, nil)

	_, err := svc.GetGifThumbnail(context.Background(), "missing.gif")
	if !errors.Is(err, domainmedia.ErrGifNotFound) {
		t.Errorf("expected ErrGifNotFound, got %v", err)
	}
}

func TestMediaService_GetGifThumbnail_EmptyThumbnailUrl(t *testing.T) {
	gifRepo := &mockGifRepo{
		getByKeyFunc: func(ctx context.Context, key string) (domainmedia.GifResponse, error) {
			return domainmedia.GifResponse{
				Key:          key,
				ThumbnailUrl: "",
			}, nil
		},
	}
	svc := newTestMediaService(gifRepo, nil, nil, nil, nil, nil, nil)

	_, err := svc.GetGifThumbnail(context.Background(), "no-thumb.gif")
	if !errors.Is(err, domainmedia.ErrThumbnailNotFound) {
		t.Errorf("expected ErrThumbnailNotFound, got %v", err)
	}
}

func TestMediaService_GetGifThumbnail_StorageError(t *testing.T) {
	gifRepo := &mockGifRepo{
		getByKeyFunc: func(ctx context.Context, key string) (domainmedia.GifResponse, error) {
			return domainmedia.GifResponse{
				Key:          key,
				ThumbnailUrl: "thumb.jpg",
			}, nil
		},
	}
	storage := &mockStorageRepo{
		getThumbnailURLFunc: func(ctx context.Context, key string) (*url.URL, error) {
			return nil, errors.New("storage error")
		},
	}
	svc := newTestMediaService(gifRepo, nil, nil, storage, nil, nil, nil)

	_, err := svc.GetGifThumbnail(context.Background(), "error.gif")
	if err == nil {
		t.Fatal("expected error on storage failure, got nil")
	}
}
