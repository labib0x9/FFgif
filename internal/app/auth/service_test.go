package auth_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	appauth "github.com/labib0x9/ffgif/internal/app/auth"
	domainauth "github.com/labib0x9/ffgif/internal/domain/auth"
	authMocks "github.com/labib0x9/ffgif/internal/domain/auth/mocks"
	domainuser "github.com/labib0x9/ffgif/internal/domain/user"
	userMocks "github.com/labib0x9/ffgif/internal/domain/user/mocks"
	cacheMocks "github.com/labib0x9/ffgif/internal/port/cache/mocks"
	dbMocks "github.com/labib0x9/ffgif/internal/port/db/mocks"
	"github.com/labib0x9/ffgif/internal/port/queue"
	queueMocks "github.com/labib0x9/ffgif/internal/port/queue/mocks"
	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
	"github.com/labib0x9/ffgif/pkg/password"
	tokenpkg "github.com/labib0x9/ffgif/pkg/token"
)

type authTestDeps struct {
	ctrl         *gomock.Controller
	authRepo     *authMocks.MockAuthRepository
	verifierRepo *authMocks.MockVerifierRepository
	profileRepo  *userMocks.MockUserRepository
	reseterRepo  *authMocks.MockReseterRepository
	quotaRepo    *userMocks.MockQuotaRepository
	cache        *cacheMocks.MockCache
	queue        *queueMocks.MockQueue
	txManager    *dbMocks.MockTxManager
	jwtSvc       *jwtpkg.Jwt
	hasher       *password.Hasher
	svc          appauth.Service
}

func newAuthTestDeps(t *testing.T) *authTestDeps {
	ctrl := gomock.NewController(t)
	hasher := password.NewHasher("test-pepper", 4)
	jwtSvc := jwtpkg.NewJwt([]byte("test-secret-32-bytes-long-key-1234"))

	deps := &authTestDeps{
		ctrl:         ctrl,
		authRepo:     authMocks.NewMockAuthRepository(ctrl),
		verifierRepo: authMocks.NewMockVerifierRepository(ctrl),
		profileRepo:  userMocks.NewMockUserRepository(ctrl),
		reseterRepo:  authMocks.NewMockReseterRepository(ctrl),
		quotaRepo:    userMocks.NewMockQuotaRepository(ctrl),
		cache:        cacheMocks.NewMockCache(ctrl),
		queue:        queueMocks.NewMockQueue(ctrl),
		txManager:    dbMocks.NewMockTxManager(ctrl),
		jwtSvc:       jwtSvc,
		hasher:       hasher,
	}

	deps.svc = appauth.NewService(
		deps.authRepo,
		deps.verifierRepo,
		deps.profileRepo,
		deps.reseterRepo,
		deps.quotaRepo,
		deps.cache,
		deps.queue,
		*deps.jwtSvc,
		*deps.hasher,
		deps.txManager,
	)
	return deps
}

