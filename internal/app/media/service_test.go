package media_test

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/labib0x9/ffgif/config"
	appmedia "github.com/labib0x9/ffgif/internal/app/media"
	authmocks "github.com/labib0x9/ffgif/internal/domain/auth/mocks"
	domainmedia "github.com/labib0x9/ffgif/internal/domain/media"
	mediamocks "github.com/labib0x9/ffgif/internal/domain/media/mocks"
	sharemocks "github.com/labib0x9/ffgif/internal/domain/share/mocks"
	domainuser "github.com/labib0x9/ffgif/internal/domain/user"
	usermocks "github.com/labib0x9/ffgif/internal/domain/user/mocks"
	portcache "github.com/labib0x9/ffgif/internal/port/cache"
	cachemocks "github.com/labib0x9/ffgif/internal/port/cache/mocks"
	dbmocks "github.com/labib0x9/ffgif/internal/port/db/mocks"
	processormocks "github.com/labib0x9/ffgif/internal/port/processor/mocks"
	"github.com/labib0x9/ffgif/internal/port/queue"
	queuemocks "github.com/labib0x9/ffgif/internal/port/queue/mocks"
)

type mediaDeps struct {
	authRepo      *authmocks.MockAuthRepository
	profileRepo   *usermocks.MockUserRepository
	quotaRepo     *usermocks.MockQuotaRepository
	gifRepo       *mediamocks.MockGifRepository
	shareRepo     *sharemocks.MockShareRepository
	lastVideoRepo *mediamocks.MockLastVideoRepository
	storage       *mediamocks.MockStorageRepository
	tnx           *dbmocks.MockTxManager
	queue         *queuemocks.MockQueue
	cache         *cachemocks.MockCache
	processor     *processormocks.MockVideoProcessor
	svc           appmedia.Service
}

func newMediaDeps(t *testing.T) *mediaDeps {
	t.Helper()
	ctrl := gomock.NewController(t)
	d := &mediaDeps{
		authRepo:      authmocks.NewMockAuthRepository(ctrl),
		profileRepo:   usermocks.NewMockUserRepository(ctrl),
		quotaRepo:     usermocks.NewMockQuotaRepository(ctrl),
		gifRepo:       mediamocks.NewMockGifRepository(ctrl),
		shareRepo:     sharemocks.NewMockShareRepository(ctrl),
		lastVideoRepo: mediamocks.NewMockLastVideoRepository(ctrl),
		storage:       mediamocks.NewMockStorageRepository(ctrl),
		tnx:           dbmocks.NewMockTxManager(ctrl),
		queue:         queuemocks.NewMockQueue(ctrl),
		cache:         cachemocks.NewMockCache(ctrl),
		processor:     processormocks.NewMockVideoProcessor(ctrl),
	}
	d.svc = appmedia.NewService(
		d.authRepo, d.profileRepo, d.quotaRepo, d.gifRepo, d.shareRepo,
		d.lastVideoRepo, d.storage, d.tnx, d.queue, d.cache, d.processor,
		&config.Config{},
	)
	return d
}

func (d *mediaDeps) passthroughRC(times int) {
	d.tnx.EXPECT().
		WithRC(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
			return fn(ctx)
		}).
		Times(times)
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse(%q): %v", raw, err)
	}
	return u
}

// gifUpdateMatcher asserts the *contents* of the update request reaching the
// repository, so a handler that drops the body cannot pass unnoticed.
type gifUpdateMatcher struct {
	name    *string
	status  *string
	persist *bool
	got     domainmedia.GifUpdateRequest
}

func (m *gifUpdateMatcher) Matches(x any) bool {
	r, ok := x.(domainmedia.GifUpdateRequest)
	if !ok {
		return false
	}
	m.got = r
	eqStr := func(a, b *string) bool {
		if a == nil || b == nil {
			return a == b
		}
		return *a == *b
	}
	eqBool := func(a, b *bool) bool {
		if a == nil || b == nil {
			return a == b
		}
		return *a == *b
	}
	return eqStr(m.name, r.Name) && eqStr(m.status, r.Status) && eqBool(m.persist, r.Persist)
}

func (m *gifUpdateMatcher) String() string {
	s := "media.GifUpdateRequest{"
	if m.name != nil {
		s += "Name:" + *m.name
	}
	if m.status != nil {
		s += " Status:" + *m.status
	}
	s += "}"
	return s
}

func strptr(s string) *string { return &s }
func boolptr(b bool) *bool    { return &b }

// ===========================================================================
// Update
// ===========================================================================

