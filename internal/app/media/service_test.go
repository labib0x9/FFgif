package media_test

// import (
// 	"context"
// 	"errors"
// 	"net/url"
// 	"testing"
// 	"time"

// 	"github.com/golang-jwt/jwt/v5"
// 	"github.com/google/uuid"
// 	"github.com/labib0x9/ffgif/config"
// 	appmedia "github.com/labib0x9/ffgif/internal/app/media"
// 	domainauth "github.com/labib0x9/ffgif/internal/domain/auth"
// 	domainmedia "github.com/labib0x9/ffgif/internal/domain/media"
// 	domainprocessor "github.com/labib0x9/ffgif/internal/domain/processor"
// 	domainqueue "github.com/labib0x9/ffgif/internal/domain/queue"
// 	domainuser "github.com/labib0x9/ffgif/internal/domain/user"
// 	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
// 	amqp "github.com/rabbitmq/amqp091-go"
// )

// // --- Mocks ---

// type mockStorageRepo struct {
// 	createFunc       func(ctx context.Context, key string, expirey time.Duration) (*url.URL, error)
// 	downloadFunc     func(ctx context.Context, key string, expirey time.Duration) (*url.URL, error)
// 	getStreamURLFunc func(ctx context.Context, key string, expiry time.Duration) (*url.URL, error)
// }

// func (m *mockStorageRepo) Create(ctx context.Context, key string, expirey time.Duration) (*url.URL, error) {
// 	if m.createFunc != nil {
// 		return m.createFunc(ctx, key, expirey)
// 	}
// 	u, _ := url.Parse("https://storage.local/upload/" + key)
// 	return u, nil
// }
// func (m *mockStorageRepo) Download(ctx context.Context, key string, expirey time.Duration) (*url.URL, error) {
// 	if m.downloadFunc != nil {
// 		return m.downloadFunc(ctx, key, expirey)
// 	}
// 	u, _ := url.Parse("https://storage.local/download/" + key)
// 	return u, nil
// }
// func (m *mockStorageRepo) IsExists(ctx context.Context, key string) (bool, error) { return true, nil }
// func (m *mockStorageRepo) Status(ctx context.Context, key string) (domainmedia.Info, error) {
// 	return domainmedia.Info{Size: 1024, ContentType: "video/mp4", UploadedAt: time.Now()}, nil
// }
// func (m *mockStorageRepo) GetObject(ctx context.Context, start, end int64, key string) (domainmedia.Object, error) {
// 	return domainmedia.Object{}, nil
// }
// func (m *mockStorageRepo) DownloadLocal(ctx context.Context, key, destPath string) error { return nil }
// func (m *mockStorageRepo) DownloadLocalRawVideo(ctx context.Context, key, destPath string) error {
// 	return nil
// }
// func (m *mockStorageRepo) Upload(ctx context.Context, key, filePath, contentType string) error {
// 	return nil
// }
// func (m *mockStorageRepo) GetStreamURL(ctx context.Context, key string, expiry time.Duration) (*url.URL, error) {
// 	if m.getStreamURLFunc != nil {
// 		return m.getStreamURLFunc(ctx, key, expiry)
// 	}
// 	u, _ := url.Parse("https://storage.local/stream/" + key)
// 	return u, nil
// }

// type mockGifRepo struct {
// 	getByKeyFunc   func(ctx context.Context, key string) (domainmedia.GifResp, error)
// 	getRecentsFunc func(ctx context.Context, user_id string) ([]domainmedia.GifResp, error)
// 	getFunc        func(ctx context.Context, user_id string, status string) ([]domainmedia.GifResp, error)
// 	deleteFunc     func(ctx context.Context, key string) error
// }