func TestAuthService_Signup(t *testing.T) {
	t.Run("success: creates user, profile, quota, verifier and publishes email", func(t *testing.T) {
		d := newAuthTestDeps(t)
		ctx := context.Background()

		d.authRepo.EXPECT().GetByEmail(gomock.Any(), "user@example.com").Return(domainauth.User{}, sql.ErrNoRows).Times(1)

		d.txManager.EXPECT().With(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
			return fn(ctx)
		}).Times(1)

		d.authRepo.EXPECT().Create(gomock.Any(), gomock.Cond(func(x any) bool {
			u, ok := x.(domainauth.User)
			return ok && u.Email == "user@example.com" && u.Username == "testuser" && u.Fullname == "Test User"
		})).DoAndReturn(func(ctx context.Context, u domainauth.User) (domainauth.User, error) {
			u.Id = uuid.New()
			return u, nil
		}).Times(1)

		d.profileRepo.EXPECT().SetProfile(gomock.Any(), gomock.Cond(func(x any) bool {
			p, ok := x.(domainuser.Profile)
			return ok && p.UserId != uuid.Nil
		})).Return(nil).Times(1)

		d.quotaRepo.EXPECT().Create(gomock.Any(), gomock.Cond(func(x any) bool {
			q, ok := x.(domainuser.Quota)
			return ok && q.UserID != uuid.Nil
		})).Return(nil).Times(1)

		d.verifierRepo.EXPECT().Create(gomock.Any(), gomock.Cond(func(x any) bool {
			v, ok := x.(domainauth.Verifier)
			return ok && v.UserId != uuid.Nil && v.Token != ""
		})).Return(nil).Times(1)

		d.queue.EXPECT().PublishEmail(gomock.Any(), gomock.Cond(func(x any) bool {
			m, ok := x.(queue.EmailMessage)
			return ok && m.To == "user@example.com" && m.Name == "signup" && m.Token != ""
		})).Return(nil).Times(1)

		res, err := d.svc.Signup(ctx, "user@example.com", "testuser", "Test User", "password123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res == nil {
			t.Errorf("expected non-nil SignupResult")
		}
	})

	t.Run("duplicate email: returns ErrUserExists", func(t *testing.T) {
		d := newAuthTestDeps(t)
		ctx := context.Background()

		d.txManager.EXPECT().With(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
			return fn(ctx)
		}).Times(1)

		d.authRepo.EXPECT().GetByEmail(gomock.Any(), "user@example.com").Return(domainauth.User{
			Id:    uuid.New(),
			Email: "user@example.com",
		}, nil).Times(1)

		_, err := d.svc.Signup(ctx, "user@example.com", "testuser", "Test User", "password123")
		if !errors.Is(err, domainauth.ErrUserExists) {
			t.Fatalf("expected ErrUserExists, got %v", err)
		}
	})
}

func TestAuthService_Login(t *testing.T) {
	t.Run("success: returns token for verified user", func(t *testing.T) {
		d := newAuthTestDeps(t)
		ctx := context.Background()

		hash, _ := d.hasher.GenerateHash("password123")
		uid := uuid.New()
		user := domainauth.User{
			Id:           uid,
			Email:        "user@example.com",
			Fullname:     "Test User",
			PasswordHash: hash,
			IsVerified:   true,
			Role:         "user",
		}

		d.authRepo.EXPECT().GetByEmail(gomock.Any(), "user@example.com").Return(user, nil).Times(1)

		res, err := d.svc.Login(ctx, "user@example.com", "password123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Token == "" || res.Id != uid {
			t.Errorf("expected valid login result, got %+v", res)
		}
	})

	t.Run("unverified user: returns ErrUserNotVerified", func(t *testing.T) {
		d := newAuthTestDeps(t)
		ctx := context.Background()

		hash, _ := d.hasher.GenerateHash("password123")
		user := domainauth.User{
			Id:           uuid.New(),
			Email:        "user@example.com",
			PasswordHash: hash,
			IsVerified:   false,
		}

		d.authRepo.EXPECT().GetByEmail(gomock.Any(), "user@example.com").Return(user, nil).Times(1)

		_, err := d.svc.Login(ctx, "user@example.com", "password123")
		if !errors.Is(err, domainauth.ErrUserNotVerified) {
			t.Fatalf("expected ErrUserNotVerified, got %v", err)
		}
	})

	t.Run("invalid password: returns ErrInvalidCredential", func(t *testing.T) {
		d := newAuthTestDeps(t)
		ctx := context.Background()

		hash, _ := d.hasher.GenerateHash("password123")
		user := domainauth.User{
			Id:           uuid.New(),
			Email:        "user@example.com",
			PasswordHash: hash,
			IsVerified:   true,
		}

		d.authRepo.EXPECT().GetByEmail(gomock.Any(), "user@example.com").Return(user, nil).Times(1)

		_, err := d.svc.Login(ctx, "user@example.com", "wrongpassword")
		if !errors.Is(err, domainauth.ErrInvalidCredential) {
			t.Fatalf("expected ErrInvalidCredential, got %v", err)
		}
	})
}