func TestUpdate_ForwardsEveryRequestedFieldToTheRepository(t *testing.T) {
	d := newMediaDeps(t)
	d.passthroughRC(1)
	now := time.Now().UTC()
	etag := now.Format(time.RFC3339Nano)

	d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq("gif-1")).Return("user-1", nil).Times(1)
	d.gifRepo.EXPECT().
		GetByKey(gomock.Any(), gomock.Eq("gif-1"), gomock.Eq(true)).
		Return(domainmedia.GifResponse{Key: "gif-1", Name: "old.gif", UpdatedAt: now}, nil).
		Times(1)

	// The exact fields the caller asked for must arrive at the repo — not a
	// zero-value struct.
	d.gifRepo.EXPECT().
		Update(gomock.Any(), gomock.Eq("gif-1"), &gifUpdateMatcher{
			name:    strptr("new.gif"),
			status:  strptr("ready"),
			persist: boolptr(true),
		}).
		Return(domainmedia.GifResponse{Key: "gif-1", Name: "new.gif", Status: "ready", Persist: true}, nil).
		Times(1)

	req := domainmedia.GifUpdateRequest{
		Name: strptr("new.gif"), Status: strptr("ready"), Persist: boolptr(true),
	}
	got, err := d.svc.Update(context.Background(), "user-1", "gif-1", req, etag)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Name != "new.gif" {
		t.Errorf("Name = %q, want new.gif", got.Name)
	}
}

// IDOR: user B must not be able to rename or unpublish user A's GIF.
func TestUpdate_NonOwnerIsRejectedBeforeAnyWrite(t *testing.T) {
	d := newMediaDeps(t)

	d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq("victims-gif")).
		Return("victim-user", nil).Times(1)

	// WithRC / GetByKey / Update must not be reached.
	_, err := d.svc.Update(context.Background(), "attacker", "victims-gif",
		domainmedia.GifUpdateRequest{Name: strptr("pwned.gif")}, "etag")
	if !errors.Is(err, domainmedia.ErrGifOwnerMismatch) {
		t.Errorf("err = %v, want ErrGifOwnerMismatch", err)
	}
}

// Optimistic concurrency: a stale If-Match must abort the write.
func TestUpdate_StaleETagAbortsTheWrite(t *testing.T) {
	d := newMediaDeps(t)
	d.passthroughRC(1)

	d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Any()).Return("user-1", nil).Times(1)
	d.gifRepo.EXPECT().
		GetByKey(gomock.Any(), gomock.Eq("gif-1"), gomock.Eq(true)).
		Return(domainmedia.GifResponse{Key: "gif-1", UpdatedAt: time.Now().UTC()}, nil).
		Times(1)

	// Update must not be called.
	_, err := d.svc.Update(context.Background(), "user-1", "gif-1",
		domainmedia.GifUpdateRequest{Name: strptr("x.gif")}, "1999-01-01T00:00:00Z")
	if !errors.Is(err, domainmedia.ErrETagValidationFailed) {
		t.Errorf("err = %v, want ErrETagValidationFailed", err)
	}
}

func TestUpdate_MissingGif(t *testing.T) {
	d := newMediaDeps(t)
	d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq("nope")).Return("", sql.ErrNoRows).Times(1)

	_, err := d.svc.Update(context.Background(), "user-1", "nope",
		domainmedia.GifUpdateRequest{}, "etag")
	if !errors.Is(err, domainmedia.ErrGifNotFound) {
		t.Errorf("err = %v, want ErrGifNotFound", err)
	}
}

// The row is locked FOR UPDATE inside the transaction, so N concurrent renames
// must all be serialised through it and none may panic. Run with -race.
func TestUpdate_ConcurrentRenamesAreSerialised(t *testing.T) {
	d := newMediaDeps(t)
	const n = 12
	now := time.Now().UTC()
	etag := now.Format(time.RFC3339Nano)

	d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Any()).Return("user-1", nil).Times(n)
	d.tnx.EXPECT().
		WithRC(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
			return fn(ctx)
		}).
		Times(n)

	var mu sync.Mutex
	d.gifRepo.EXPECT().
		GetByKey(gomock.Any(), gomock.Any(), gomock.Eq(true)).
		DoAndReturn(func(_ context.Context, _ string, _ bool) (domainmedia.GifResponse, error) {
			mu.Lock()
			defer mu.Unlock()
			return domainmedia.GifResponse{Key: "gif-1", UpdatedAt: now}, nil
		}).
		Times(n)

	var updates int64
	d.gifRepo.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, r domainmedia.GifUpdateRequest) (domainmedia.GifResponse, error) {
			atomic.AddInt64(&updates, 1)
			return domainmedia.GifResponse{Key: "gif-1", Name: *r.Name}, nil
		}).
		Times(n)

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := "rename.gif"
			if _, err := d.svc.Update(context.Background(), "user-1", "gif-1",
				domainmedia.GifUpdateRequest{Name: &name}, etag); err != nil {
				t.Errorf("concurrent Update: %v", err)
			}
		}(i)
	}
	wg.Wait()

	if got := atomic.LoadInt64(&updates); got != n {
		t.Errorf("%d updates reached the repo, want %d", got, n)
	}
}

