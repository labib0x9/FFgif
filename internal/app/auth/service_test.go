package auth_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	appauth "github.com/labib0x9/ffgif/internal/app/auth"
	domainauth "github.com/labib0x9/ffgif/internal/domain/auth"
	domainuser "github.com/labib0x9/ffgif/internal/domain/user"
	"github.com/labib0x9/ffgif/internal/port/queue"
	"github.com/labib0x9/ffgif/pkg/apperr"
	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
	"github.com/labib0x9/ffgif/pkg/password"
	tokenpkg "github.com/labib0x9/ffgif/pkg/token"
	amqp "github.com/rabbitmq/amqp091-go"
)

// --- Mocks ---

type mockTxManager struct {
	withFunc func(ctx context.Context, fn func(ctx context.Context) (any, error)) (any, error)
}

func (m *mockTxManager) With(ctx context.Context, fn func(ctx context.Context) (any, error)) (any, error) {
	if m.withFunc != nil {
		return m.withFunc(ctx, fn)
	}
	return fn(ctx)
}

type mockAuthRepo struct {
	getByEmailFunc     func(ctx context.Context, email string) (domainauth.User, error)
	getByIdFunc        func(ctx context.Context, id uuid.UUID) (domainauth.User, error)
	createFunc         func(ctx context.Context, user domainauth.User) (domainauth.User, error)
	deleteByIdFunc     func(ctx context.Context, id uuid.UUID) error
	deleteByEmailFunc  func(ctx context.Context, email string) error
	updatePasswordFunc func(ctx context.Context, id uuid.UUID, passHash string) error
	setVerifiedFunc    func(ctx context.Context, userId uuid.UUID) error
	upgradeFunc        func(ctx context.Context, id string, user domainauth.User) (domainauth.User, error)
}

func (m *mockAuthRepo) GetByEmail(ctx context.Context, email string) (domainauth.User, error) {
	if m.getByEmailFunc != nil {
		return m.getByEmailFunc(ctx, email)
	}
	return domainauth.User{}, errors.New("not implemented")
}
func (m *mockAuthRepo) GetById(ctx context.Context, id uuid.UUID) (domainauth.User, error) {
	if m.getByIdFunc != nil {
		return m.getByIdFunc(ctx, id)
	}
	return domainauth.User{}, errors.New("not implemented")
}
func (m *mockAuthRepo) Create(ctx context.Context, user domainauth.User) (domainauth.User, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, user)
	}
	user.Id = uuid.New()
	return user, nil
}
func (m *mockAuthRepo) DeleteById(ctx context.Context, id uuid.UUID) error {
	if m.deleteByIdFunc != nil {
		return m.deleteByIdFunc(ctx, id)
	}
	return nil
}
func (m *mockAuthRepo) DeleteByEmail(ctx context.Context, email string) error {
	if m.deleteByEmailFunc != nil {
		return m.deleteByEmailFunc(ctx, email)
	}
	return nil
}
func (m *mockAuthRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passHash string) error {
	if m.updatePasswordFunc != nil {
		return m.updatePasswordFunc(ctx, id, passHash)
	}
	return nil
}
func (m *mockAuthRepo) SetVerified(ctx context.Context, userId uuid.UUID) error {
	if m.setVerifiedFunc != nil {
		return m.setVerifiedFunc(ctx, userId)
	}
	return nil
}
func (m *mockAuthRepo) Upgrade(ctx context.Context, id string, user domainauth.User) (domainauth.User, error) {
	if m.upgradeFunc != nil {
		return m.upgradeFunc(ctx, id, user)
	}
	return user, nil
}

type mockVerifierRepo struct {
	createFunc    func(ctx context.Context, verifier domainauth.Verifier) error
	getByHashFunc func(ctx context.Context, tokenHash string) (domainauth.Verifier, error)
	getByIdFunc   func(ctx context.Context, userId uuid.UUID) (domainauth.Verifier, error)
	deleteFunc    func(ctx context.Context, id int64) error
}

