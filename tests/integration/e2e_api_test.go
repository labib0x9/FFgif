package integration_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labib0x9/ffgif/config"
	authapp "github.com/labib0x9/ffgif/internal/app/auth"
	mediaapp "github.com/labib0x9/ffgif/internal/app/media"
	shareapp "github.com/labib0x9/ffgif/internal/app/share"
	userapp "github.com/labib0x9/ffgif/internal/app/user"
	domainauth "github.com/labib0x9/ffgif/internal/domain/auth"
	domainmedia "github.com/labib0x9/ffgif/internal/domain/media"
	domainshare "github.com/labib0x9/ffgif/internal/domain/share"
	domainuser "github.com/labib0x9/ffgif/internal/domain/user"
	domainprocessor "github.com/labib0x9/ffgif/internal/port/processor"
	domainqueue "github.com/labib0x9/ffgif/internal/port/queue"
	rest "github.com/labib0x9/ffgif/internal/transport/http"
	authhandler "github.com/labib0x9/ffgif/internal/transport/http/handlers/auth"
	mediahandler "github.com/labib0x9/ffgif/internal/transport/http/handlers/media"
	sharehandler "github.com/labib0x9/ffgif/internal/transport/http/handlers/share"
	"github.com/labib0x9/ffgif/internal/transport/http/handlers/static"
	userhandler "github.com/labib0x9/ffgif/internal/transport/http/handlers/user"
	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
	"github.com/labib0x9/ffgif/pkg/password"
	amqp "github.com/rabbitmq/amqp091-go"
)

// --- In-Memory Integration Test Fixtures ---

type inMemoryCache struct {
	store map[string]string
}

func newInMemoryCache() *inMemoryCache {
	return &inMemoryCache{store: make(map[string]string)}
}
func (c *inMemoryCache) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	c.store[key] = value
	return nil
}
func (c *inMemoryCache) Get(ctx context.Context, key string) (string, error) {
	if v, ok := c.store[key]; ok {
		return v, nil
	}
	return "", domainauth.ErrTokenFetchFailed
}

type inMemoryQueue struct {
	PublishedEmails []domainqueue.EmailMessage
	PublishedVideos []domainqueue.VideoMessage
}

func (q *inMemoryQueue) PublishEmail(ctx context.Context, msg domainqueue.EmailMessage) error {
	q.PublishedEmails = append(q.PublishedEmails, msg)
	return nil
}
func (q *inMemoryQueue) PublishVideo(ctx context.Context, msg domainqueue.VideoMessage) error {
	q.PublishedVideos = append(q.PublishedVideos, msg)
	return nil
}
func (q *inMemoryQueue) PublishSaveVideo(ctx context.Context, msg domainqueue.SaveVideoMessage) error {
	return nil
}
func (q *inMemoryQueue) PublishRetrySaveVideo(ctx context.Context, msg domainqueue.SaveVideoMessage) error {
	return nil
}
func (q *inMemoryQueue) ConsumeEmail(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error) {
	return nil, nil
}
func (q *inMemoryQueue) ConsumeSave(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error) {
	return nil, nil
}
func (q *inMemoryQueue) ConsumeVideo(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error) {
	return nil, nil
}
func (q *inMemoryQueue) ConsumeRawVideo(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error) {
	return nil, nil
}
func (q *inMemoryQueue) Close() error                           { return nil }
func (q *inMemoryQueue) CloseConsumerChannel(name string) error { return nil }

type inMemoryAuthRepo struct {
	users map[string]domainauth.User
}