// ===========================================================================
// Status — cache miss must not look like a backend fault
// ===========================================================================

// EXPECTED TO FAIL: internal/app/media/status.go treats every error from
// cache.Get identically:
//
//	status, err := s.cache.Get(ctx, lookupKey); if err != nil { return "", "", err }
//
// go-redis returns redis.Nil for a key that does not exist, so an ordinary
// cache miss — an unknown or long-finished upload — is bubbled up as an error
// and the handler answers 500 Internal Server Error. The `status == ""` ->
// "failed" branch below it is dead code, because a miss never gets that far.
//
// Contract: a miss (cache.ErrCacheMiss) is a normal outcome and must be
// reported as a status, while a real backend fault must stay an error.
func TestStatus_CacheMissIsNotAnInternalError(t *testing.T) {
	d := newMediaDeps(t)

	d.cache.EXPECT().
		Get(gomock.Any(), gomock.Eq("uploading:gif-1")).
		Return("", portcache.ErrCacheMiss).
		Times(1)

	_, status, err := d.svc.Status(context.Background(), "user-1", "gif-1")
	if err != nil {
		t.Fatalf("a cache miss surfaced as an error (%v); the caller answers 500 "+
			"for an upload it simply has no record of", err)
	}
	if status != "failed" {
		t.Errorf("status = %q, want %q for an unknown upload", status, "failed")
	}
}

func TestStatus_BackendOutageStaysAnError(t *testing.T) {
	d := newMediaDeps(t)
	outage := errors.New("dial tcp 127.0.0.1:6379: connect: connection refused")

	d.cache.EXPECT().
		Get(gomock.Any(), gomock.Eq("uploading:gif-1")).
		Return("", outage).
		Times(1)

	_, status, err := d.svc.Status(context.Background(), "user-1", "gif-1")
	if err == nil {
		t.Fatal("a Redis outage was reported as a normal status")
	}
	if status == "failed" {
		t.Error("a Redis outage was reported to the user as a failed upload")
	}
}

func TestStatus_OkLooksUpTheLastVideo(t *testing.T) {
	d := newMediaDeps(t)

	d.cache.EXPECT().Get(gomock.Any(), gomock.Eq("uploading:gif-1")).Return("ok", nil).Times(1)
	d.lastVideoRepo.EXPECT().
		GetLastVideo(gomock.Any(), gomock.Eq("user-1")).
		Return(domainmedia.LastUploadResponse{FileKey: "user-1:abc.mp4"}, nil).
		Times(1)

	key, status, err := d.svc.Status(context.Background(), "user-1", "gif-1")
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status != "ok" || key != "user-1:abc.mp4" {
		t.Errorf("got (%q, %q), want (user-1:abc.mp4, ok)", key, status)
	}
}

func TestStatus_StillUploadingDoesNotTouchTheDatabase(t *testing.T) {
	d := newMediaDeps(t)
	d.cache.EXPECT().Get(gomock.Any(), gomock.Any()).Return("uploading", nil).Times(1)
	// GetLastVideo must not be called.
	_, status, err := d.svc.Status(context.Background(), "user-1", "gif-1")
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status != "uploading" {
		t.Errorf("status = %q, want uploading", status)
	}
}

// ===========================================================================
// ConversionStatus
// ===========================================================================