func TestAuthService_Verify(t *testing.T) {
	t.Run("success: marks user verified and deletes verifier token", func(t *testing.T) {
		d := newAuthTestDeps(t)
		ctx := context.Background()

		rawToken := "raw-token-123"
		tokenHash := tokenpkg.GetTokenHash(rawToken)
		uid := uuid.New()

		d.verifierRepo.EXPECT().GetByHash(gomock.Any(), tokenHash).Return(domainauth.Verifier{
			Id:        10,
			UserId:    uid,
			Token:     tokenHash,
			ExpireAt:  time.Now().Add(10 * time.Minute),
			CreatedAt: time.Now(),
		}, nil).Times(1)

		d.txManager.EXPECT().With(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
			return fn(ctx)
		}).Times(1)

		d.authRepo.EXPECT().SetVerified(gomock.Any(), uid).Return(nil).Times(1)
		d.verifierRepo.EXPECT().Delete(gomock.Any(), int64(10)).Return(nil).Times(1)

		err := d.svc.Verify(ctx, rawToken)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// EXPECTED TO FAIL: verify.go does not validate verifier.ExpireAt, allowing expired tokens to verify users.
	t.Run("expired token: should reject verification and return error", func(t *testing.T) {
		d := newAuthTestDeps(t)
		ctx := context.Background()

		rawToken := "expired-token-123"
		tokenHash := tokenpkg.GetTokenHash(rawToken)
		uid := uuid.New()

		d.verifierRepo.EXPECT().GetByHash(gomock.Any(), tokenHash).Return(domainauth.Verifier{
			Id:        11,
			UserId:    uid,
			Token:     tokenHash,
			ExpireAt:  time.Now().Add(-10 * time.Minute), // expired in the past
			CreatedAt: time.Now().Add(-20 * time.Minute),
		}, nil).Times(1)

		d.txManager.EXPECT().With(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
			return fn(ctx)
		}).AnyTimes()
		d.authRepo.EXPECT().SetVerified(gomock.Any(), uid).Return(nil).AnyTimes()
		d.verifierRepo.EXPECT().Delete(gomock.Any(), int64(11)).Return(nil).AnyTimes()

		err := d.svc.Verify(ctx, rawToken)
		if err == nil {
			t.Errorf("BUG DETECTED: Verify accepted an expired token (ExpireAt was in the past)")
		}
	})

	t.Run("double-verify / token not found: returns ErrInvalidToken", func(t *testing.T) {
		d := newAuthTestDeps(t)
		ctx := context.Background()

		rawToken := "consumed-token"
		tokenHash := tokenpkg.GetTokenHash(rawToken)

		d.verifierRepo.EXPECT().GetByHash(gomock.Any(), tokenHash).Return(domainauth.Verifier{}, sql.ErrNoRows).Times(1)

		err := d.svc.Verify(ctx, rawToken)
		if !errors.Is(err, domainauth.ErrInvalidToken) {
			t.Fatalf("expected ErrInvalidToken, got %v", err)
		}
	})
}

// EXPECTED TO FAIL: Reset token hashing bug in internal/app/auth/forgot_password.go.
// The code stores raw resetToken in reseter.Token instead of sha256(rawToken),
// exposing raw reset tokens in the database (db: "token_hash").
func TestAuthService_ForgotPassword_TokenHashing_Adversarial(t *testing.T) {
	d := newAuthTestDeps(t)
	ctx := context.Background()

	uid := uuid.New()
	user := domainauth.User{
		Id:         uid,
		Email:      "user@example.com",
		IsVerified: true,
	}

	d.authRepo.EXPECT().GetByEmail(gomock.Any(), "user@example.com").Return(user, nil).Times(1)
	d.reseterRepo.EXPECT().GetById(gomock.Any(), uid).Return(domainauth.Reseter{}, sql.ErrNoRows).Times(1)

	var savedReseterToken string
	var publishedEmailToken string

	d.reseterRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, r domainauth.Reseter) error {
		savedReseterToken = r.Token
		return nil
	}).Times(1)

	d.queue.EXPECT().PublishEmail(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, msg queue.EmailMessage) error {
		publishedEmailToken = msg.Token
		return nil
	}).Times(1)

	err := d.svc.ForgotPassword(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedHash := tokenpkg.GetTokenHash(publishedEmailToken)
	if savedReseterToken != expectedHash {
		t.Errorf("BUG DETECTED: ReseterRepository.Create received raw token '%s' instead of hashed token '%s'",
			savedReseterToken, expectedHash)
	}
}