func (r *inMemoryAuthRepo) GetByEmail(ctx context.Context, email string) (domainauth.User, error) {
	if u, ok := r.users[email]; ok {
		return u, nil
	}
	return domainauth.User{}, domainauth.ErrInvalidCredential
}
func (r *inMemoryAuthRepo) GetById(ctx context.Context, id uuid.UUID) (domainauth.User, error) {
	for _, u := range r.users {
		if u.Id == id {
			return u, nil
		}
	}
	return domainauth.User{}, domainauth.ErrInvalidCredential
}
func (r *inMemoryAuthRepo) Create(ctx context.Context, user domainauth.User) (domainauth.User, error) {
	user.Id = uuid.New()
	r.users[user.Email] = user
	return user, nil
}
func (r *inMemoryAuthRepo) DeleteById(ctx context.Context, id uuid.UUID) error {
	for email, u := range r.users {
		if u.Id == id {
			delete(r.users, email)
		}
	}
	return nil
}
func (r *inMemoryAuthRepo) DeleteByEmail(ctx context.Context, email string) error {
	delete(r.users, email)
	return nil
}
func (r *inMemoryAuthRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passHash string) error {
	for email, u := range r.users {
		if u.Id == id {
			u.PasswordHash = passHash
			r.users[email] = u
		}
	}
	return nil
}
func (r *inMemoryAuthRepo) SetVerified(ctx context.Context, userId uuid.UUID) error {
	for email, u := range r.users {
		if u.Id == userId {
			u.IsVerified = true
			r.users[email] = u
		}
	}
	return nil
}
func (r *inMemoryAuthRepo) Upgrade(ctx context.Context, id string, user domainauth.User) (domainauth.User, error) {
	return user, nil
}

type inMemoryVerifierRepo struct {
	tokens map[string]domainauth.Verifier
}

func (r *inMemoryVerifierRepo) Create(ctx context.Context, verifier domainauth.Verifier) error {
	verifier.Id = int64(len(r.tokens) + 1)
	r.tokens[verifier.Token] = verifier
	return nil
}
func (r *inMemoryVerifierRepo) GetByHash(ctx context.Context, tokenHash string) (domainauth.Verifier, error) {
	if v, ok := r.tokens[tokenHash]; ok {
		return v, nil
	}
	return domainauth.Verifier{}, domainauth.ErrInvalidToken
}
func (r *inMemoryVerifierRepo) GetById(ctx context.Context, userId uuid.UUID) (domainauth.Verifier, error) {
	for _, v := range r.tokens {
		if v.UserId == userId {
			return v, nil
		}
	}
	return domainauth.Verifier{}, sql.ErrNoRows
}
func (r *inMemoryVerifierRepo) Delete(ctx context.Context, id int64) error {
	for hash, v := range r.tokens {
		if v.Id == id {
			delete(r.tokens, hash)
		}
	}
	return nil
}

type inMemoryProfileRepo struct {
	profiles map[string]domainuser.ProfileResponse
	authRepo *inMemoryAuthRepo
}

func (r *inMemoryProfileRepo) GetProfile(ctx context.Context, id string) (domainuser.ProfileResponse, error) {
	if p, ok := r.profiles[id]; ok {
		return p, nil
	}
	return domainuser.ProfileResponse{Username: "default", Email: "default@example.com"}, nil
}
func (r *inMemoryProfileRepo) UpdateProfile(ctx context.Context, profile domainuser.ProfileResponse, id string) (domainuser.ProfileResponse, error) {
	r.profiles[id] = profile
	return profile, nil
}
func (r *inMemoryProfileRepo) SetProfile(ctx context.Context, profile domainuser.Profile) error {
	return nil
}
func (r *inMemoryProfileRepo) ChangePassword(ctx context.Context, userId string, hash string) error {
	if r.authRepo != nil {
		uid, _ := uuid.Parse(userId)
		return r.authRepo.UpdatePassword(ctx, uid, hash)
	}
	return nil
}

type inMemoryQuotaRepo struct{}

func (r *inMemoryQuotaRepo) Create(ctx context.Context, quota domainuser.Quota) error { return nil }
func (r *inMemoryQuotaRepo) GetById(ctx context.Context, userId string) (*domainuser.Quota, error) {
	return &domainuser.Quota{TotalBytes: 1024 * 1024 * 100, GifCount: 20}, nil
}

type inMemoryGifRepo struct {
	gifs map[string]domainmedia.Gif
}