func (m *mockVerifierRepo) Create(ctx context.Context, verifier domainauth.Verifier) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, verifier)
	}
	return nil
}
func (m *mockVerifierRepo) GetByHash(ctx context.Context, tokenHash string) (domainauth.Verifier, error) {
	if m.getByHashFunc != nil {
		return m.getByHashFunc(ctx, tokenHash)
	}
	return domainauth.Verifier{}, errors.New("not found")
}
func (m *mockVerifierRepo) GetById(ctx context.Context, userId uuid.UUID) (domainauth.Verifier, error) {
	if m.getByIdFunc != nil {
		return m.getByIdFunc(ctx, userId)
	}
	return domainauth.Verifier{}, errors.New("not found")
}
func (m *mockVerifierRepo) Delete(ctx context.Context, id int64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

type mockProfileRepo struct {
	getProfileFunc    func(ctx context.Context, id string) (domainuser.ProfileResponse, error)
	updateProfileFunc func(ctx context.Context, profile domainuser.ProfileResponse, id string) (domainuser.ProfileResponse, error)
	setProfileFunc    func(ctx context.Context, profile domainuser.Profile) error
}

func (m *mockProfileRepo) GetProfile(ctx context.Context, id string) (domainuser.ProfileResponse, error) {
	if m.getProfileFunc != nil {
		return m.getProfileFunc(ctx, id)
	}
	return domainuser.ProfileResponse{}, nil
}
func (m *mockProfileRepo) UpdateProfile(ctx context.Context, profile domainuser.ProfileResponse, id string) (domainuser.ProfileResponse, error) {
	if m.updateProfileFunc != nil {
		return m.updateProfileFunc(ctx, profile, id)
	}
	return profile, nil
}
func (m *mockProfileRepo) SetProfile(ctx context.Context, profile domainuser.Profile) error {
	if m.setProfileFunc != nil {
		return m.setProfileFunc(ctx, profile)
	}
	return nil
}
func (m *mockProfileRepo) ChangePassword(ctx context.Context, userId string, hash string) error {
	return nil
}

type mockReseterRepo struct {
	createFunc     func(ctx context.Context, reseter domainauth.Reseter) error
	getByTokenFunc func(ctx context.Context, token string) (domainauth.Reseter, error)
	getByIdFunc    func(ctx context.Context, id uuid.UUID) (domainauth.Reseter, error)
	deleteByIdFunc func(ctx context.Context, id int64) error
	updateFunc     func(ctx context.Context, reseter domainauth.Reseter) error
}

func (m *mockReseterRepo) Create(ctx context.Context, reseter domainauth.Reseter) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, reseter)
	}
	return nil
}
func (m *mockReseterRepo) GetByToken(ctx context.Context, token string) (domainauth.Reseter, error) {
	if m.getByTokenFunc != nil {
		return m.getByTokenFunc(ctx, token)
	}
	return domainauth.Reseter{}, errors.New("not found")
}
func (m *mockReseterRepo) GetById(ctx context.Context, id uuid.UUID) (domainauth.Reseter, error) {
	if m.getByIdFunc != nil {
		return m.getByIdFunc(ctx, id)
	}
	return domainauth.Reseter{}, errors.New("not found")
}
func (m *mockReseterRepo) DeleteById(ctx context.Context, id int64) error {
	if m.deleteByIdFunc != nil {
		return m.deleteByIdFunc(ctx, id)
	}
	return nil
}
func (m *mockReseterRepo) Update(ctx context.Context, reseter domainauth.Reseter) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, reseter)
	}
	return nil
}

type mockQuotaRepo struct {
	createFunc  func(ctx context.Context, quota domainuser.Quota) error
	getByIdFunc func(ctx context.Context, id string) (*domainuser.Quota, error)
}

func (m *mockQuotaRepo) Create(ctx context.Context, quota domainuser.Quota) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, quota)
	}
	return nil
}
func (m *mockQuotaRepo) GetById(ctx context.Context, id string) (*domainuser.Quota, error) {
	if m.getByIdFunc != nil {
		return m.getByIdFunc(ctx, id)
	}
	return &domainuser.Quota{}, nil
}