// func (m *mockGifRepo) Create(ctx context.Context, gif domainmedia.Gif) error { return nil }
// func (m *mockGifRepo) Get(ctx context.Context, user_id string, status string) ([]domainmedia.GifResp, error) {
// 	if m.getFunc != nil {
// 		return m.getFunc(ctx, user_id, status)
// 	}
// 	return []domainmedia.GifResp{}, nil
// }
// func (m *mockGifRepo) GetByKey(ctx context.Context, key string) (domainmedia.GifResp, error) {
// 	if m.getByKeyFunc != nil {
// 		return m.getByKeyFunc(ctx, key)
// 	}
// 	return domainmedia.GifResp{Key: key, Url: "https://minio/gif/" + key}, nil
// }
// func (m *mockGifRepo) GetRecents(ctx context.Context, user_id string) ([]domainmedia.GifResp, error) {
// 	if m.getRecentsFunc != nil {
// 		return m.getRecentsFunc(ctx, user_id)
// 	}
// 	return []domainmedia.GifResp{}, nil
// }
// func (m *mockGifRepo) Delete(ctx context.Context, key string) error {
// 	if m.deleteFunc != nil {
// 		return m.deleteFunc(ctx, key)
// 	}
// 	return nil
// }
// func (m *mockGifRepo) Update(ctx context.Context, key string, gif domainmedia.GifResp) error {
// 	return nil
// }
// func (m *mockGifRepo) SaveRecent(ctx context.Context, key string) error { return nil }

// type mockLastVideoRepo struct {
// 	createFunc       func(ctx context.Context, upload domainmedia.LastUpload) error
// 	getLastVideoFunc func(ctx context.Context, user_id string) (domainmedia.LastUploadResp, error)
// }

// func (m *mockLastVideoRepo) Create(ctx context.Context, upload domainmedia.LastUpload) error {
// 	if m.createFunc != nil {
// 		return m.createFunc(ctx, upload)
// 	}
// 	return nil
// }
// func (m *mockLastVideoRepo) GetLastVideo(ctx context.Context, user_id string) (domainmedia.LastUploadResp, error) {
// 	if m.getLastVideoFunc != nil {
// 		return m.getLastVideoFunc(ctx, user_id)
// 	}
// 	return domainmedia.LastUploadResp{UserID: uuid.MustParse(user_id), FileKey: "last-video.mp4"}, nil
// }

// type mockProcessor struct {
// 	preProcessFunc func(ctx context.Context, key string) (*domainprocessor.PrePrecessedResult, error)
// }

// func (m *mockProcessor) Process(ctx context.Context, JobId string, Key string, Start float32, End float32, Width int, FPS int, Loop bool) (*domainprocessor.JobResult, error) {
// 	return nil, nil
// }
// func (m *mockProcessor) PreProcess(ctx context.Context, key string) (*domainprocessor.PrePrecessedResult, error) {
// 	if m.preProcessFunc != nil {
// 		return m.preProcessFunc(ctx, key)
// 	}
// 	return &domainprocessor.PrePrecessedResult{
// 		VideoKey:     "converted_video.mp4",
// 		ThumbnailKey: "thumb.jpg",
// 		ContentType:  "video/mp4",
// 		Size:         "2048",
// 		Duration:     "10.0",
// 	}, nil
// }

// type mockCache struct {
// 	store map[string]string
// }

// func (m *mockCache) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
// 	m.store[key] = value
// 	return nil
// }
// func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
// 	if v, ok := m.store[key]; ok {
// 		return v, nil
// 	}
// 	return "", errors.New("not found in cache")
// }

// type mockQueue struct{}

// func (m *mockQueue) PublishEmail(ctx context.Context, msg domainqueue.EmailMessage) error      { return nil }
// func (m *mockQueue) PublishVideo(ctx context.Context, msg domainqueue.VideoMessage) error      { return nil }
// func (m *mockQueue) PublishSaveVideo(ctx context.Context, msg domainqueue.SaveVideoMessage) error { return nil }
// func (m *mockQueue) PublishRetrySaveVideo(ctx context.Context, msg domainqueue.SaveVideoMessage) error {
// 	return nil
// }
// func (m *mockQueue) ConsumeEmail(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error) {
// 	return nil, nil
// }
// func (m *mockQueue) ConsumeSave(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error) {
// 	return nil, nil
// }
// func (m *mockQueue) ConsumeVideo(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error) {
// 	return nil, nil
// }
// func (m *mockQueue) ConsumeRawVideo(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error) {
// 	return nil, nil
// }
// func (m *mockQueue) Close() error                           { return nil }
// func (m *mockQueue) CloseConsumerChannel(name string) error { return nil }

