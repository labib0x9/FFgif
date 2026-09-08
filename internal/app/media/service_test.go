package media_test

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/labib0x9/ffgif/config"
	appmedia "github.com/labib0x9/ffgif/internal/app/media"
	authMocks "github.com/labib0x9/ffgif/internal/domain/auth/mocks"
	domainmedia "github.com/labib0x9/ffgif/internal/domain/media"
	mediaMocks "github.com/labib0x9/ffgif/internal/domain/media/mocks"
	shareMocks "github.com/labib0x9/ffgif/internal/domain/share/mocks"
	domainuser "github.com/labib0x9/ffgif/internal/domain/user"
	userMocks "github.com/labib0x9/ffgif/internal/domain/user/mocks"
	cacheMocks "github.com/labib0x9/ffgif/internal/port/cache/mocks"
	dbMocks "github.com/labib0x9/ffgif/internal/port/db/mocks"
	processorMocks "github.com/labib0x9/ffgif/internal/port/processor/mocks"
	"github.com/labib0x9/ffgif/internal/port/queue"
	queueMocks "github.com/labib0x9/ffgif/internal/port/queue/mocks"
)

type mediaTestDeps struct {
	ctrl          *gomock.Controller
	authRepo      *authMocks.MockAuthRepository
	profileRepo   *userMocks.MockUserRepository
	quotaRepo     *userMocks.MockQuotaRepository
	gifRepo       *mediaMocks.MockGifRepository
	shareRepo     *shareMocks.MockShareRepository
	lastVideoRepo *mediaMocks.MockLastVideoRepository
	storage       *mediaMocks.MockStorageRepository
	txManager     *dbMocks.MockTxManager
	queue         *queueMocks.MockQueue
	cache         *cacheMocks.MockCache
	processor     *processorMocks.MockVideoProcessor
	cnf           *config.Config
	svc           appmedia.Service
}

func newMediaTestDeps(t *testing.T) *mediaTestDeps {
	ctrl := gomock.NewController(t)
	cnf := &config.Config{
		Minio: &config.Minio{
			StorageBucket: "ffgif-media",
			TempBucket:    "ffgif-temp",
		},
	}

	deps := &mediaTestDeps{
		ctrl:          ctrl,
		authRepo:      authMocks.NewMockAuthRepository(ctrl),
		profileRepo:   userMocks.NewMockUserRepository(ctrl),
		quotaRepo:     userMocks.NewMockQuotaRepository(ctrl),
		gifRepo:       mediaMocks.NewMockGifRepository(ctrl),
		shareRepo:     shareMocks.NewMockShareRepository(ctrl),
		lastVideoRepo: mediaMocks.NewMockLastVideoRepository(ctrl),
		storage:       mediaMocks.NewMockStorageRepository(ctrl),
		txManager:     dbMocks.NewMockTxManager(ctrl),
		queue:         queueMocks.NewMockQueue(ctrl),
		cache:         cacheMocks.NewMockCache(ctrl),
		processor:     processorMocks.NewMockVideoProcessor(ctrl),
		cnf:           cnf,
	}

	deps.svc = appmedia.NewService(
		deps.authRepo,
		deps.profileRepo,
		deps.quotaRepo,
		deps.gifRepo,
		deps.shareRepo,
		deps.lastVideoRepo,
		deps.storage,
		deps.txManager,
		deps.queue,
		deps.cache,
		deps.processor,
		deps.cnf,
	)
	return deps
}

func TestMediaService_Upload(t *testing.T) {
	t.Run("success: generates presigned upload url and updates cache status", func(t *testing.T) {
		d := newMediaTestDeps(t)
		ctx := context.Background()

		userId := "user-123"
		uploadURL, _ := url.Parse("https://minio.local/upload/presigned-url")

		d.storage.EXPECT().Create(gomock.Any(), gomock.Cond(func(x any) bool {
			key, ok := x.(string)
			return ok && len(key) > len(userId) && key[:len(userId)] == userId
		}), gomock.Eq(5*time.Minute)).Return(uploadURL, nil).Times(1)

		d.cache.EXPECT().Set(gomock.Any(), gomock.Cond(func(x any) bool {
			k, ok := x.(string)
			return ok && len(k) > 10 && k[:10] == "uploading:"
		}), gomock.Eq("uploading"), gomock.Eq(5*time.Minute)).Return(nil).Times(1)

		res, err := d.svc.Upload(ctx, "sample.mp4", userId)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Url != uploadURL.String() || res.Key == "" {
			t.Errorf("unexpected upload result: %+v", res)
		}
	})

	// EXPECTED TO FAIL: Quota enforcement is not implemented in Upload (internal/app/media/upload.go).
	// When user quota is exceeded, Upload should check QuotaRepository and reject the request.
	t.Run("quota exhausted: should reject upload with quota exceeded error", func(t *testing.T) {
		d := newMediaTestDeps(t)
		ctx := context.Background()

		userId := "user-at-limit"
		d.quotaRepo.EXPECT().GetById(gomock.Any(), gomock.Eq(userId)).Return(&domainuser.Quota{
			UsedBytes:  50 * 1024 * 1024,
			TotalBytes: 50 * 1024 * 1024, // 100% full
			GifCount:   10,
			GitCount:   10,
		}, nil).AnyTimes()

		d.storage.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(&url.URL{}, nil).AnyTimes()
		d.cache.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		_, err := d.svc.Upload(ctx, "large.mp4", userId)
		if err == nil {
			t.Errorf("BUG/GAP: Upload allowed when user quota was 100%% exhausted")
		}
	})
}