// EXPECTED TO FAIL: internal/app/media/conversion_status.go wraps any cache.Get
// error in ErrCacheGetFailed, so an unknown job id (a miss) is indistinguishable
// from a Redis outage. media.JOB_NOT_FOUND exists in the domain and the handler
// maps it to 404, but the service can never produce it.
func TestConversionStatus_UnknownJobIsDistinguishableFromAnOutage(t *testing.T) {
	t.Run("unknown job id", func(t *testing.T) {
		d := newMediaDeps(t)
		d.cache.EXPECT().
			Get(gomock.Any(), gomock.Eq("messaage_queue:job_id:no-such-job")).
			Return("", portcache.ErrCacheMiss).
			Times(1)
		d.cache.EXPECT().Get(gomock.Any(), gomock.Any()).Return("", portcache.ErrCacheMiss).AnyTimes()

		_, err := d.svc.ConversionStatus(context.Background(), "no-such-job")
		if err == nil {
			t.Fatal("want an error for an unknown job id")
		}
		if !errors.Is(err, portcache.ErrCacheMiss) {
			t.Errorf("err = %v; an unknown job id must stay distinguishable from a "+
				"backend fault so the handler can answer 404 JOB_NOT_FOUND rather than 500", err)
		}
	})

	t.Run("backend outage", func(t *testing.T) {
		d := newMediaDeps(t)
		outage := errors.New("connection refused")
		d.cache.EXPECT().Get(gomock.Any(), gomock.Any()).Return("", outage).Times(1)

		_, err := d.svc.ConversionStatus(context.Background(), "job-1")
		if errors.Is(err, portcache.ErrCacheMiss) {
			t.Error("a backend outage was reported as a cache miss")
		}
	})
}

func TestConversionStatus_ReadsBothJobAndGifKeys(t *testing.T) {
	d := newMediaDeps(t)

	d.cache.EXPECT().Get(gomock.Any(), gomock.Eq("messaage_queue:job_id:job-1")).
		Return("done", nil).Times(1)
	d.cache.EXPECT().Get(gomock.Any(), gomock.Eq("messaage_queue_gif:job_id:job-1")).
		Return("gif-42", nil).Times(1)

	res, err := d.svc.ConversionStatus(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("ConversionStatus: %v", err)
	}
	if res.Status != "done" || res.GifId != "gif-42" || res.JobId != "job-1" {
		t.Errorf("got %+v, want {JobId:job-1 Status:done GifId:gif-42}", res)
	}
}

// ===========================================================================
// Quota enforcement
// ===========================================================================

// EXPECTED TO FAIL: user.QuotaRepository is injected into the media service and
// user.Quota carries UsedBytes/TotalBytes/GifCount/GitCount(gif_limit), but no
// code path in internal/app/media reads any of it. Quota is tracked and shown
// to the user, never enforced — a user at 100% can keep uploading forever.
func TestUpload_IsRejectedWhenTheUserIsOverQuota(t *testing.T) {
	d := newMediaDeps(t)
	userID := uuid.New()

	d.quotaRepo.EXPECT().
		GetById(gomock.Any(), gomock.Eq(userID.String())).
		Return(&domainuser.Quota{
			UserID:     userID,
			UsedBytes:  500 * 1024 * 1024,
			TotalBytes: 500 * 1024 * 1024, // completely full
			GifCount:   50,
			GitCount:   50,
		}, nil).
		AnyTimes()

	// Handing out a presigned PUT here lets the user write past their quota.
	d.storage.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, key string, _ time.Duration) (*url.URL, error) {
			t.Errorf("a presigned upload URL was issued for key %q to a user "+
				"who is at 100%% of their storage quota", key)
			return mustURL(t, "https://storage/put"), nil
		}).
		AnyTimes()
	d.cache.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	_, err := d.svc.Upload(context.Background(), "clip.mp4", userID.String())
	if err == nil {
		t.Error("Upload succeeded for a user who is over quota")
	}
}

// EXPECTED TO FAIL: same gap on the conversion path — a user at their gif_limit
// can still enqueue unlimited conversion jobs, each of which produces a new GIF.
func TestConvert_IsRejectedWhenTheUserIsAtTheirGifLimit(t *testing.T) {
	d := newMediaDeps(t)
	userID := uuid.New()

	d.quotaRepo.EXPECT().
		GetById(gomock.Any(), gomock.Eq(userID.String())).
		Return(&domainuser.Quota{
			UserID:   userID,
			GifCount: 50,
			GitCount: 50, // gif_limit reached
		}, nil).
		AnyTimes()

	d.cache.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	d.queue.EXPECT().
		PublishVideo(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, msg queue.VideoMessage) error {
			t.Errorf("conversion job %q was enqueued for a user already at their "+
				"gif_limit of 50", msg.JobId)
			return nil
		}).
		AnyTimes()

	_, err := d.svc.Convert(context.Background(), userID.String(), "upload-1", 0, 5, 15, 480, false)
	if err == nil {
		t.Error("Convert succeeded for a user at their GIF limit")
	}
}