// type mockAuthRepo struct{}

// func (m *mockAuthRepo) GetByEmail(ctx context.Context, email string) (domainauth.User, error) {
// 	return domainauth.User{}, nil
// }
// func (m *mockAuthRepo) GetById(ctx context.Context, id uuid.UUID) (domainauth.User, error) {
// 	return domainauth.User{}, nil
// }
// func (m *mockAuthRepo) Create(ctx context.Context, user domainauth.User) (domainauth.User, error) {
// 	return user, nil
// }
// func (m *mockAuthRepo) DeleteById(ctx context.Context, id uuid.UUID) error { return nil }
// func (m *mockAuthRepo) DeleteByEmail(ctx context.Context, email string) error { return nil }
// func (m *mockAuthRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passHash string) error {
// 	return nil
// }
// func (m *mockAuthRepo) SetVerified(ctx context.Context, userId uuid.UUID) error { return nil }
// func (m *mockAuthRepo) Upgrade(ctx context.Context, id string, user domainauth.User) (domainauth.User, error) {
// 	return user, nil
// }

// type mockUserRepo struct{}

// func (m *mockUserRepo) GetProfile(ctx context.Context, id string) (domainuser.ProfileResp, error) {
// 	return domainuser.ProfileResp{}, nil
// }
// func (m *mockUserRepo) UpdateProfile(ctx context.Context, profile domainuser.ProfileResp, id string) (domainuser.ProfileResp, error) {
// 	return profile, nil
// }
// func (m *mockUserRepo) SetProfile(ctx context.Context, profile domainuser.Profile) error { return nil }
// func (m *mockUserRepo) ChangePassword(ctx context.Context, userId string, hash string) error {
// 	return nil
// }

// type mockQuotaRepo struct{}

// func (m *mockQuotaRepo) Create(ctx context.Context, quota domainuser.Quota) error { return nil }
// func (m *mockQuotaRepo) GetById(ctx context.Context, id string) (*domainuser.Quota, error) {
// 	return &domainuser.Quota{}, nil
// }

// // --- Tests ---

// func TestMediaService_Upload_Success(t *testing.T) {
// 	cache := &mockCache{store: make(map[string]string)}
// 	storage := &mockStorageRepo{}

// 	svc := appmedia.NewService(
// 		&mockAuthRepo{},
// 		&mockUserRepo{},
// 		&mockQuotaRepo{},
// 		&mockGifRepo{},
// 		&mockLastVideoRepo{},
// 		storage,
// 		&mockQueue{},
// 		cache,
// 		&mockProcessor{},
// 		&config.Config{},
// 	)

// 	claims := jwtpkg.Payload{
// 		RegisteredClaims: jwt.RegisteredClaims{
// 			Subject: "user-123",
// 		},
// 	}

// 	result, err := svc.Upload(context.Background(), "sample.mp4", claims)
// 	if err != nil {
// 		t.Fatalf("expected Upload to succeed, got error: %v", err)
// 	}

// 	if result.Url == "" {
// 		t.Errorf("expected presigned upload URL, got empty")
// 	}
// 	if result.Key == "" {
// 		t.Errorf("expected upload key, got empty")
// 	}
// }

// func TestMediaService_ProcessAndSave_Success(t *testing.T) {
// 	userId := uuid.New().String()
// 	uploadKey := userId + ":video-uuid.mp4"
// 	saved := false