func TestMediaService_Convert(t *testing.T) {
	t.Run("success: publishes conversion job to queue and sets cache status", func(t *testing.T) {
		d := newMediaTestDeps(t)
		ctx := context.Background()

		userId := "user-123"
		key := "user-123:video.mp4"

		d.cache.EXPECT().Set(gomock.Any(), gomock.Cond(func(x any) bool {
			k, ok := x.(string)
			return ok && strings.HasPrefix(k, "messaage_queue:job_id:")
		}), gomock.Eq("queued"), gomock.Eq(5*time.Minute)).Return(nil).Times(1)

		d.cache.EXPECT().Set(gomock.Any(), gomock.Cond(func(x any) bool {
			k, ok := x.(string)
			return ok && strings.HasPrefix(k, "messaage_queue_gif:job_id:")
		}), gomock.Eq("queued"), gomock.Eq(5*time.Minute)).Return(nil).Times(1)

		d.queue.EXPECT().PublishVideo(gomock.Any(), gomock.Cond(func(x any) bool {
			m, ok := x.(queue.VideoMessage)
			return ok && m.UserID == userId && m.Key == key && m.Start == 0.0 && m.End == 5.0 && m.FPS == 15 && m.Width == 480 && m.Loop == true
		})).Return(nil).Times(1)

		res, err := d.svc.Convert(ctx, userId, key, 0.0, 5.0, 15, 480, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Id == "" || res.Status != "queued" {
			t.Errorf("unexpected convert result: %+v", res)
		}
	})

	// EXPECTED TO FAIL: Quota enforcement is not implemented in Convert (internal/app/media/convert.go).
	t.Run("quota exhausted: should reject conversion with quota exceeded error", func(t *testing.T) {
		d := newMediaTestDeps(t)
		ctx := context.Background()

		userId := "user-maxed-gifs"
		d.quotaRepo.EXPECT().GetById(gomock.Any(), gomock.Eq(userId)).Return(&domainuser.Quota{
			GifCount: 10,
			GitCount: 10, // gif count limit reached
		}, nil).AnyTimes()

		d.cache.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		d.queue.EXPECT().PublishVideo(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		_, err := d.svc.Convert(ctx, userId, "video.mp4", 0.0, 5.0, 15, 480, true)
		if err == nil {
			t.Errorf("BUG/GAP: Convert allowed when GIF count quota limit was reached")
		}
	})
}

func TestMediaService_Download(t *testing.T) {
	t.Run("success: owner downloads gif", func(t *testing.T) {
		d := newMediaTestDeps(t)
		ctx := context.Background()

		userId := "owner-123"
		gifKey := "my-gif.gif"
		downloadURL, _ := url.Parse("https://minio.local/download/my-gif.gif")

		d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq(gifKey)).Return(userId, nil).Times(1)
		d.shareRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq(userId), gomock.Eq(gifKey)).Return("", sql.ErrNoRows).Times(1)
		d.storage.EXPECT().Download(gomock.Any(), gomock.Eq(gifKey), gomock.Eq(5*time.Minute)).Return(downloadURL, nil).Times(1)

		res, err := d.svc.Download(ctx, userId, gifKey)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != downloadURL.String() {
			t.Errorf("expected url %s, got %s", downloadURL.String(), res)
		}
	})

	t.Run("success: shared recipient downloads gif", func(t *testing.T) {
		d := newMediaTestDeps(t)
		ctx := context.Background()

		ownerId := "owner-123"
		recipientId := "recipient-456"
		gifKey := "shared-gif.gif"
		downloadURL, _ := url.Parse("https://minio.local/download/shared-gif.gif")

		d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq(gifKey)).Return(ownerId, nil).Times(1)
		d.shareRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq(recipientId), gomock.Eq(gifKey)).Return(ownerId, nil).Times(1)
		d.storage.EXPECT().Download(gomock.Any(), gomock.Eq(gifKey), gomock.Eq(5*time.Minute)).Return(downloadURL, nil).Times(1)

		res, err := d.svc.Download(ctx, recipientId, gifKey)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != downloadURL.String() {
			t.Errorf("expected url %s, got %s", downloadURL.String(), res)
		}
	})

	t.Run("unauthorized: neither owner nor shared recipient", func(t *testing.T) {
		d := newMediaTestDeps(t)
		ctx := context.Background()

		ownerId := "owner-123"
		strangerId := "stranger-999"
		gifKey := "private-gif.gif"

		d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq(gifKey)).Return(ownerId, nil).Times(1)
		d.shareRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq(strangerId), gomock.Eq(gifKey)).Return("", sql.ErrNoRows).Times(1)

		_, err := d.svc.Download(ctx, strangerId, gifKey)
		if !errors.Is(err, domainmedia.ErrGifOwnerMismatch) {
			t.Fatalf("expected ErrGifOwnerMismatch, got %v", err)
		}
	})

	t.Run("gif not found: returns ErrGifNotFound", func(t *testing.T) {
		d := newMediaTestDeps(t)
		ctx := context.Background()

		d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq("nonexistent.gif")).Return("", sql.ErrNoRows).Times(1)

		_, err := d.svc.Download(ctx, "user-123", "nonexistent.gif")
		if !errors.Is(err, domainmedia.ErrGifNotFound) {
			t.Fatalf("expected ErrGifNotFound, got %v", err)
		}
	})
}