// EXPECTED TO FAIL: with no quota check at all there is nothing to double-spend
// against, so N concurrent conversions for a user with 1 slot left all go
// through. Once a check exists this test guards it against a TOCTOU race.
func TestConvert_ConcurrentJobsCannotDoubleSpendTheLastQuotaSlot(t *testing.T) {
	d := newMediaDeps(t)
	const n = 20
	userID := uuid.New()

	d.quotaRepo.EXPECT().
		GetById(gomock.Any(), gomock.Any()).
		Return(&domainuser.Quota{UserID: userID, GifCount: 49, GitCount: 50}, nil).
		AnyTimes()
	d.cache.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	var published int64
	d.queue.EXPECT().
		PublishVideo(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ queue.VideoMessage) error {
			atomic.AddInt64(&published, 1)
			return nil
		}).
		AnyTimes()

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = d.svc.Convert(context.Background(), userID.String(), "upload-1", 0, 5, 15, 480, false)
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt64(&published); got > 1 {
		t.Errorf("%d conversion jobs were enqueued against a single remaining "+
			"quota slot; quota can be double-spent by concurrent requests", got)
	}
}

// ===========================================================================
// Convert — job bookkeeping
// ===========================================================================

func TestConvert_PublishesTheRequestedParametersVerbatim(t *testing.T) {
	d := newMediaDeps(t)

	var jobID string
	d.cache.EXPECT().
		Set(gomock.Any(), gomock.Any(), gomock.Eq("queued"), gomock.Eq(5*time.Minute)).
		DoAndReturn(func(_ context.Context, key, _ string, _ time.Duration) error {
			return nil
		}).
		Times(2) // one entry for the job, one for the resulting gif

	d.queue.EXPECT().
		PublishVideo(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, msg queue.VideoMessage) error {
			jobID = msg.JobId
			if msg.UserID != "user-1" {
				t.Errorf("UserID = %q, want user-1", msg.UserID)
			}
			if msg.Key != "upload-1" {
				t.Errorf("Key = %q, want upload-1", msg.Key)
			}
			if msg.Start != 1.5 || msg.End != 9.25 {
				t.Errorf("trim = [%v,%v], want [1.5,9.25]", msg.Start, msg.End)
			}
			if msg.Width != 640 || msg.FPS != 24 {
				t.Errorf("got width=%d fps=%d, want 640/24", msg.Width, msg.FPS)
			}
			if !msg.Loop {
				t.Error("Loop was dropped on the way to the queue")
			}
			if msg.Retries != 0 {
				t.Errorf("Retries = %d, want 0 on a fresh job", msg.Retries)
			}
			return nil
		}).
		Times(1)

	res, err := d.svc.Convert(context.Background(), "user-1", "upload-1", 1.5, 9.25, 24, 640, true)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if res.Status != "queued" {
		t.Errorf("Status = %q, want queued", res.Status)
	}
	if res.Id != jobID {
		t.Errorf("returned job id %q != published job id %q", res.Id, jobID)
	}
}

// If the job's status entry cannot be written, the job must not be enqueued —
// otherwise a worker processes a job whose status nobody can ever read.
func TestConvert_CacheFailureMeansNothingIsEnqueued(t *testing.T) {
	d := newMediaDeps(t)

	d.cache.EXPECT().
		Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errors.New("redis down")).
		Times(1)

	// PublishVideo must not be called.
	if _, err := d.svc.Convert(context.Background(), "user-1", "upload-1", 0, 5, 15, 480, false); err == nil {
		t.Fatal("want an error when the job status cannot be recorded")
	}
}

// EXPECTED TO FAIL: internal/app/media/convert.go writes two cache entries and
// then publishes. If the publish fails the entries are left behind saying
// "queued" for a job no worker will ever see, and they only disappear after the
// 5-minute TTL. The status entries should be rolled back on a publish failure.
func TestConvert_PublishFailureDoesNotLeaveAPhantomQueuedJob(t *testing.T) {
	d := newMediaDeps(t)

	written := map[string]string{}
	var mu sync.Mutex
	d.cache.EXPECT().
		Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, k, v string, _ time.Duration) error {
			mu.Lock()
			defer mu.Unlock()
			written[k] = v
			return nil
		}).
		AnyTimes()

	d.queue.EXPECT().
		PublishVideo(gomock.Any(), gomock.Any()).
		Return(errors.New("rabbitmq unreachable")).
		Times(1)

	_, err := d.svc.Convert(context.Background(), "user-1", "upload-1", 0, 5, 15, 480, false)
	if err == nil {
		t.Fatal("want an error when the job cannot be published")
	}

	mu.Lock()
	defer mu.Unlock()
	for k, v := range written {
		if v == "queued" {
			t.Errorf("cache key %q was left reading %q after the publish failed; "+
				"the client will poll a job that will never run", k, v)
		}
	}
}