// 	lastVideoRepo := &mockLastVideoRepo{
// 		createFunc: func(ctx context.Context, upload domainmedia.LastUpload) error {
// 			saved = true
// 			if upload.UserID.String() != userId {
// 				t.Errorf("expected userID %s, got %s", userId, upload.UserID)
// 			}
// 			return nil
// 		},
// 	}

// 	svc := appmedia.NewService(
// 		&mockAuthRepo{},
// 		&mockUserRepo{},
// 		&mockQuotaRepo{},
// 		&mockGifRepo{},
// 		lastVideoRepo,
// 		&mockStorageRepo{},
// 		&mockQueue{},
// 		&mockCache{store: make(map[string]string)},
// 		&mockProcessor{},
// 		&config.Config{},
// 	)

// 	err := svc.ProcessAndSave(context.Background(), uploadKey)
// 	if err != nil {
// 		t.Fatalf("expected ProcessAndSave to succeed, got: %v", err)
// 	}

// 	if !saved {
// 		t.Errorf("expected LastUpload to be created")
// 	}
// }

// func TestMediaService_Download_Success(t *testing.T) {
// 	gifRepo := &mockGifRepo{
// 		getByKeyFunc: func(ctx context.Context, key string) (domainmedia.GifResp, error) {
// 			return domainmedia.GifResp{Key: key}, nil
// 		},
// 	}
// 	storage := &mockStorageRepo{
// 		downloadFunc: func(ctx context.Context, key string, expirey time.Duration) (*url.URL, error) {
// 			u, _ := url.Parse("https://minio.download/" + key)
// 			return u, nil
// 		},
// 	}

// 	svc := appmedia.NewService(
// 		&mockAuthRepo{},
// 		&mockUserRepo{},
// 		&mockQuotaRepo{},
// 		gifRepo,
// 		&mockLastVideoRepo{},
// 		storage,
// 		&mockQueue{},
// 		&mockCache{store: make(map[string]string)},
// 		&mockProcessor{},
// 		&config.Config{},
// 	)

// 	downloadUrl, err := svc.Download(context.Background(), "my-gif.gif")
// 	if err != nil {
// 		t.Fatalf("expected Download to succeed, got: %v", err)
// 	}

// 	if downloadUrl != "https://minio.download/my-gif.gif" {
// 		t.Errorf("expected https://minio.download/my-gif.gif, got %s", downloadUrl)
// 	}
// }

// func TestMediaService_Download_NotFound(t *testing.T) {
// 	gifRepo := &mockGifRepo{
// 		getByKeyFunc: func(ctx context.Context, key string) (domainmedia.GifResp, error) {
// 			return domainmedia.GifResp{}, errors.New("sql: no rows in result set")
// 		},
// 	}

// 	svc := appmedia.NewService(
// 		&mockAuthRepo{},
// 		&mockUserRepo{},
// 		&mockQuotaRepo{},
// 		gifRepo,
// 		&mockLastVideoRepo{},
// 		&mockStorageRepo{},
// 		&mockQueue{},
// 		&mockCache{store: make(map[string]string)},
// 		&mockProcessor{},
// 		&config.Config{},
// 	)

// 	_, err := svc.Download(context.Background(), "non-existent.gif")
// 	if !errors.Is(err, domainmedia.ErrGifNotFound) {
// 		t.Errorf("expected ErrGifNotFound, got %v", err)
// 	}
// }

// package job_test

// import (
// 	"context"
// 	"errors"
// 	"net/url"
// 	"testing"
// 	"time"

// 	"github.com/google/uuid"
// 	appjob "github.com/labib0x9/ffgif/internal/app/job"
// 	domainjob "github.com/labib0x9/ffgif/internal/domain/job"
// 	domainmedia "github.com/labib0x9/ffgif/internal/domain/media"
// 	domainprocessor "github.com/labib0x9/ffgif/internal/domain/processor"
// 	domainqueue "github.com/labib0x9/ffgif/internal/domain/queue"
// 	amqp "github.com/rabbitmq/amqp091-go"
// )

// // --- Mocks ---