func (r *inMemoryGifRepo) Create(ctx context.Context, gif domainmedia.Gif) error {
	r.gifs[gif.Key] = gif
	return nil
}
func (r *inMemoryGifRepo) Get(ctx context.Context, user_id string, status string) ([]domainmedia.GifResponse, error) {
	var results []domainmedia.GifResponse
	for _, g := range r.gifs {
		if g.UserId == user_id {
			results = append(results, domainmedia.GifResponse{
				Key: g.Key,
				Url: "https://minio.local/gifs/" + g.Key,
			})
		}
	}
	return results, nil
}
func (r *inMemoryGifRepo) GetByKey(ctx context.Context, key string) (domainmedia.GifResponse, error) {
	if g, ok := r.gifs[key]; ok {
		return domainmedia.GifResponse{
			Key:          g.Key,
			Url:          "https://minio.local/gifs/" + g.Key,
			ThumbnailUrl: g.ThumbnailUrl,
		}, nil
	}
	return domainmedia.GifResponse{}, sql.ErrNoRows
}
func (r *inMemoryGifRepo) GetRecents(ctx context.Context, user_id string) ([]domainmedia.GifResponse, error) {
	return r.Get(ctx, user_id, "all")
}
func (r *inMemoryGifRepo) Delete(ctx context.Context, key string) error {
	delete(r.gifs, key)
	return nil
}
func (r *inMemoryGifRepo) Update(ctx context.Context, key string, gif domainmedia.GifResponse) error {
	return nil
}
func (r *inMemoryGifRepo) SaveRecent(ctx context.Context, key string) error { return nil }
func (r *inMemoryGifRepo) GetOwner(ctx context.Context, key string) (string, error) {
	if g, ok := r.gifs[key]; ok {
		return g.UserId, nil
	}
	return "", sql.ErrNoRows
}

type inMemoryShareRepo struct {
	shares map[string]domainshare.Share
}

func (r *inMemoryShareRepo) Create(ctx context.Context, gif domainshare.Share) error {
	r.shares[gif.GifKey+":"+gif.SharedWith] = gif
	return nil
}
func (r *inMemoryShareRepo) Get(ctx context.Context, userID string) ([]domainshare.GifResponse, error) {
	var list []domainshare.GifResponse
	for _, s := range r.shares {
		if s.OwnerID == userID || s.SharedWith == userID {
			list = append(list, domainshare.GifResponse{
				GifKey:     s.GifKey,
				OwnerID:    s.OwnerID,
				SharedWith: s.SharedWith,
				ExpiresAt:  s.ExpiresAt,
			})
		}
	}
	return list, nil
}
func (r *inMemoryShareRepo) GetOwner(ctx context.Context, user string, key string) (string, error) {
	for _, s := range r.shares {
		if s.GifKey == key && s.SharedWith == user {
			return s.OwnerID, nil
		}
	}
	return "", sql.ErrNoRows
}
func (r *inMemoryShareRepo) Delete(ctx context.Context, key, shareWithId string) error {
	delete(r.shares, key+":"+shareWithId)
	return nil
}

type inMemoryStorageRepo struct{}

func (s *inMemoryStorageRepo) Create(ctx context.Context, key string, expirey time.Duration) (*url.URL, error) {
	return url.Parse("https://minio.local/upload/" + key)
}
func (s *inMemoryStorageRepo) Download(ctx context.Context, key string, expirey time.Duration) (*url.URL, error) {
	return url.Parse("https://minio.local/download/" + key)
}
func (s *inMemoryStorageRepo) IsExists(ctx context.Context, key string) (bool, error) { return true, nil }
func (s *inMemoryStorageRepo) Status(ctx context.Context, key string) (domainmedia.Info, error) {
	return domainmedia.Info{Size: 1024, ContentType: "video/mp4", UploadedAt: time.Now()}, nil
}
func (s *inMemoryStorageRepo) GetObject(ctx context.Context, start, end int64, key string) (domainmedia.Object, error) {
	return domainmedia.Object{}, nil
}
func (s *inMemoryStorageRepo) DownloadLocal(ctx context.Context, key, destPath string) error { return nil }
func (s *inMemoryStorageRepo) DownloadLocalRawVideo(ctx context.Context, key, destPath string) error {
	return nil
}
func (s *inMemoryStorageRepo) Upload(ctx context.Context, key, filePath, contentType string) error {
	return nil
}
func (s *inMemoryStorageRepo) Delete(ctx context.Context, key string) error { return nil }
func (s *inMemoryStorageRepo) GetStreamURL(ctx context.Context, key string, expiry time.Duration) (*url.URL, error) {
	return url.Parse("https://minio.local/stream/" + key)
}

type inMemoryLastVideoRepo struct {
	lastUploads map[string]domainmedia.LastUploadResponse
}