// ===========================================================================
// Upload
// ===========================================================================

func TestUpload_KeyLayoutAndStatusSeeding(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantExt  string
	}{
		{"ordinary name", "holiday.mp4", ".mp4"},
		{"uppercase extension", "CLIP.MOV", ".MOV"},
		{"multiple dots", "my.holiday.clip.webm", ".webm"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := newMediaDeps(t)
			var gotKey string

			d.storage.EXPECT().
				Create(gomock.Any(), gomock.Any(), gomock.Eq(5*time.Minute)).
				DoAndReturn(func(_ context.Context, key string, _ time.Duration) (*url.URL, error) {
					gotKey = key
					return mustURL(t, "https://storage/presigned-put"), nil
				}).
				Times(1)

			d.cache.EXPECT().
				Set(gomock.Any(), gomock.Any(), gomock.Eq("uploading"), gomock.Eq(5*time.Minute)).
				Return(nil).
				Times(1)

			res, err := d.svc.Upload(context.Background(), tc.filename, "user-1")
			if err != nil {
				t.Fatalf("Upload: %v", err)
			}
			// key = <userId>:<uuid><ext>
			if len(gotKey) < len("user-1:") || gotKey[:7] != "user-1:" {
				t.Errorf("key %q is not namespaced by the user id", gotKey)
			}
			if tc.wantExt != "" && !hasSuffix(gotKey, tc.wantExt) {
				t.Errorf("key %q does not end in %q", gotKey, tc.wantExt)
			}
			if res.Key != gotKey {
				t.Errorf("returned key %q != storage key %q", res.Key, gotKey)
			}
			if res.ExpireIn != 300 {
				t.Errorf("ExpireIn = %d, want 300", res.ExpireIn)
			}
		})
	}
}

// EXPECTED TO FAIL: internal/app/media/upload.go derives the object key with
// filepath.Ext and never validates it. A filename with no extension yields a
// key with no extension, and ffprobe/ffmpeg later have to guess the container.
// media.ErrInvalidExt exists in the domain for exactly this and is never used
// on this path.
func TestUpload_RejectsFilenameWithNoExtension(t *testing.T) {
	cases := []string{"", "noextension", "trailingdot."}

	for _, filename := range cases {
		t.Run("filename="+filename, func(t *testing.T) {
			d := newMediaDeps(t)

			d.storage.EXPECT().
				Create(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, key string, _ time.Duration) (*url.URL, error) {
					t.Errorf("an upload slot was issued for key %q from filename %q, "+
						"which carries no usable file extension", key, filename)
					return mustURL(t, "https://storage/put"), nil
				}).
				AnyTimes()
			d.cache.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			if _, err := d.svc.Upload(context.Background(), filename, "user-1"); err == nil {
				t.Errorf("Upload(%q) succeeded; want media.ErrInvalidExt", filename)
			}
		})
	}
}

// If the status entry cannot be seeded the client can never poll the upload, so
// handing back a presigned URL is a dead end.
func TestUpload_StatusSeedFailureIsReported(t *testing.T) {
	d := newMediaDeps(t)

	d.storage.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(mustURL(t, "https://storage/put"), nil).Times(1)
	d.cache.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errors.New("redis down")).Times(1)

	if _, err := d.svc.Upload(context.Background(), "clip.mp4", "user-1"); err == nil {
		t.Error("want an error when the upload status cannot be recorded")
	}
}

// ===========================================================================
// Download / Stream / Thumbnail — object-level authorisation
// ===========================================================================