// type mockProcessor struct {
// 	processFunc func(ctx context.Context, JobId string, Key string, Start float32, End float32, Width int, FPS int, Loop bool) (*domainprocessor.JobResult, error)
// }

// func (m *mockProcessor) Process(ctx context.Context, JobId string, Key string, Start float32, End float32, Width int, FPS int, Loop bool) (*domainprocessor.JobResult, error) {
// 	if m.processFunc != nil {
// 		return m.processFunc(ctx, JobId, Key, Start, End, Width, FPS, Loop)
// 	}
// 	return &domainprocessor.JobResult{GifKey: "gif_123.gif", ThumbKey: "thumb_123.jpg"}, nil
// }
// func (m *mockProcessor) PreProcess(ctx context.Context, key string) (*domainprocessor.PrePrecessedResult, error) {
// 	return nil, nil
// }

// type mockGifRepo struct {
// 	createFunc func(ctx context.Context, gif domainmedia.Gif) error
// }

// func (m *mockGifRepo) Create(ctx context.Context, gif domainmedia.Gif) error {
// 	if m.createFunc != nil {
// 		return m.createFunc(ctx, gif)
// 	}
// 	return nil
// }
// func (m *mockGifRepo) Get(ctx context.Context, user_id string, status string) ([]domainmedia.GifResp, error) {
// 	return nil, nil
// }
// func (m *mockGifRepo) GetByKey(ctx context.Context, key string) (domainmedia.GifResp, error) {
// 	return domainmedia.GifResp{}, nil
// }
// func (m *mockGifRepo) GetRecents(ctx context.Context, user_id string) ([]domainmedia.GifResp, error) {
// 	return nil, nil
// }
// func (m *mockGifRepo) Delete(ctx context.Context, key string) error { return nil }
// func (m *mockGifRepo) Update(ctx context.Context, key string, gif domainmedia.GifResp) error {
// 	return nil
// }
// func (m *mockGifRepo) SaveRecent(ctx context.Context, key string) error { return nil }

// type mockLastVideoRepo struct {
// 	createFunc func(ctx context.Context, upload domainmedia.LastUpload) error
// }

// func (m *mockLastVideoRepo) Create(ctx context.Context, upload domainmedia.LastUpload) error {
// 	if m.createFunc != nil {
// 		return m.createFunc(ctx, upload)
// 	}
// 	return nil
// }
// func (m *mockLastVideoRepo) GetLastVideo(ctx context.Context, user_id string) (domainmedia.LastUploadResp, error) {
// 	return domainmedia.LastUploadResp{}, nil
// }

// type mockStorageRepo struct {
// 	statusFunc func(ctx context.Context, key string) (domainmedia.Info, error)
// }

// func (m *mockStorageRepo) Status(ctx context.Context, key string) (domainmedia.Info, error) {
// 	if m.statusFunc != nil {
// 		return m.statusFunc(ctx, key)
// 	}
// 	size := int64(1024)
// 	return domainmedia.Info{Size: size, ContentType: "video/mp4", UploadedAt: time.Now()}, nil
// }
// func (m *mockStorageRepo) Create(ctx context.Context, key string, expire time.Duration) (*url.URL, error) {
// 	return nil, nil
// }
// func (m *mockStorageRepo) Upload(ctx context.Context, key string, path string, contentType string) error {
// 	return nil
// }
// func (m *mockStorageRepo) DownloadLocal(ctx context.Context, key string, path string) error {
// 	return nil
// }
// func (m *mockStorageRepo) DownloadLocalRawVideo(ctx context.Context, key string, path string) error {
// 	return nil
// }
// func (m *mockStorageRepo) Download(ctx context.Context, key string, expire time.Duration) (*url.URL, error) {
// 	u, _ := url.Parse("https://storage/download/" + key)
// 	return u, nil
// }
// func (m *mockStorageRepo) Delete(ctx context.Context, key string) error { return nil }
// func (m *mockStorageRepo) IsExists(ctx context.Context, key string) (bool, error) {
// 	return true, nil
// }
// func (m *mockStorageRepo) GetObject(ctx context.Context, start, end int64, key string) (domainmedia.Object, error) {
// 	return domainmedia.Object{}, nil
// }
// func (m *mockStorageRepo) GetStreamURL(ctx context.Context, key string, expiry time.Duration) (*url.URL, error) {
// 	return nil, nil
// }

