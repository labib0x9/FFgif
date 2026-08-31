package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	appauth "github.com/labib0x9/ffgif/internal/app/auth"
	domainauth "github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/port/queue"
	domainuser "github.com/labib0x9/ffgif/internal/domain/user"
	"github.com/labib0x9/ffgif/pkg/jwt"
	"github.com/labib0x9/ffgif/pkg/password"
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
	verifierRepo := &mockVerifierRepo{}
	profileRepo := &mockProfileRepo{}
	reseterRepo := &mockReseterRepo{}
	quotaRepo := &mockQuotaRepo{}
	cache := &mockCache{}
	emailPublished := false
	queue := &mockQueue{
		publishEmailFunc: func(ctx context.Context, msg queue.EmailMessage) error {
			emailPublished = true
			if msg.To != "user@example.com" {
				t.Errorf("expected email to user@example.com, got %s", msg.To)
			}
			return nil
		},
	}
	jwtProvider := jwt.NewJwt([]byte("secret"))
	hasher := password.NewHasher("pepper", 10)
	txManager := &mockTxManager{}

	svc := appauth.NewService(
		authRepo, verifierRepo, profileRepo, reseterRepo, quotaRepo,
		cache, queue, *jwtProvider, *hasher, txManager,
	)

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
	svc := appauth.NewService(
		authRepo, &mockVerifierRepo{}, &mockProfileRepo{}, &mockReseterRepo{}, &mockQuotaRepo{},
		&mockCache{}, &mockQueue{}, *jwt.NewJwt([]byte("s")), *password.NewHasher("p", 10), &mockTxManager{},
	)

	_, err := svc.Signup(context.Background(), "exists@example.com", "u", "fn", "p")
	if !errors.Is(err, domainauth.ErrUserExits) {
		t.Errorf("expected ErrUserExits, got %v", err)
	}
}

func TestAuthService_Login_Success(t *testing.T) {
	hasher := password.NewHasher("pepper", 10)
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
	jwtProvider := jwt.NewJwt([]byte("secret"))

	svc := appauth.NewService(
		authRepo, &mockVerifierRepo{}, &mockProfileRepo{}, &mockReseterRepo{}, &mockQuotaRepo{},
		&mockCache{}, &mockQueue{}, *jwtProvider, *hasher, &mockTxManager{},
	)

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

func TestAuthService_Login_Unverified(t *testing.T) {
	hasher := password.NewHasher("pepper", 10)
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

	svc := appauth.NewService(
		authRepo, &mockVerifierRepo{}, &mockProfileRepo{}, &mockReseterRepo{}, &mockQuotaRepo{},
		&mockCache{}, &mockQueue{}, *jwt.NewJwt([]byte("s")), *hasher, &mockTxManager{},
	)

	_, err := svc.Login(context.Background(), "unverified@example.com", "password123")
	if !errors.Is(err, domainauth.ErrUserNotVerified) {
		t.Errorf("expected ErrUserNotVerified, got %v", err)
	}
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	hasher := password.NewHasher("pepper", 10)
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

	svc := appauth.NewService(
		authRepo, &mockVerifierRepo{}, &mockProfileRepo{}, &mockReseterRepo{}, &mockQuotaRepo{},
		&mockCache{}, &mockQueue{}, *jwt.NewJwt([]byte("s")), *hasher, &mockTxManager{},
	)

	_, err := svc.Login(context.Background(), "test@example.com", "wrongPass")
	if !errors.Is(err, domainauth.ErrInvalidCredential) {
		t.Errorf("expected ErrInvalidCredential, got %v", err)
	}
}