type mockCache struct {
	setFunc func(ctx context.Context, key string, value string, expiration time.Duration) error
	getFunc func(ctx context.Context, key string) (string, error)
}

func (m *mockCache) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	if m.setFunc != nil {
		return m.setFunc(ctx, key, value, expiration)
	}
	return nil
}
func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, key)
	}
	return "", nil
}

type mockQueue struct {
	publishEmailFunc func(ctx context.Context, msg queue.EmailMessage) error
	publishVideoFunc func(ctx context.Context, msg queue.VideoMessage) error
}

func (m *mockQueue) PublishEmail(ctx context.Context, msg queue.EmailMessage) error {
	if m.publishEmailFunc != nil {
		return m.publishEmailFunc(ctx, msg)
	}
	return nil
}
func (m *mockQueue) PublishVideo(ctx context.Context, msg queue.VideoMessage) error {
	if m.publishVideoFunc != nil {
		return m.publishVideoFunc(ctx, msg)
	}
	return nil
}
func (m *mockQueue) PublishSaveVideo(ctx context.Context, msg queue.SaveVideoMessage) error      { return nil }
func (m *mockQueue) PublishRetrySaveVideo(ctx context.Context, msg queue.SaveVideoMessage) error { return nil }
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

func newTestAuthService(
	authRepo domainauth.AuthRepository,
	verifierRepo domainauth.VerifierRepository,
	profileRepo domainuser.UserRepository,
	reseterRepo domainauth.ReseterRepository,
	quotaRepo domainuser.QuotaRepository,
	cache *mockCache,
	queue *mockQueue,
	txManager *mockTxManager,
) appauth.Service {
	if cache == nil {
		cache = &mockCache{}
	}
	if queue == nil {
		queue = &mockQueue{}
	}
	if txManager == nil {
		txManager = &mockTxManager{}
	}
	jwtProvider := jwtpkg.NewJwt([]byte("test-auth-secret-12345"))
	hasher := password.NewHasher("test-pepper", 10)
	return appauth.NewService(
		authRepo, verifierRepo, profileRepo, reseterRepo, quotaRepo,
		cache, queue, *jwtProvider, *hasher, txManager,
	)
}

// --- Tests ---

func TestAuthService_Signup_Success(t *testing.T) {
	authRepo := &mockAuthRepo{
		getByEmailFunc: func(ctx context.Context, email string) (domainauth.User, error) {
			return domainauth.User{}, errors.New("not found")
		},
		createFunc: func(ctx context.Context, user domainauth.User) (domainauth.User, error) {
			user.Id = uuid.New()
			return user, nil
		},
	}
	emailPublished := false
	queueMock := &mockQueue{
		publishEmailFunc: func(ctx context.Context, msg queue.EmailMessage) error {
			emailPublished = true
			if msg.To != "user@example.com" {
				t.Errorf("expected email to user@example.com, got %s", msg.To)
			}
			return nil
		},
	}

	svc := newTestAuthService(authRepo, &mockVerifierRepo{}, &mockProfileRepo{}, &mockReseterRepo{}, &mockQuotaRepo{}, nil, queueMock, nil)

	_, err := svc.Signup(context.Background(), "user@example.com", "username", "Full Name", "password123")
	if err != nil {
		t.Fatalf("expected signup to succeed, got error: %v", err)
	}

	if !emailPublished {
		t.Errorf("expected verification email to be published to queue")
	}
}

func TestAuthService_Signup_UserAlreadyExists(t *testing.T) {
	authRepo := &mockAuthRepo{
		getByEmailFunc: func(ctx context.Context, email string) (domainauth.User, error) {
			return domainauth.User{Email: email}, nil
		},
	}
	svc := newTestAuthService(authRepo, &mockVerifierRepo{}, &mockProfileRepo{}, &mockReseterRepo{}, &mockQuotaRepo{}, nil, nil, nil)

	_, err := svc.Signup(context.Background(), "exists@example.com", "u", "fn", "p")
	if !errors.Is(err, domainauth.ErrUserExists) {
		t.Errorf("expected ErrUserExists, got %v", err)
	}
}