func TestDownload(t *testing.T) {
	tests := []struct {
		name        string
		gifOwner    string
		shareOwner  string
		shareErr    error
		caller      string
		wantAllowed bool
	}{
		{name: "owner downloads their own gif", gifOwner: "user-1", shareErr: sql.ErrNoRows, caller: "user-1", wantAllowed: true},
		{name: "valid share recipient", gifOwner: "user-1", shareOwner: "user-1", caller: "user-2", wantAllowed: true},
		{name: "stranger with no share", gifOwner: "user-1", shareErr: sql.ErrNoRows, caller: "user-3"},
		{name: "expired share is filtered out by the repo", gifOwner: "user-1", shareErr: sql.ErrNoRows, caller: "user-2"},
		{name: "share issued by someone who is not the owner", gifOwner: "user-1", shareOwner: "impostor", caller: "user-2"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := newMediaDeps(t)

			d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq("gif-1")).
				Return(tc.gifOwner, nil).Times(1)
			d.shareRepo.EXPECT().
				GetOwner(gomock.Any(), gomock.Eq(tc.caller), gomock.Eq("gif-1")).
				Return(tc.shareOwner, tc.shareErr).
				Times(1)

			if tc.wantAllowed {
				d.storage.EXPECT().
					Download(gomock.Any(), gomock.Eq("gif-1"), gomock.Eq(5*time.Minute)).
					Return(mustURL(t, "https://storage/get?sig=x"), nil).
					Times(1)
			}
			// when not allowed, storage.Download must not be reached

			got, err := d.svc.Download(context.Background(), tc.caller, "gif-1")
			if tc.wantAllowed {
				if err != nil {
					t.Fatalf("Download: %v", err)
				}
				if got == "" {
					t.Error("no download URL returned")
				}
				return
			}
			if !errors.Is(err, domainmedia.ErrGifOwnerMismatch) {
				t.Errorf("err = %v, want ErrGifOwnerMismatch — %s got a download URL", err, tc.caller)
			}
		})
	}
}

// EXPECTED TO FAIL: internal/app/media/stream.go takes userId and never reads
// it. It goes straight to storage.GetStreamURL(key), so ANY authenticated user
// can mint a presigned stream URL for ANY object key belonging to anyone.
// Every sibling method (Download, GetByKey, Delete, GetGifThumbnail) checks
// gifRepo.GetOwner first; Stream does not.
func TestStream_NonOwnerCannotMintAStreamUrl(t *testing.T) {
	d := newMediaDeps(t)

	// The ownership lookup that ought to happen.
	d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Any()).Return("victim-user", nil).AnyTimes()
	d.shareRepo.EXPECT().GetOwner(gomock.Any(), gomock.Any(), gomock.Any()).
		Return("", sql.ErrNoRows).AnyTimes()

	d.storage.EXPECT().
		GetStreamURL(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, key string, _ time.Duration) (*url.URL, error) {
			t.Errorf("a presigned stream URL was minted for key %q on behalf of "+
				"attacker-user, who neither owns it nor holds a share for it", key)
			return mustURL(t, "https://storage/stream?sig=x"), nil
		}).
		AnyTimes()

	res, err := d.svc.Stream(context.Background(), "attacker-user", "victims-gif-key")
	if err == nil {
		t.Errorf("Stream returned %q to a non-owner; want ErrGifOwnerMismatch", res.PresignedUrl)
	}
}

func TestGetGifThumbnail(t *testing.T) {
	t.Run("owner with a thumbnail", func(t *testing.T) {
		d := newMediaDeps(t)
		d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq("gif-1")).Return("user-1", nil).Times(1)
		d.gifRepo.EXPECT().GetByKey(gomock.Any(), gomock.Eq("gif-1"), gomock.Eq(false)).
			Return(domainmedia.GifResponse{Key: "gif-1", ThumbnailUrl: "thumb-1.jpg"}, nil).Times(1)
		d.storage.EXPECT().GetThumbnailURL(gomock.Any(), gomock.Eq("thumb-1.jpg")).
			Return(mustURL(t, "https://storage/thumb-1.jpg"), nil).Times(1)

		got, err := d.svc.GetGifThumbnail(context.Background(), "user-1", "gif-1")
		if err != nil {
			t.Fatalf("GetGifThumbnail: %v", err)
		}
		if got == "" {
			t.Error("empty thumbnail URL")
		}
	})

	t.Run("gif exists but has no thumbnail", func(t *testing.T) {
		d := newMediaDeps(t)
		d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Any()).Return("user-1", nil).Times(1)
		d.gifRepo.EXPECT().GetByKey(gomock.Any(), gomock.Any(), gomock.Eq(false)).
			Return(domainmedia.GifResponse{Key: "gif-1", ThumbnailUrl: ""}, nil).Times(1)
		// storage must not be asked for an empty key
		_, err := d.svc.GetGifThumbnail(context.Background(), "user-1", "gif-1")
		if !errors.Is(err, domainmedia.ErrThumbnailNotFound) {
			t.Errorf("err = %v, want ErrThumbnailNotFound", err)
		}
	})

	t.Run("gif does not exist", func(t *testing.T) {
		d := newMediaDeps(t)
		d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Any()).Return("", sql.ErrNoRows).Times(1)
		_, err := d.svc.GetGifThumbnail(context.Background(), "user-1", "nope")
		if !errors.Is(err, domainmedia.ErrGifNotFound) {
			t.Errorf("err = %v, want ErrGifNotFound", err)
		}
		if errors.Is(err, domainmedia.ErrThumbnailNotFound) {
			t.Error("a missing gif was reported as a missing thumbnail")
		}
	})

	t.Run("non-owner is refused", func(t *testing.T) {
		d := newMediaDeps(t)
		d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Any()).Return("victim", nil).Times(1)
		// GetByKey and storage must not be reached
		_, err := d.svc.GetGifThumbnail(context.Background(), "attacker", "gif-1")
		if !errors.Is(err, domainmedia.ErrGifOwnerMismatch) {
			t.Errorf("err = %v, want ErrGifOwnerMismatch", err)
		}
	})
}

