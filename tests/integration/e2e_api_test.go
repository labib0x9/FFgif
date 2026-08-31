package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	domainuser "github.com/labib0x9/ffgif/internal/domain/user"
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

// --- In-Memory Integration Test Fixture ---

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
func (r *inMemoryAuthRepo) DeleteById(ctx context.Context, id uuid.UUID) error    { return nil }
func (r *inMemoryAuthRepo) DeleteByEmail(ctx context.Context, email string) error { return nil }
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
	return domainauth.Verifier{}, nil
}
func (r *inMemoryVerifierRepo) Delete(ctx context.Context, id int64) error { return nil }

type inMemoryProfileRepo struct {
	profiles map[string]domainuser.ProfileResponse
}

func (r *inMemoryProfileRepo) GetProfile(ctx context.Context, id string) (domainuser.ProfileResponse, error) {
	if p, ok := r.profiles[id]; ok {
		return p, nil
	}
	return domainuser.ProfileResponse{Username: "default"}, nil
}
func (r *inMemoryProfileRepo) UpdateProfile(ctx context.Context, profile domainuser.ProfileResponse, id string) (domainuser.ProfileResponse, error) {
	r.profiles[id] = profile
	return profile, nil
}
func (r *inMemoryProfileRepo) SetProfile(ctx context.Context, profile domainuser.Profile) error {
	return nil
}
func (r *inMemoryProfileRepo) ChangePassword(ctx context.Context, userId string, hash string) error {
	return nil
}

type inMemoryQuotaRepo struct{}

func (r *inMemoryQuotaRepo) Create(ctx context.Context, quota domainuser.Quota) error { return nil }
func (r *inMemoryQuotaRepo) GetById(ctx context.Context, userId string) (*domainuser.Quota, error) {
	return &domainuser.Quota{TotalBytes: 1024 * 1024 * 100, GifCount: 20}, nil
}

type inMemoryTx struct{}

func (t *inMemoryTx) With(ctx context.Context, fn func(ctx context.Context) (any, error)) (any, error) {
	return fn(ctx)
}

// --- End-to-End Integration Test ---

func TestE2E_FullUserAndJobLifecycle(t *testing.T) {
	jwtSecret := []byte("integration-test-secret-key-32bytes")
	jwtProvider := jwtpkg.NewJwt(jwtSecret)
	hasher := password.NewHasher("test-pepper", 10)
	cache := newInMemoryCache()
	queue := &inMemoryQueue{}
	authRepo := &inMemoryAuthRepo{users: make(map[string]domainauth.User)}
	verifierRepo := &inMemoryVerifierRepo{tokens: make(map[string]domainauth.Verifier)}
	profileRepo := &inMemoryProfileRepo{profiles: make(map[string]domainuser.ProfileResponse)}
	quotaRepo := &inMemoryQuotaRepo{}
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
	mediaService := mediaapp.NewService(authRepo, profileRepo, quotaRepo, nil, nil, nil, queue, cache, nil, cnf)
	shareService := shareapp.NewService()

	authH := authhandler.NewHandler(authService, middlewares, val)
	userH := userhandler.NewHandler(userService, middlewares, val)
	mediaH := mediahandler.NewHandler(mediaService, middlewares, val)
	shareH := sharehandler.NewHandler(shareService, middlewares, val)
	staticH := static.NewHandler()

	_ = rest.NewServer(authH, mediaH, shareH, userH, staticH)

	// 1. Signup Flow
	signupPayload, _ := json.Marshal(map[string]string{
		"username":         "integrationuser",
		"fullname":         "Integration Tester",
		"email":            "test@integration.local",
		"password":         "ComplexPass123!",
		"confirm_password": "ComplexPass123!",
	})
	signupReq := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(signupPayload))
	signupRec := httptest.NewRecorder()
	authH.Signup(signupRec, signupReq)

	if signupRec.Code != http.StatusCreated {
		t.Fatalf("Signup failed with status: %d. Body: %s", signupRec.Code, signupRec.Body.String())
	}

	if len(queue.PublishedEmails) == 0 {
		t.Fatalf("Expected signup verification email to be published to queue")
	}

	// 2. Mark User Verified (Simulate verification)
	u := authRepo.users["test@integration.local"]
	authRepo.SetVerified(context.Background(), u.Id)

	// 3. Login Flow
	loginPayload, _ := json.Marshal(map[string]string{
		"email":    "test@integration.local",
		"password": "ComplexPass123!",
	})
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginPayload))
	loginRec := httptest.NewRecorder()
	authH.Login(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Fatalf("Login failed with status: %d. Body: %s", loginRec.Code, loginRec.Body.String())
	}

	var loginResp map[string]string
	json.NewDecoder(loginRec.Body).Decode(&loginResp)
	token := loginResp["token"]
	if token == "" {
		t.Fatalf("Expected JWT token in login response")
	}

	// 4. Authenticated Request: Get Profile
	profileReq := httptest.NewRequest(http.MethodGet, "/users/profile/me", nil)
	profileReq.Header.Set("Authorization", "Bearer "+token)
	profileRec := httptest.NewRecorder()

	middlewares.Auth(http.HandlerFunc(userH.GetProfile)).ServeHTTP(profileRec, profileReq)

	if profileRec.Code != http.StatusOK {
		t.Fatalf("GetProfile failed with status: %d. Body: %s", profileRec.Code, profileRec.Body.String())
	}

	// 5. Authenticated Request: Convert Video Job
	convertPayload, _ := json.Marshal(map[string]any{
		"upload_key": "raw_video_uuid.mp4",
		"start_time": 0.0,
		"end_time":   4.0,
		"fps":        15,
		"width":      480,
		"loop":       true,
	})
	convertReq := httptest.NewRequest(http.MethodPost, "/convert", bytes.NewReader(convertPayload))
	convertReq.Header.Set("Authorization", "Bearer "+token)
	convertRec := httptest.NewRecorder()

	// middlewares.Auth(http.HandlerFunc(jobH.Convert)).ServeHTTP(convertRec, convertReq)

	if convertRec.Code != http.StatusOK {
		t.Fatalf("Convert request failed with status: %d. Body: %s", convertRec.Code, convertRec.Body.String())
	}

	if len(queue.PublishedVideos) == 0 {
		t.Fatalf("Expected video conversion job to be published to RabbitMQ queue")
	}

	jobMsg := queue.PublishedVideos[0]
	if jobMsg.Key != "raw_video_uuid.mp4" {
		t.Errorf("Expected job key raw_video_uuid.mp4, got %s", jobMsg.Key)
	}
}