func (r *inMemoryLastVideoRepo) Create(ctx context.Context, upload domainmedia.LastUpload) error {
	r.lastUploads[upload.UserID.String()] = domainmedia.LastUploadResponse{
		UserID:      upload.UserID,
		FileKey:     upload.FileKey,
		Filename:    upload.Filename,
		ContentType: upload.ContentType,
	}
	return nil
}
func (r *inMemoryLastVideoRepo) GetLastVideo(ctx context.Context, user_id string) (domainmedia.LastUploadResponse, error) {
	if last, ok := r.lastUploads[user_id]; ok {
		return last, nil
	}
	return domainmedia.LastUploadResponse{}, sql.ErrNoRows
}

type inMemoryProcessor struct{}

func (p *inMemoryProcessor) Process(ctx context.Context, JobId string, Key string, Start float32, End float32, Width int, FPS int, Loop bool) (*domainprocessor.JobResult, error) {
	return &domainprocessor.JobResult{
		GifKey:   "converted_" + JobId + ".gif",
		ThumbKey: "thumb_" + JobId + ".jpg",
	}, nil
}
func (p *inMemoryProcessor) PreProcess(ctx context.Context, key string) (*domainprocessor.PrePrecessedResult, error) {
	return &domainprocessor.PrePrecessedResult{
		VideoKey:     "preprocessed_" + key,
		ThumbnailKey: "thumb_" + key + ".jpg",
		ContentType:  "video/mp4",
		Size:         "2048",
		Duration:     "5.0",
	}, nil
}

type inMemoryTx struct{}

func (t *inMemoryTx) With(ctx context.Context, fn func(ctx context.Context) (any, error)) (any, error) {
	return fn(ctx)
}

// --- End-to-End Integration Tests ---