// ===========================================================================
// GetByKey / Delete / Save
// ===========================================================================

func TestGetByKey_NonOwnerIsRefused(t *testing.T) {
	d := newMediaDeps(t)
	d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq("gif-1")).Return("victim", nil).Times(1)
	// GetByKey must not be reached
	_, err := d.svc.GetByKey(context.Background(), "attacker", "gif-1")
	if !errors.Is(err, domainmedia.ErrGifOwnerMismatch) {
		t.Errorf("err = %v, want ErrGifOwnerMismatch", err)
	}
}

func TestDelete_NonOwnerIsRefused(t *testing.T) {
	d := newMediaDeps(t)
	d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq("gif-1")).Return("victim", nil).Times(1)
	// gifRepo.Delete must not be reached
	if err := d.svc.Delete(context.Background(), "attacker", "gif-1"); !errors.Is(err, domainmedia.ErrGifOwnerMismatch) {
		t.Errorf("err = %v, want ErrGifOwnerMismatch", err)
	}
}

func TestDelete_OwnerDeletesTheirOwnGif(t *testing.T) {
	d := newMediaDeps(t)
	d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq("gif-1")).Return("user-1", nil).Times(1)
	d.gifRepo.EXPECT().Delete(gomock.Any(), gomock.Eq("gif-1")).Return(nil).Times(1)

	if err := d.svc.Delete(context.Background(), "user-1", "gif-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

// EXPECTED TO FAIL: internal/app/media/save.go is
//
//	// To-do, validate user
//	return s.gifRepo.SaveRecent(ctx, key)
//
// It never checks ownership, so any authenticated user can pin any other user's
// GIF into their recents by key.
func TestSave_NonOwnerCannotSaveSomeoneElsesGif(t *testing.T) {
	d := newMediaDeps(t)

	d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Any()).Return("victim-user", nil).AnyTimes()
	d.gifRepo.EXPECT().
		SaveRecent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, key string) error {
			t.Errorf("attacker-user saved gif %q, which belongs to victim-user", key)
			return nil
		}).
		AnyTimes()

	if err := d.svc.Save(context.Background(), "attacker-user", "victims-gif"); err == nil {
		t.Error("Save succeeded for a non-owner; want ErrGifOwnerMismatch")
	}
}

// ===========================================================================
// Listing
// ===========================================================================

func TestGetGifs_ScopesToTheCallerAndForwardsTheFilter(t *testing.T) {
	d := newMediaDeps(t)
	rows := []domainmedia.GifResponse{{Key: "a"}, {Key: "b"}}

	d.gifRepo.EXPECT().
		Get(gomock.Any(), gomock.Eq("user-1"), gomock.Eq("ready")).
		Return(rows, nil).
		Times(1)

	res, err := d.svc.GetGifs(context.Background(), "user-1", "ready")
	if err != nil {
		t.Fatalf("GetGifs: %v", err)
	}
	if res.Total != 2 {
		t.Errorf("Total = %d, want 2", res.Total)
	}
}

func TestGetRecents_EmptyResultIsNotAnError(t *testing.T) {
	d := newMediaDeps(t)
	d.gifRepo.EXPECT().GetRecents(gomock.Any(), gomock.Eq("user-1")).
		Return([]domainmedia.GifResponse{}, nil).Times(1)

	got, err := d.svc.GetRecents(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetRecents: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

func TestLastVideo_MissingRowIsReportedAsNotFound(t *testing.T) {
	d := newMediaDeps(t)
	d.lastVideoRepo.EXPECT().GetLastVideo(gomock.Any(), gomock.Eq("user-1")).
		Return(domainmedia.LastUploadResponse{}, sql.ErrNoRows).Times(1)

	_, err := d.svc.LastVideo(context.Background(), "user-1")
	if !errors.Is(err, domainmedia.ErrLastVideoNotFound) {
		t.Errorf("err = %v, want ErrLastVideoNotFound", err)
	}
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