func TestAuthService_Signup_QueuePublishFailure(t *testing.T) {
	authRepo := &mockAuthRepo{
		getByEmailFunc: func(ctx context.Context, email string) (domainauth.User, error) {
			return domainauth.User{}, errors.New("not found")
		},
		createFunc: func(ctx context.Context, user domainauth.User) (domainauth.User, error) {
			user.Id = uuid.New()
			return user, nil
		},
	}
	queueMock := &mockQueue{
		publishEmailFunc: func(ctx context.Context, msg queue.EmailMessage) error {
			return errors.New("rabbitmq down")
		},
	}
	svc := newTestAuthService(authRepo, &mockVerifierRepo{}, &mockProfileRepo{}, &mockReseterRepo{}, &mockQuotaRepo{}, nil, queueMock, nil)

	_, err := svc.Signup(context.Background(), "user@example.com", "username", "Full Name", "password123")
	if !errors.Is(err, apperr.ErrMessageQueueFailed) {
		t.Errorf("expected ErrMessageQueueFailed, got %v", err)
	}
}

func TestAuthService_Login_Success(t *testing.T) {
	hasher := password.NewHasher("test-pepper", 10)
	hashedPass, _ := hasher.GenerateHash("password123")
	userId := uuid.New()

	authRepo := &mockAuthRepo{
		getByEmailFunc: func(ctx context.Context, email string) (domainauth.User, error) {
			return domainauth.User{
				Id:           userId,
				Email:        email,
				Fullname:     "Test User",
				Role:         "user",
				PasswordHash: hashedPass,
				IsVerified:   true,
			}, nil
		},
	}

	svc := newTestAuthService(authRepo, &mockVerifierRepo{}, &mockProfileRepo{}, &mockReseterRepo{}, &mockQuotaRepo{}, nil, nil, nil)

	result, err := svc.Login(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("expected login to succeed, got: %v", err)
	}
	if result.Token == "" {
		t.Errorf("expected token in login result, got empty")
	}
	if result.Id != userId {
		t.Errorf("expected userId %s, got %s", userId, result.Id)
	}
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	authRepo := &mockAuthRepo{
		getByEmailFunc: func(ctx context.Context, email string) (domainauth.User, error) {
			return domainauth.User{}, sql.ErrNoRows
		},
	}
	svc := newTestAuthService(authRepo, &mockVerifierRepo{}, &mockProfileRepo{}, &mockReseterRepo{}, &mockQuotaRepo{}, nil, nil, nil)

	_, err := svc.Login(context.Background(), "missing@example.com", "password123")
	if !errors.Is(err, domainauth.ErrInvalidCredential) {
		t.Errorf("expected ErrInvalidCredential, got %v", err)
	}
}

func TestAuthService_Login_Unverified(t *testing.T) {
	hasher := password.NewHasher("test-pepper", 10)
	hashedPass, _ := hasher.GenerateHash("password123")

	authRepo := &mockAuthRepo{
		getByEmailFunc: func(ctx context.Context, email string) (domainauth.User, error) {
			return domainauth.User{
				Email:        email,
				PasswordHash: hashedPass,
				IsVerified:   false,
			}, nil
		},
	}
	svc := newTestAuthService(authRepo, &mockVerifierRepo{}, &mockProfileRepo{}, &mockReseterRepo{}, &mockQuotaRepo{}, nil, nil, nil)

	_, err := svc.Login(context.Background(), "unverified@example.com", "password123")
	if !errors.Is(err, domainauth.ErrUserNotVerified) {
		t.Errorf("expected ErrUserNotVerified, got %v", err)
	}
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	hasher := password.NewHasher("test-pepper", 10)
	hashedPass, _ := hasher.GenerateHash("correctPass")

	authRepo := &mockAuthRepo{
		getByEmailFunc: func(ctx context.Context, email string) (domainauth.User, error) {
			return domainauth.User{
				Email:        email,
				PasswordHash: hashedPass,
				IsVerified:   true,
			}, nil
		},
	}
	svc := newTestAuthService(authRepo, &mockVerifierRepo{}, &mockProfileRepo{}, &mockReseterRepo{}, &mockQuotaRepo{}, nil, nil, nil)

	_, err := svc.Login(context.Background(), "test@example.com", "wrongPass")
	if !errors.Is(err, domainauth.ErrInvalidCredential) {
		t.Errorf("expected ErrInvalidCredential, got %v", err)
	}
}