// type mockCache struct {
// 	store map[string]string
// }

// func newMockCache() *mockCache {
// 	return &mockCache{store: make(map[string]string)}
// }
// func (m *mockCache) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
// 	m.store[key] = value
// 	return nil
// }
// func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
// 	if v, ok := m.store[key]; ok {
// 		return v, nil
// 	}
// 	return "", errors.New("key not found in cache")
// }

// type mockQueue struct {
// 	publishVideoFunc func(ctx context.Context, msg domainqueue.VideoMessage) error
// }

// func (m *mockQueue) PublishVideo(ctx context.Context, msg domainqueue.VideoMessage) error {
// 	if m.publishVideoFunc != nil {
// 		return m.publishVideoFunc(ctx, msg)
// 	}
// 	return nil
// }
// func (m *mockQueue) PublishEmail(ctx context.Context, msg domainqueue.EmailMessage) error      { return nil }
// func (m *mockQueue) PublishSaveVideo(ctx context.Context, msg domainqueue.SaveVideoMessage) error { return nil }
// func (m *mockQueue) PublishRetrySaveVideo(ctx context.Context, msg domainqueue.SaveVideoMessage) error {
// 	return nil
// }
// func (m *mockQueue) ConsumeEmail(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error) {
// 	return nil, nil
// }
// func (m *mockQueue) ConsumeSave(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error) {
// 	return nil, nil
// }
// func (m *mockQueue) ConsumeVideo(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error) {
// 	return nil, nil
// }
// func (m *mockQueue) ConsumeRawVideo(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error) {
// 	return nil, nil
// }
// func (m *mockQueue) Close() error                           { return nil }
// func (m *mockQueue) CloseConsumerChannel(name string) error { return nil }

// // --- Tests ---

// func TestJobService_Convert_Success(t *testing.T) {
// 	cache := newMockCache()
// 	videoPublished := false
// 	queue := &mockQueue{
// 		publishVideoFunc: func(ctx context.Context, msg domainqueue.VideoMessage) error {
// 			videoPublished = true
// 			if msg.Key != "test_key.mp4" {
// 				t.Errorf("expected key test_key.mp4, got %s", msg.Key)
// 			}
// 			return nil
// 		},
// 	}

// 	svc := appjob.NewService(
// 		&mockProcessor{},
// 		&mockGifRepo{},
// 		&mockLastVideoRepo{},
// 		&mockStorageRepo{},
// 		cache,
// 		queue,
// 	)

// 	res, err := svc.Convert(context.Background(), "user-123", "test_key.mp4", 0.0, 5.0, 15, 480, true)
// 	if err != nil {
// 		t.Fatalf("expected Convert to succeed, got error: %v", err)
// 	}

// 	if res.Id == "" {
// 		t.Errorf("expected non-empty JobId")
// 	}
// 	if res.Status != "queued" {
// 		t.Errorf("expected status 'queued', got %s", res.Status)
// 	}
// 	if !videoPublished {
// 		t.Errorf("expected video message to be published to queue")
// 	}
// }

// func TestJobService_Process_Success(t *testing.T) {
// 	cache := newMockCache()
// 	createdGif := false
// 	gifRepo := &mockGifRepo{
// 		createFunc: func(ctx context.Context, gif domainmedia.Gif) error {
// 			createdGif = true
// 			if gif.Key != "output.gif" {
// 				t.Errorf("expected gif key output.gif, got %s", gif.Key)
// 			}
// 			return nil
// 		},
// 	}
// 	proc := &mockProcessor{
// 		processFunc: func(ctx context.Context, JobId string, Key string, Start float32, End float32, Width int, FPS int, Loop bool) (*domainprocessor.JobResult, error) {
// 			return &domainprocessor.JobResult{GifKey: "output.gif", ThumbKey: "thumb.jpg"}, nil
// 		},
// 	}