func TestE2E_FullUserAndJobLifecycle(t *testing.T) {
	jwtSecret := []byte("integration-test-secret-key-32bytes")
	jwtProvider := jwtpkg.NewJwt(jwtSecret)
	hasher := password.NewHasher("test-pepper", 10)
	cache := newInMemoryCache()
	queue := &inMemoryQueue{}
	authRepo := &inMemoryAuthRepo{users: make(map[string]domainauth.User)}
	verifierRepo := &inMemoryVerifierRepo{tokens: make(map[string]domainauth.Verifier)}
	profileRepo := &inMemoryProfileRepo{profiles: make(map[string]domainuser.ProfileResponse), authRepo: authRepo}
	quotaRepo := &inMemoryQuotaRepo{}
	gifRepo := &inMemoryGifRepo{gifs: make(map[string]domainmedia.Gif)}
	shareRepo := &inMemoryShareRepo{shares: make(map[string]domainshare.Share)}
	storage := &inMemoryStorageRepo{}
	lastVideoRepo := &inMemoryLastVideoRepo{lastUploads: make(map[string]domainmedia.LastUploadResponse)}
	proc := &inMemoryProcessor{}
	txManager := &inMemoryTx{}

	cnf := &config.Config{
		Addr:      "127.0.0.1",
		Port:      8080,
		JwtSecret: jwtSecret,
	}

	middlewares := middleware.NewMiddlewares(cnf, cache, *jwtProvider)
	val := validator.New()

	authService := authapp.NewService(
		authRepo, verifierRepo, profileRepo, nil, quotaRepo,
		cache, queue, *jwtProvider, *hasher, txManager,
	)
	userService := userapp.NewService(profileRepo, quotaRepo, authRepo, *jwtProvider, *hasher)
	mediaService := mediaapp.NewService(authRepo, profileRepo, quotaRepo, gifRepo, shareRepo, lastVideoRepo, storage, queue, cache, proc, cnf)
	shareService := shareapp.NewService(authRepo, gifRepo, shareRepo)

	authH := authhandler.NewHandler(authService, middlewares, val)
	userH := userhandler.NewHandler(userService, middlewares, val)
	mediaH := mediahandler.NewHandler(mediaService, middlewares, val)
	shareH := sharehandler.NewHandler(shareService, middlewares, val)
	staticH := static.NewHandler()

	_ = rest.NewServer(authH, mediaH, shareH, userH, staticH)

	// 1. Signup User 1
	signupPayload, _ := json.Marshal(map[string]string{
		"username":         "alice",
		"fullname":         "Alice User",
		"email":            "alice@example.com",
		"password":         "Password123!",
		"confirm_password": "Password123!",
	})
	signupReq := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(signupPayload))
	signupRec := httptest.NewRecorder()
	authH.Signup(signupRec, signupReq)

	if signupRec.Code != http.StatusCreated {
		t.Fatalf("Signup failed: %d. Body: %s", signupRec.Code, signupRec.Body.String())
	}
	if len(queue.PublishedEmails) == 0 {
		t.Fatalf("Expected signup verification email")
	}

	// 2. Verify User 1 using Token
	emailMsg := queue.PublishedEmails[0]
	verifyReq := httptest.NewRequest(http.MethodGet, "/auth/verify?token="+emailMsg.Token, nil)
	verifyRec := httptest.NewRecorder()
	authH.Verify(verifyRec, verifyReq)

	if verifyRec.Code != http.StatusOK {
		t.Fatalf("Verification failed: %d", verifyRec.Code)
	}

	// 3. Login User 1
	loginPayload, _ := json.Marshal(map[string]string{
		"email":    "alice@example.com",
		"password": "Password123!",
	})
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginPayload))
	loginRec := httptest.NewRecorder()
	authH.Login(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Fatalf("Login failed: %d", loginRec.Code)
	}
	var loginResp map[string]string
	json.NewDecoder(loginRec.Body).Decode(&loginResp)
	token1 := loginResp["token"]
	if token1 == "" {
		t.Fatalf("Expected token in login response")
	}

	// 4. Query Profile
	profileReq := httptest.NewRequest(http.MethodGet, "/users/profile/me", nil)
	profileReq.Header.Set("Authorization", "Bearer "+token1)
	profileRec := httptest.NewRecorder()
	middlewares.Auth(http.HandlerFunc(userH.GetProfile)).ServeHTTP(profileRec, profileReq)

	if profileRec.Code != http.StatusOK {
		t.Fatalf("GetProfile failed: %d", profileRec.Code)
	}

	// 5. Query Quota
	quotaReq := httptest.NewRequest(http.MethodGet, "/users/me/quota", nil)
	quotaReq.Header.Set("Authorization", "Bearer "+token1)
	quotaRec := httptest.NewRecorder()
	middlewares.Auth(http.HandlerFunc(userH.GetQuota)).ServeHTTP(quotaRec, quotaReq)

	if quotaRec.Code != http.StatusOK {
		t.Fatalf("GetQuota failed: %d", quotaRec.Code)
	}

	// 6. Request Presigned Upload URL
	uploadPayload, _ := json.Marshal(map[string]string{"filename": "video.mp4"})
	uploadReq := httptest.NewRequest(http.MethodPost, "/uploads", bytes.NewReader(uploadPayload))
	uploadReq.Header.Set("Authorization", "Bearer "+token1)
	uploadRec := httptest.NewRecorder()
	middlewares.Auth(http.HandlerFunc(mediaH.Upload)).ServeHTTP(uploadRec, uploadReq)

	if uploadRec.Code != http.StatusCreated {
		t.Fatalf("Upload presign failed: %d", uploadRec.Code)
	}

	var uploadResp domainmedia.UploadResult
	json.NewDecoder(uploadRec.Body).Decode(&uploadResp)
	if uploadResp.Key == "" {
		t.Fatalf("Expected non-empty upload key")
	}

	// 7. Submit Convert Job
	convertPayload, _ := json.Marshal(map[string]any{
		"upload_key": uploadResp.Key,
		"start_time": 0.0,
		"end_time":   3.0,
		"fps":        15,
		"width":      480,
		"loop":       true,
	})
	convertReq := httptest.NewRequest(http.MethodPost, "/convert", bytes.NewReader(convertPayload))
	convertReq.Header.Set("Authorization", "Bearer "+token1)
	convertRec := httptest.NewRecorder()
	middlewares.Auth(http.HandlerFunc(mediaH.Convert)).ServeHTTP(convertRec, convertReq)

	if convertRec.Code != http.StatusOK {
		t.Fatalf("Convert request failed: %d", convertRec.Code)
	}

	var convertResult map[string]string
	json.NewDecoder(convertRec.Body).Decode(&convertResult)
	jobId := convertResult["job_id"]
	if jobId == "" {
		t.Fatalf("Expected job_id")
	}

	// 8. Simulate Background Worker Processing
	if len(queue.PublishedVideos) == 0 {
		t.Fatalf("Expected video job in queue")
	}
	videoMsg := queue.PublishedVideos[0]
	err := mediaService.Process(context.Background(), videoMsg)
	if err != nil {
		t.Fatalf("Background worker process failed: %v", err)
	}

	// 9. Query Conversion Status
	statusMux := http.NewServeMux()
	statusMux.Handle("GET /convert/{jobId}/status", middlewares.Auth(http.HandlerFunc(mediaH.ConversionStatus)))
	statusReq := httptest.NewRequest(http.MethodGet, "/convert/"+jobId+"/status", nil)
	statusReq.Header.Set("Authorization", "Bearer "+token1)
	statusRec := httptest.NewRecorder()
	statusMux.ServeHTTP(statusRec, statusReq)

	if statusRec.Code != http.StatusOK {
		t.Fatalf("ConversionStatus failed: %d", statusRec.Code)
	}
	var convStatus mediaapp.StatusResult
	json.NewDecoder(statusRec.Body).Decode(&convStatus)
	if convStatus.Status != "completed" || convStatus.GifId == "" {
		t.Fatalf("Expected completed status with GifId, got %+v", convStatus)
	}
	generatedGifKey := convStatus.GifId

	// 10. Signup User 2 (Bob)
	bobSignup, _ := json.Marshal(map[string]string{
		"username":         "bobuser",
		"fullname":         "Bob User",
		"email":            "bob@example.com",
		"password":         "Password123!",
		"confirm_password": "Password123!",
	})
	bobSignupReq := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(bobSignup))
	bobSignupRec := httptest.NewRecorder()
	authH.Signup(bobSignupRec, bobSignupReq)

	bobUser := authRepo.users["bob@example.com"]
	authRepo.SetVerified(context.Background(), bobUser.Id)

	bobLogin, _ := json.Marshal(map[string]string{"email": "bob@example.com", "password": "Password123!"})
	bobLoginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(bobLogin))
	bobLoginRec := httptest.NewRecorder()
	authH.Login(bobLoginRec, bobLoginReq)

	var bobLoginResp map[string]string
	json.NewDecoder(bobLoginRec.Body).Decode(&bobLoginResp)
	token2 := bobLoginResp["token"]

	// 11. Alice Shares GIF with Bob
	shareMux := http.NewServeMux()
	shareMux.Handle("POST /gifs/me/{key}/shares", middlewares.Auth(http.HandlerFunc(shareH.Create)))

	sharePayload, _ := json.Marshal(map[string]any{
		"shared_with": "bob@example.com",
		"expire_at":   time.Now().Add(24 * time.Hour),
	})
	shareReq := httptest.NewRequest(http.MethodPost, "/gifs/me/"+generatedGifKey+"/shares", bytes.NewReader(sharePayload))
	shareReq.Header.Set("Authorization", "Bearer "+token1)
	shareRec := httptest.NewRecorder()
	shareMux.ServeHTTP(shareRec, shareReq)

	if shareRec.Code != http.StatusCreated {
		t.Fatalf("Share GIF failed: %d", shareRec.Code)
	}

	// 12. Bob Downloads Shared GIF
	downloadMux := http.NewServeMux()
	downloadMux.Handle("GET /gifs/me/{key}/download", middlewares.Auth(http.HandlerFunc(mediaH.Download)))

	bobDownloadReq := httptest.NewRequest(http.MethodGet, "/gifs/me/"+generatedGifKey+"/download", nil)
	bobDownloadReq.Header.Set("Authorization", "Bearer "+token2)
	bobDownloadRec := httptest.NewRecorder()
	downloadMux.ServeHTTP(bobDownloadRec, bobDownloadReq)

	if bobDownloadRec.Code != http.StatusOK {
		t.Fatalf("Bob download shared GIF failed: %d", bobDownloadRec.Code)
	}

	// 13. Alice Deletes Share with Bob
	deleteShareMux := http.NewServeMux()
	deleteShareMux.Handle("DELETE /gifs/me/{key}/shares/{shareWithId}", middlewares.Auth(http.HandlerFunc(shareH.Delete)))

	deleteShareReq := httptest.NewRequest(http.MethodDelete, "/gifs/me/"+generatedGifKey+"/shares/"+bobUser.Id.String(), nil)
	deleteShareReq.Header.Set("Authorization", "Bearer "+token1)
	deleteShareRec := httptest.NewRecorder()
	deleteShareMux.ServeHTTP(deleteShareRec, deleteShareReq)

	if deleteShareRec.Code != http.StatusOK {
		t.Fatalf("Delete Share failed: %d", deleteShareRec.Code)
	}

	// 14. Alice Deletes GIF
	deleteMux := http.NewServeMux()
	deleteMux.Handle("DELETE /gifs/me/{key}", middlewares.Auth(http.HandlerFunc(mediaH.Delete)))

	deleteReq := httptest.NewRequest(http.MethodDelete, "/gifs/me/"+generatedGifKey, nil)
	deleteReq.Header.Set("Authorization", "Bearer "+token1)
	deleteRec := httptest.NewRecorder()
	deleteMux.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusOK {
		t.Fatalf("Delete GIF failed: %d", deleteRec.Code)
	}

	// 14. Change Password & Re-login
	changePassPayload, _ := json.Marshal(map[string]string{
		"current_password": "Password123!",
		"password":         "BrandNewPass123!",
		"confirm_password": "BrandNewPass123!",
	})
	changePassReq := httptest.NewRequest(http.MethodPatch, "/users/change-password", bytes.NewReader(changePassPayload))
	changePassReq.Header.Set("Authorization", "Bearer "+token1)
	changePassRec := httptest.NewRecorder()
	middlewares.Auth(http.HandlerFunc(userH.ChangePassword)).ServeHTTP(changePassRec, changePassReq)

	if changePassRec.Code != http.StatusOK {
		t.Fatalf("Change password failed: %d", changePassRec.Code)
	}

	// Re-login with new password
	reLoginPayload, _ := json.Marshal(map[string]string{
		"email":    "alice@example.com",
		"password": "BrandNewPass123!",
	})
	reLoginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(reLoginPayload))
	reLoginRec := httptest.NewRecorder()
	authH.Login(reLoginRec, reLoginReq)

	if reLoginRec.Code != http.StatusOK {
		t.Fatalf("Re-login with new password failed: %d", reLoginRec.Code)
	}
}