func TestAuthService_Logout_Success(t *testing.T) {
	cachedKey := ""
	cacheMock := &mockCache{
		setFunc: func(ctx context.Context, key string, value string, expiration time.Duration) error {
			cachedKey = key
			return nil
		},
	}
	svc := newTestAuthService(&mockAuthRepo{}, &mockVerifierRepo{}, &mockProfileRepo{}, &mockReseterRepo{}, &mockQuotaRepo{}, cacheMock, nil, nil)

	claims := jwtpkg.Payload{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}

	err := svc.Logout(context.Background(), "my-jwt-token", claims)
	if err != nil {
		t.Fatalf("expected Logout to succeed, got %v", err)
	}
	if cachedKey != "token_blocklist:my-jwt-token" {
		t.Errorf("expected blocklist key 'token_blocklist:my-jwt-token', got %s", cachedKey)
	}
}

func TestAuthService_Logout_ExpiredToken(t *testing.T) {
	cacheMock := &mockCache{
		setFunc: func(ctx context.Context, key string, value string, expiration time.Duration) error {
			t.Errorf("expected no cache write for already-expired token")
			return nil
		},
	}
	svc := newTestAuthService(&mockAuthRepo{}, &mockVerifierRepo{}, &mockProfileRepo{}, &mockReseterRepo{}, &mockQuotaRepo{}, cacheMock, nil, nil)

	claims := jwtpkg.Payload{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	}

	err := svc.Logout(context.Background(), "expired-jwt", claims)
	if err != nil {
		t.Fatalf("expected Logout to succeed with nil error, got %v", err)
	}
}

func TestAuthService_Verify_Success(t *testing.T) {
	userId := uuid.New()
	verifiedUser := false
	deletedVerifier := false

	tokenStr, _ := tokenpkg.GenerateToken()
	tokenHash := tokenpkg.GetTokenHash(tokenStr)

	verifierRepo := &mockVerifierRepo{
		getByHashFunc: func(ctx context.Context, hash string) (domainauth.Verifier, error) {
			if hash == tokenHash {
				return domainauth.Verifier{Id: 10, UserId: userId}, nil
			}
			return domainauth.Verifier{}, sql.ErrNoRows
		},
		deleteFunc: func(ctx context.Context, id int64) error {
			deletedVerifier = true
			return nil
		},
	}
	authRepo := &mockAuthRepo{
		setVerifiedFunc: func(ctx context.Context, id uuid.UUID) error {
			if id == userId {
				verifiedUser = true
			}
			return nil
		},
	}

	svc := newTestAuthService(authRepo, verifierRepo, &mockProfileRepo{}, &mockReseterRepo{}, &mockQuotaRepo{}, nil, nil, nil)

	err := svc.Verify(context.Background(), tokenStr)
	if err != nil {
		t.Fatalf("expected Verify to succeed, got %v", err)
	}
	if !verifiedUser || !deletedVerifier {
		t.Errorf("expected user to be verified and verifier record deleted")
	}
}