// 	svc := appjob.NewService(proc, gifRepo, &mockLastVideoRepo{}, &mockStorageRepo{}, cache, &mockQueue{})

// 	msg := domainqueue.VideoMessage{
// 		UserID: "user-123",
// 		JobId:  "job-456",
// 		Key:    "raw.mp4",
// 		Start:  0,
// 		End:    3,
// 		FPS:    10,
// 		Width:  320,
// 		Loop:   true,
// 	}

// 	err := svc.Process(context.Background(), msg)
// 	if err != nil {
// 		t.Fatalf("expected Process to succeed, got: %v", err)
// 	}

// 	if !createdGif {
// 		t.Errorf("expected gif record to be created in repository")
// 	}

// 	status, _ := cache.Get(context.Background(), "messaage_queue:job_id:job-456")
// 	if status != "completed" {
// 		t.Errorf("expected status completed, got %s", status)
// 	}
// }

// func TestJobService_Status_Success(t *testing.T) {
// 	cache := newMockCache()
// 	cache.Set(context.Background(), "messaage_queue:job_id:job-789", "completed", time.Minute)
// 	cache.Set(context.Background(), "messaage_queue_gif:job_id:job-789", "gif-result-key", time.Minute)

// 	svc := appjob.NewService(&mockProcessor{}, &mockGifRepo{}, &mockLastVideoRepo{}, &mockStorageRepo{}, cache, &mockQueue{})

// 	result, err := svc.Status(context.Background(), "job-789")
// 	if err != nil {
// 		t.Fatalf("expected Status to succeed, got: %v", err)
// 	}

// 	if result.Status != "completed" {
// 		t.Errorf("expected status 'completed', got %s", result.Status)
// 	}
// 	if result.GifId != "gif-result-key" {
// 		t.Errorf("expected gifId 'gif-result-key', got %s", result.GifId)
// 	}
// }

// func TestJobService_SaveMetadata_InvalidUserID(t *testing.T) {
// 	svc := appjob.NewService(&mockProcessor{}, &mockGifRepo{}, &mockLastVideoRepo{}, &mockStorageRepo{}, newMockCache(), &mockQueue{})

// 	msg := domainqueue.SaveVideoMessage{
// 		UserID:   "not-a-valid-uuid",
// 		Key:      "key-123",
// 		Filename: "test.mp4",
// 	}

// 	err := svc.SaveMetadata(context.Background(), msg)
// 	if !errors.Is(err, domainjob.ErrInvalidUserID) {
// 		t.Errorf("expected ErrInvalidUserID, got %v", err)
// 	}
// }

// func TestJobService_SaveMetadata_Success(t *testing.T) {
// 	saved := false
// 	validUUID := uuid.New().String()

// 	lastVideoRepo := &mockLastVideoRepo{
// 		createFunc: func(ctx context.Context, upload domainmedia.LastUpload) error {
// 			saved = true
// 			if upload.FileKey != "valid-key" {
// 				t.Errorf("expected file key 'valid-key', got %s", upload.FileKey)
// 			}
// 			return nil
// 		},
// 	}

// 	svc := appjob.NewService(&mockProcessor{}, &mockGifRepo{}, lastVideoRepo, &mockStorageRepo{}, newMockCache(), &mockQueue{})

// 	msg := domainqueue.SaveVideoMessage{
// 		UserID:   validUUID,
// 		Key:      "valid-key",
// 		Filename: "video.mp4",
// 	}

// 	err := svc.SaveMetadata(context.Background(), msg)
// 	if err != nil {
// 		t.Fatalf("expected SaveMetadata to succeed, got: %v", err)
// 	}

// 	if !saved {
// 		t.Errorf("expected last video metadata to be saved")
// 	}
// }