func TestE2E_UnauthorizedAndEdgeCases(t *testing.T) {
	jwtSecret := []byte("integration-test-secret-key-32bytes")
	jwtProvider := jwtpkg.NewJwt(jwtSecret)
	cache := newInMemoryCache()
	cnf := &config.Config{Addr: "127.0.0.1", Port: 8080, JwtSecret: jwtSecret}
	middlewares := middleware.NewMiddlewares(cnf, cache, *jwtProvider)
	val := validator.New()

	authRepo := &inMemoryAuthRepo{users: make(map[string]domainauth.User)}
	userService := userapp.NewService(&inMemoryProfileRepo{}, &inMemoryQuotaRepo{}, authRepo, *jwtProvider, *password.NewHasher("p", 10))
	userH := userhandler.NewHandler(userService, middlewares, val)

	// 1. Unauthenticated Request to Protected Route
	req := httptest.NewRequest(http.MethodGet, "/users/profile/me", nil)
	rec := httptest.NewRecorder()
	middlewares.Auth(http.HandlerFunc(userH.GetProfile)).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for request missing token, got %d", rec.Code)
	}

	// 2. Request with Invalid/Tampered Token
	req = httptest.NewRequest(http.MethodGet, "/users/profile/me", nil)
	req.Header.Set("Authorization", "Bearer invalid.fake.token")
	rec = httptest.NewRecorder()
	middlewares.Auth(http.HandlerFunc(userH.GetProfile)).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for tampered token, got %d", rec.Code)
	}

	// 3. Request with Blocklisted Token
	validToken, _ := jwtProvider.Create("Blocked User", "blocked-id", "block@example.com", "user")
	cache.Set(context.Background(), "token_blocklist:"+validToken, "1", time.Hour)

	req = httptest.NewRequest(http.MethodGet, "/users/profile/me", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	rec = httptest.NewRecorder()
	middlewares.Auth(http.HandlerFunc(userH.GetProfile)).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for blocklisted token, got %d", rec.Code)
	}
}