func TestMediaService_GetGifThumbnail(t *testing.T) {
	t.Run("success: returns presigned thumbnail url", func(t *testing.T) {
		d := newMediaTestDeps(t)
		ctx := context.Background()

		userId := "user-123"
		gifKey := "my-gif.gif"
		thumbKey := "thumbnails/thumb_my-gif.jpg"
		thumbURL, _ := url.Parse("https://minio.local/" + thumbKey)

		d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq(gifKey)).Return(userId, nil).Times(1)
		d.gifRepo.EXPECT().GetByKey(gomock.Any(), gomock.Eq(gifKey), gomock.Eq(false)).Return(domainmedia.GifResponse{
			Key:          gifKey,
			ThumbnailUrl: thumbKey,
		}, nil).Times(1)
		d.storage.EXPECT().GetThumbnailURL(gomock.Any(), gomock.Eq(thumbKey)).Return(thumbURL, nil).Times(1)

		res, err := d.svc.GetGifThumbnail(ctx, userId, gifKey)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != thumbURL.String() {
			t.Errorf("expected thumb url %s, got %s", thumbURL.String(), res)
		}
	})

	t.Run("missing thumbnail key on existing gif: returns ErrThumbnailNotFound", func(t *testing.T) {
		d := newMediaTestDeps(t)
		ctx := context.Background()

		userId := "user-123"
		gifKey := "no-thumb.gif"

		d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq(gifKey)).Return(userId, nil).Times(1)
		d.gifRepo.EXPECT().GetByKey(gomock.Any(), gomock.Eq(gifKey), gomock.Eq(false)).Return(domainmedia.GifResponse{
			Key:          gifKey,
			ThumbnailUrl: "", // empty thumbnail
		}, nil).Times(1)

		_, err := d.svc.GetGifThumbnail(ctx, userId, gifKey)
		if !errors.Is(err, domainmedia.ErrThumbnailNotFound) {
			t.Fatalf("expected ErrThumbnailNotFound, got %v", err)
		}
	})
}

func TestMediaService_Status(t *testing.T) {
	t.Run("success: status ok returns file key", func(t *testing.T) {
		d := newMediaTestDeps(t)
		ctx := context.Background()

		userId := "user-123"
		key := "video-123"

		d.cache.EXPECT().Get(gomock.Any(), gomock.Eq("uploading:"+key)).Return("ok", nil).Times(1)
		d.lastVideoRepo.EXPECT().GetLastVideo(gomock.Any(), gomock.Eq(userId)).Return(domainmedia.LastUploadResponse{
			FileKey: "user-123:video-123.mp4",
		}, nil).Times(1)

		fileKey, status, err := d.svc.Status(ctx, userId, key)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if status != "ok" || fileKey != "user-123:video-123.mp4" {
			t.Errorf("unexpected status result: fileKey=%s, status=%s", fileKey, status)
		}
	})

	t.Run("cache backend error: returns error", func(t *testing.T) {
		d := newMediaTestDeps(t)
		ctx := context.Background()

		d.cache.EXPECT().Get(gomock.Any(), gomock.Eq("uploading:err-key")).
			Return("", errors.New("redis connection refused")).Times(1)

		_, _, err := d.svc.Status(ctx, "user-123", "err-key")
		if err == nil {
			t.Fatalf("expected error when cache.Get fails, got nil")
		}
	})
}