func TestAuthService_ResetPasswordPost(t *testing.T) {
	t.Run("success: hashes password, updates user, deletes token, publishes email", func(t *testing.T) {
		d := newAuthTestDeps(t)
		ctx := context.Background()

		rawToken := "reset-token-xyz"
		uid := uuid.New()

		d.reseterRepo.EXPECT().GetByToken(gomock.Any(), rawToken).Return(domainauth.Reseter{
			Id:     5,
			UserId: uid,
			Token:  rawToken,
		}, nil).Times(1)

		d.authRepo.EXPECT().GetById(gomock.Any(), uid).Return(domainauth.User{
			Id:    uid,
			Email: "user@example.com",
		}, nil).Times(1)

		d.txManager.EXPECT().With(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
			return fn(ctx)
		}).Times(1)

		d.authRepo.EXPECT().UpdatePassword(gomock.Any(), uid, gomock.Any()).Return(nil).Times(1)
		d.reseterRepo.EXPECT().DeleteById(gomock.Any(), int64(5)).Return(nil).Times(1)
		d.queue.EXPECT().PublishEmail(gomock.Any(), gomock.Cond(func(x any) bool {
			m, ok := x.(queue.EmailMessage)
			return ok && m.To == "user@example.com" && m.Name == "reset-password"
		})).Return(nil).Times(1)

		err := d.svc.ResetPasswordPost(ctx, rawToken, "NewPass123!", "NewPass123!")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("token not found: returns error", func(t *testing.T) {
		d := newAuthTestDeps(t)
		ctx := context.Background()

		d.reseterRepo.EXPECT().GetByToken(gomock.Any(), "unknown-token").Return(domainauth.Reseter{}, sql.ErrNoRows).Times(1)

		err := d.svc.ResetPasswordPost(ctx, "unknown-token", "NewPass123!", "NewPass123!")
		if err == nil {
			t.Fatalf("expected error when reset token not found")
		}
	})
}

func TestAuthService_Logout(t *testing.T) {
	t.Run("success: caches blocklisted token with remaining expiration", func(t *testing.T) {
		d := newAuthTestDeps(t)
		ctx := context.Background()

		exp := time.Now().Add(15 * time.Minute)
		payload := jwtpkg.Payload{
			Role: "user",
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   uuid.New().String(),
				ExpiresAt: jwt.NewNumericDate(exp),
			},
		}

		rawJwt := "sample.jwt.token"

		d.cache.EXPECT().Set(gomock.Any(), gomock.Eq("token_blocklist:"+rawJwt), gomock.Eq("1"), gomock.Any()).
			DoAndReturn(func(ctx context.Context, k, v string, ttl time.Duration) error {
				if ttl <= 0 || ttl > 16*time.Minute {
					t.Errorf("unexpected TTL: %v", ttl)
				}
				return nil
			}).Times(1)

		err := d.svc.Logout(ctx, rawJwt, payload)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("concurrent logouts for same token", func(t *testing.T) {
		d := newAuthTestDeps(t)
		ctx := context.Background()

		exp := time.Now().Add(15 * time.Minute)
		payload := jwtpkg.Payload{
			Role: "user",
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   uuid.New().String(),
				ExpiresAt: jwt.NewNumericDate(exp),
			},
		}
		rawJwt := "concurrent.jwt.token"

		d.cache.EXPECT().Set(gomock.Any(), gomock.Eq("token_blocklist:"+rawJwt), gomock.Eq("1"), gomock.Any()).
			Return(nil).AnyTimes()

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = d.svc.Logout(ctx, rawJwt, payload)
			}()
		}
		wg.Wait()
	})
}