func TestAuthService_Verify_InvalidToken(t *testing.T) {
	verifierRepo := &mockVerifierRepo{
		getByHashFunc: func(ctx context.Context, hash string) (domainauth.Verifier, error) {
			return domainauth.Verifier{}, sql.ErrNoRows
		},
	}
	svc := newTestAuthService(&mockAuthRepo{}, verifierRepo, &mockProfileRepo{}, &mockReseterRepo{}, &mockQuotaRepo{}, nil, nil, nil)

	err := svc.Verify(context.Background(), "invalid-token")
	if !errors.Is(err, domainauth.ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestAuthService_ForgotPassword_Success(t *testing.T) {
	userId := uuid.New()
	authRepo := &mockAuthRepo{
		getByEmailFunc: func(ctx context.Context, email string) (domainauth.User, error) {
			return domainauth.User{Id: userId, Email: email, IsVerified: true}, nil
		},
	}
	reseterRepo := &mockReseterRepo{
		getByIdFunc: func(ctx context.Context, id uuid.UUID) (domainauth.Reseter, error) {
			return domainauth.Reseter{}, sql.ErrNoRows
		},
		createFunc: func(ctx context.Context, reseter domainauth.Reseter) error {
			return nil
		},
	}
	emailSent := false
	queueMock := &mockQueue{
		publishEmailFunc: func(ctx context.Context, msg queue.EmailMessage) error {
			emailSent = true
			return nil
		},
	}

	svc := newTestAuthService(authRepo, &mockVerifierRepo{}, &mockProfileRepo{}, reseterRepo, &mockQuotaRepo{}, nil, queueMock, nil)

	err := svc.ForgotPassword(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("expected ForgotPassword to succeed, got %v", err)
	}
	if !emailSent {
		t.Errorf("expected forgot-password email to be published")
	}
}

func TestAuthService_ForgotPassword_UserNotFound(t *testing.T) {
	authRepo := &mockAuthRepo{
		getByEmailFunc: func(ctx context.Context, email string) (domainauth.User, error) {
			return domainauth.User{}, sql.ErrNoRows
		},
	}
	svc := newTestAuthService(authRepo, &mockVerifierRepo{}, &mockProfileRepo{}, &mockReseterRepo{}, &mockQuotaRepo{}, nil, nil, nil)

	err := svc.ForgotPassword(context.Background(), "unknown@example.com")
	if !errors.Is(err, domainauth.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestAuthService_ForgotPassword_Unverified(t *testing.T) {
	authRepo := &mockAuthRepo{
		getByEmailFunc: func(ctx context.Context, email string) (domainauth.User, error) {
			return domainauth.User{Email: email, IsVerified: false}, nil
		},
	}
	svc := newTestAuthService(authRepo, &mockVerifierRepo{}, &mockProfileRepo{}, &mockReseterRepo{}, &mockQuotaRepo{}, nil, nil, nil)

	err := svc.ForgotPassword(context.Background(), "unverified@example.com")
	if !errors.Is(err, domainauth.ErrUserNotVerified) {
		t.Errorf("expected ErrUserNotVerified, got %v", err)
	}
}

func TestAuthService_ResendVerify_Success(t *testing.T) {
	userId := uuid.New()
	authRepo := &mockAuthRepo{
		getByEmailFunc: func(ctx context.Context, email string) (domainauth.User, error) {
			return domainauth.User{Id: userId, Email: email, IsVerified: false}, nil
		},
	}
	verifierRepo := &mockVerifierRepo{
		getByIdFunc: func(ctx context.Context, id uuid.UUID) (domainauth.Verifier, error) {
			return domainauth.Verifier{Id: 1}, nil
		},
		deleteFunc: func(ctx context.Context, id int64) error {
			return nil
		},
		createFunc: func(ctx context.Context, v domainauth.Verifier) error {
			return nil
		},
	}
	emailSent := false
	queueMock := &mockQueue{
		publishEmailFunc: func(ctx context.Context, msg queue.EmailMessage) error {
			emailSent = true
			return nil
		},
	}

	svc := newTestAuthService(authRepo, verifierRepo, &mockProfileRepo{}, &mockReseterRepo{}, &mockQuotaRepo{}, nil, queueMock, nil)

	err := svc.ResendVerify(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("expected ResendVerify to succeed, got %v", err)
	}
	if !emailSent {
		t.Errorf("expected resend-verify email to be published")
	}
}

func TestAuthService_ResendVerify_AlreadyVerified(t *testing.T) {
	authRepo := &mockAuthRepo{
		getByEmailFunc: func(ctx context.Context, email string) (domainauth.User, error) {
			return domainauth.User{Email: email, IsVerified: true}, nil
		},
	}
	svc := newTestAuthService(authRepo, &mockVerifierRepo{}, &mockProfileRepo{}, &mockReseterRepo{}, &mockQuotaRepo{}, nil, nil, nil)

	err := svc.ResendVerify(context.Background(), "user@example.com")
	if !errors.Is(err, domainauth.ErrUserAlreadyVerified) {
		t.Errorf("expected ErrUserAlreadyVerified, got %v", err)
	}
}

func TestAuthService_ResetPasswordGet_Success(t *testing.T) {
	reseterRepo := &mockReseterRepo{
		getByTokenFunc: func(ctx context.Context, token string) (domainauth.Reseter, error) {
			return domainauth.Reseter{Token: "valid-reset-token"}, nil
		},
	}
	svc := newTestAuthService(&mockAuthRepo{}, &mockVerifierRepo{}, &mockProfileRepo{}, reseterRepo, &mockQuotaRepo{}, nil, nil, nil)

	token, err := svc.ResetPasswordGet(context.Background(), "valid-reset-token")
	if err != nil {
		t.Fatalf("expected ResetPasswordGet to succeed, got %v", err)
	}
	if token != "valid-reset-token" {
		t.Errorf("expected valid-reset-token, got %s", token)
	}
}

func TestAuthService_ResetPasswordGet_NotFound(t *testing.T) {
	reseterRepo := &mockReseterRepo{
		getByTokenFunc: func(ctx context.Context, token string) (domainauth.Reseter, error) {
			return domainauth.Reseter{}, errors.New("not found")
		},
	}
	svc := newTestAuthService(&mockAuthRepo{}, &mockVerifierRepo{}, &mockProfileRepo{}, reseterRepo, &mockQuotaRepo{}, nil, nil, nil)

	_, err := svc.ResetPasswordGet(context.Background(), "invalid-token")
	if !errors.Is(err, domainauth.ErrReseterTokenFatchFailed) {
		t.Errorf("expected ErrReseterTokenFatchFailed, got %v", err)
	}
}

func TestAuthService_ResetPasswordPost_Success(t *testing.T) {
	userId := uuid.New()
	reseterRepo := &mockReseterRepo{
		getByTokenFunc: func(ctx context.Context, token string) (domainauth.Reseter, error) {
			return domainauth.Reseter{Id: 5, UserId: userId, Token: token}, nil
		},
		deleteByIdFunc: func(ctx context.Context, id int64) error {
			return nil
		},
	}
	authRepo := &mockAuthRepo{
		getByIdFunc: func(ctx context.Context, id uuid.UUID) (domainauth.User, error) {
			return domainauth.User{Id: userId, Email: "user@example.com"}, nil
		},
		updatePasswordFunc: func(ctx context.Context, id uuid.UUID, passHash string) error {
			return nil
		},
	}
	emailSent := false
	queueMock := &mockQueue{
		publishEmailFunc: func(ctx context.Context, msg queue.EmailMessage) error {
			emailSent = true
			return nil
		},
	}

	svc := newTestAuthService(authRepo, &mockVerifierRepo{}, &mockProfileRepo{}, reseterRepo, &mockQuotaRepo{}, nil, queueMock, nil)

	err := svc.ResetPasswordPost(context.Background(), "valid-token", "NewPassword123!", "NewPassword123!")
	if err != nil {
		t.Fatalf("expected ResetPasswordPost to succeed, got %v", err)
	}
	if !emailSent {
		t.Errorf("expected reset-password confirmation email")
	}
}
