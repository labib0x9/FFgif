package auth_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"

	appauth "github.com/labib0x9/ffgif/internal/app/auth"
	domainauth "github.com/labib0x9/ffgif/internal/domain/auth"
	authmocks "github.com/labib0x9/ffgif/internal/domain/auth/mocks"
	domainuser "github.com/labib0x9/ffgif/internal/domain/user"
	usermocks "github.com/labib0x9/ffgif/internal/domain/user/mocks"
	cachemocks "github.com/labib0x9/ffgif/internal/port/cache/mocks"
	dbmocks "github.com/labib0x9/ffgif/internal/port/db/mocks"
	"github.com/labib0x9/ffgif/internal/port/queue"
	queuemocks "github.com/labib0x9/ffgif/internal/port/queue/mocks"
	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
	"github.com/labib0x9/ffgif/pkg/password"
	tokenpkg "github.com/labib0x9/ffgif/pkg/token"
)

// ---------------------------------------------------------------------------
// harness
// ---------------------------------------------------------------------------

type authDeps struct {
	authRepo     *authmocks.MockAuthRepository
	verifierRepo *authmocks.MockVerifierRepository
	profileRepo  *usermocks.MockUserRepository
	reseterRepo  *authmocks.MockReseterRepository
	quotaRepo    *usermocks.MockQuotaRepository
	cache        *cachemocks.MockCache
	queue        *queuemocks.MockQueue
	tnx          *dbmocks.MockTxManager
	jwt          *jwtpkg.Jwt
	svc          appauth.Service
}

const testSecret = "test-secret-key-1234"

func newAuthDeps(t *testing.T) *authDeps {
	t.Helper()
	ctrl := gomock.NewController(t)

	d := &authDeps{
		authRepo:     authmocks.NewMockAuthRepository(ctrl),
		verifierRepo: authmocks.NewMockVerifierRepository(ctrl),
		profileRepo:  usermocks.NewMockUserRepository(ctrl),
		reseterRepo:  authmocks.NewMockReseterRepository(ctrl),
		quotaRepo:    usermocks.NewMockQuotaRepository(ctrl),
		cache:        cachemocks.NewMockCache(ctrl),
		queue:        queuemocks.NewMockQueue(ctrl),
		tnx:          dbmocks.NewMockTxManager(ctrl),
		jwt:          jwtpkg.NewJwt([]byte(testSecret)),
	}

	hasher := password.NewHasher("pepper", bcrypt.MinCost)
	d.svc = appauth.NewService(
		d.authRepo, d.verifierRepo, d.profileRepo, d.reseterRepo, d.quotaRepo,
		d.cache, d.queue, *d.jwt, *hasher, d.tnx,
	)
	return d
}

// passthroughTx makes the mocked TxManager actually run the closure it is given,
// so a rollback in the service is observable as "the later mocks were never called".
func (d *authDeps) passthroughTx(times int) {
	d.tnx.EXPECT().
		With(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
			return fn(ctx)
		}).
		Times(times)
}

func hashOf(raw string) string { return tokenpkg.GetTokenHash(raw) }

// ---------------------------------------------------------------------------
// custom gomock matchers — assert on struct *fields*, not just "some struct"
// ---------------------------------------------------------------------------

// reseterWithToken matches an auth.Reseter whose UserId matches and whose Token
// field satisfies tokenCheck. Used to assert what is actually persisted.
type reseterWithToken struct {
	userID     uuid.UUID
	tokenCheck func(string) bool
	desc       string
	got        string
}

func (m *reseterWithToken) Matches(x any) bool {
	r, ok := x.(domainauth.Reseter)
	if !ok {
		return false
	}
	m.got = r.Token
	return r.UserId == m.userID && m.tokenCheck(r.Token)
}

func (m *reseterWithToken) String() string {
	return "auth.Reseter{UserId: " + m.userID.String() + ", Token: " + m.desc + "} (got Token=" + m.got + ")"
}

// userMatcher asserts the exact field values handed to AuthRepository.Create.
type userMatcher struct {
	email, username, fullname, role string
	plaintextPassword               string
	got                             domainauth.User
}

func (m *userMatcher) Matches(x any) bool {
	u, ok := x.(domainauth.User)
	if !ok {
		return false
	}
	m.got = u
	if u.Email != m.email || u.Username != m.username || u.Fullname != m.fullname {
		return false
	}
	if u.Role != m.role || u.IsVerified {
		return false
	}
	// the password must be hashed, never stored or passed in the clear
	if u.PasswordHash == "" || u.PasswordHash == m.plaintextPassword {
		return false
	}
	return true
}

func (m *userMatcher) String() string {
	return "auth.User{Email:" + m.email + ", Role:" + m.role + ", IsVerified:false, PasswordHash: bcrypt(not plaintext)}"
}

// ===========================================================================
// ForgotPassword
// ===========================================================================

// EXPECTED TO FAIL: internal/app/auth/forgot_password.go discards the hash
// returned by token.GenerateToken() ("resetToken, _ := token.GenerateToken()")
// and stores the RAW token in Reseter.Token, which is persisted to the
// reseter.token_hash column. pkg/token returns (raw, hash) precisely so the
// raw token goes to the user's inbox and only the SHA-256 hash is stored.
// As written, anyone with read access to the DB can reset any account.
//
// Contract: the value handed to ReseterRepository.Create MUST equal
// sha256(the token that is emailed).
func TestForgotPassword_StoresHashOfEmailedToken(t *testing.T) {
	d := newAuthDeps(t)
	userID := uuid.New()

	d.authRepo.EXPECT().
		GetByEmail(gomock.Any(), gomock.Eq("user@example.com")).
		Return(domainauth.User{Id: userID, Email: "user@example.com", IsVerified: true}, nil).
		Times(1)

	// no pre-existing reset token
	d.reseterRepo.EXPECT().
		GetById(gomock.Any(), gomock.Eq(userID)).
		Return(domainauth.Reseter{}, sql.ErrNoRows).
		Times(1)

	var storedToken string
	d.reseterRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, r domainauth.Reseter) error {
			storedToken = r.Token
			return nil
		}).
		Times(1)

	var emailedToken string
	d.queue.EXPECT().
		PublishEmail(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, msg queue.EmailMessage) error {
			if msg.To != "user@example.com" {
				t.Errorf("email addressed to %q, want user@example.com", msg.To)
			}
			if msg.Name != "forgot-password" {
				t.Errorf("email job type %q, want forgot-password", msg.Name)
			}
			emailedToken = msg.Token
			return nil
		}).
		Times(1)

	if err := d.svc.ForgotPassword(context.Background(), "user@example.com"); err != nil {
		t.Fatalf("ForgotPassword: %v", err)
	}

	if emailedToken == "" {
		t.Fatal("no token was emailed to the user")
	}
	if storedToken == emailedToken {
		t.Errorf("raw reset token was persisted verbatim (%q); "+
			"a DB reader can take over any account", storedToken)
	}
	if want := hashOf(emailedToken); storedToken != want {
		t.Errorf("persisted token = %q, want sha256(emailed token) = %q", storedToken, want)
	}
}

// EXPECTED TO FAIL: forgot_password.go reuses whatever row GetById returns
// without inspecting Reseter.Used. A token that has already been redeemed gets
// re-sent by email, so a leaked-and-used token stays live for the whole TTL.
func TestForgotPassword_DoesNotReuseAlreadyUsedToken(t *testing.T) {
	d := newAuthDeps(t)
	userID := uuid.New()

	d.authRepo.EXPECT().
		GetByEmail(gomock.Any(), gomock.Eq("user@example.com")).
		Return(domainauth.User{Id: userID, Email: "user@example.com", IsVerified: true}, nil).
		Times(1)

	d.reseterRepo.EXPECT().
		GetById(gomock.Any(), gomock.Eq(userID)).
		Return(domainauth.Reseter{
			Id:     7,
			UserId: userID,
			Token:  "already-redeemed-token",
			Used:   true,
			// still inside its TTL, so the repo's `expire_at > now()` filter
			// does not screen it out
			ExpireAt: time.Now().Add(time.Hour),
		}, nil).
		Times(1)

	// A consumed token must not be handed back out: the service is expected to
	// mint a fresh one instead.
	d.reseterRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	d.queue.EXPECT().
		PublishEmail(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, msg queue.EmailMessage) error {
			if msg.Token == "already-redeemed-token" {
				t.Errorf("a spent reset token was emailed again: %q", msg.Token)
			}
			return nil
		}).
		Times(1)

	if err := d.svc.ForgotPassword(context.Background(), "user@example.com"); err != nil {
		t.Fatalf("ForgotPassword: %v", err)
	}
}

func TestForgotPassword_UnverifiedUserIsRejected(t *testing.T) {
	d := newAuthDeps(t)

	d.authRepo.EXPECT().
		GetByEmail(gomock.Any(), gomock.Eq("unverified@example.com")).
		Return(domainauth.User{Id: uuid.New(), Email: "unverified@example.com", IsVerified: false}, nil).
		Times(1)

	// no token minted, no mail sent
	err := d.svc.ForgotPassword(context.Background(), "unverified@example.com")
	if !errors.Is(err, domainauth.ErrUserNotVerified) {
		t.Errorf("err = %v, want ErrUserNotVerified", err)
	}
}

func TestForgotPassword_UnknownUser(t *testing.T) {
	d := newAuthDeps(t)

	d.authRepo.EXPECT().
		GetByEmail(gomock.Any(), gomock.Eq("nobody@example.com")).
		Return(domainauth.User{}, sql.ErrNoRows).
		Times(1)

	err := d.svc.ForgotPassword(context.Background(), "nobody@example.com")
	if !errors.Is(err, domainauth.ErrUserNotFound) {
		t.Errorf("err = %v, want ErrUserNotFound", err)
	}
}

// A Redis/Postgres outage on the lookup must not be mistaken for "no token yet".
func TestForgotPassword_LookupBackendErrorIsPropagated(t *testing.T) {
	d := newAuthDeps(t)
	userID := uuid.New()

	d.authRepo.EXPECT().
		GetByEmail(gomock.Any(), gomock.Any()).
		Return(domainauth.User{Id: userID, Email: "u@example.com", IsVerified: true}, nil).
		Times(1)

	d.reseterRepo.EXPECT().
		GetById(gomock.Any(), gomock.Eq(userID)).
		Return(domainauth.Reseter{}, errors.New("dial tcp 127.0.0.1:5432: connect: connection refused")).
		Times(1)

	// Create and PublishEmail must NOT be reached; gomock fails the test if they are.
	err := d.svc.ForgotPassword(context.Background(), "u@example.com")
	if !errors.Is(err, domainauth.ErrTokenFetchFailed) {
		t.Errorf("err = %v, want it to wrap ErrTokenFetchFailed", err)
	}
}

// ===========================================================================
// Signup
// ===========================================================================

func TestSignup_PersistsHashedPasswordAndHashedVerifyToken(t *testing.T) {
	d := newAuthDeps(t)
	createdID := uuid.New()
	d.passthroughTx(1)

	d.authRepo.EXPECT().
		GetByEmail(gomock.Any(), gomock.Eq("new@example.com")).
		Return(domainauth.User{}, sql.ErrNoRows).
		Times(1)

	um := &userMatcher{
		email: "new@example.com", username: "newbie", fullname: "New Bie",
		role: "user", plaintextPassword: "sup3r-s3cret!",
	}
	d.authRepo.EXPECT().
		Create(gomock.Any(), um).
		Return(domainauth.User{Id: createdID, Email: "new@example.com"}, nil).
		Times(1)

	var storedVerifyToken string
	d.verifierRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, v domainauth.Verifier) error {
			if v.UserId != createdID {
				t.Errorf("verifier.UserId = %v, want %v", v.UserId, createdID)
			}
			storedVerifyToken = v.Token
			return nil
		}).
		Times(1)

	d.profileRepo.EXPECT().
		SetProfile(gomock.Any(), gomock.Eq(domainuser.Profile{UserId: createdID, ProfilePic: ""})).
		Return(nil).
		Times(1)

	d.quotaRepo.EXPECT().
		Create(gomock.Any(), gomock.Eq(domainuser.Quota{UserID: createdID})).
		Return(nil).
		Times(1)

	var emailedToken string
	d.queue.EXPECT().
		PublishEmail(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, msg queue.EmailMessage) error {
			if msg.Name != "signup" {
				t.Errorf("email job type %q, want signup", msg.Name)
			}
			emailedToken = msg.Token
			return nil
		}).
		Times(1)

	if _, err := d.svc.Signup(context.Background(), "new@example.com", "newbie", "New Bie", "sup3r-s3cret!"); err != nil {
		t.Fatalf("Signup: %v", err)
	}

	// signup gets this right — it stores the hash and mails the raw token.
	// This is the behaviour ForgotPassword is missing.
	if storedVerifyToken != hashOf(emailedToken) {
		t.Errorf("stored verify token = %q, want sha256(emailed) = %q", storedVerifyToken, hashOf(emailedToken))
	}
	if storedVerifyToken == emailedToken {
		t.Error("raw verification token was persisted verbatim")
	}
}

func TestSignup_DuplicateEmailIsRejected(t *testing.T) {
	d := newAuthDeps(t)
	d.passthroughTx(1)

	d.authRepo.EXPECT().
		GetByEmail(gomock.Any(), gomock.Eq("taken@example.com")).
		Return(domainauth.User{Id: uuid.New()}, nil).
		Times(1)

	// no Create, no verifier, no mail
	_, err := d.svc.Signup(context.Background(), "taken@example.com", "u", "U", "pw")
	if !errors.Is(err, domainauth.ErrUserExists) {
		t.Errorf("err = %v, want ErrUserExists", err)
	}
}

// A failure late in the transaction must abort the whole signup: crucially, no
// verification email may go out for a user that was rolled back.
func TestSignup_QuotaFailureAbortsAndSendsNoEmail(t *testing.T) {
	d := newAuthDeps(t)
	createdID := uuid.New()
	d.passthroughTx(1)

	d.authRepo.EXPECT().GetByEmail(gomock.Any(), gomock.Any()).Return(domainauth.User{}, sql.ErrNoRows).Times(1)
	d.authRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(domainauth.User{Id: createdID}, nil).Times(1)
	d.verifierRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	d.profileRepo.EXPECT().SetProfile(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	d.quotaRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(errors.New("quota table deadlock")).
		Times(1)

	// PublishEmail is deliberately NOT expected: gomock fails on an unexpected call.
	_, err := d.svc.Signup(context.Background(), "new@example.com", "u", "U", "pw")
	if !errors.Is(err, domainauth.ErrQuotaCreateFailed) {
		t.Errorf("err = %v, want it to wrap ErrQuotaCreateFailed", err)
	}
}

// ===========================================================================
// Login
// ===========================================================================

func TestLogin(t *testing.T) {
	hasher := password.NewHasher("pepper", bcrypt.MinCost)
	goodHash, err := hasher.GenerateHash("correct-horse")
	if err != nil {
		t.Fatalf("GenerateHash: %v", err)
	}
	userID := uuid.New()

	tests := []struct {
		name     string
		stored   domainauth.User
		repoErr  error
		password string
		wantErr  error
	}{
		{
			name:     "correct credentials",
			stored:   domainauth.User{Id: userID, Email: "a@b.c", Fullname: "A B", Role: "user", IsVerified: true, PasswordHash: goodHash},
			password: "correct-horse",
		},
		{
			name:     "wrong password",
			stored:   domainauth.User{Id: userID, Email: "a@b.c", Fullname: "A B", Role: "user", IsVerified: true, PasswordHash: goodHash},
			password: "wrong-horse",
			wantErr:  domainauth.ErrInvalidCredential,
		},
		{
			name:     "empty password must not authenticate",
			stored:   domainauth.User{Id: userID, Email: "a@b.c", Fullname: "A B", Role: "user", IsVerified: true, PasswordHash: goodHash},
			password: "",
			wantErr:  domainauth.ErrInvalidCredential,
		},
		{
			name:     "unverified user",
			stored:   domainauth.User{Id: userID, Email: "a@b.c", Fullname: "A B", Role: "user", IsVerified: false, PasswordHash: goodHash},
			password: "correct-horse",
			wantErr:  domainauth.ErrUserNotVerified,
		},
		{
			name:     "unknown user is indistinguishable from a bad password",
			repoErr:  sql.ErrNoRows,
			password: "correct-horse",
			wantErr:  domainauth.ErrInvalidCredential,
		},
		{
			name:     "empty stored hash must never authenticate",
			stored:   domainauth.User{Id: userID, Email: "a@b.c", Fullname: "A B", Role: "user", IsVerified: true, PasswordHash: ""},
			password: "",
			wantErr:  domainauth.ErrInvalidCredential,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := newAuthDeps(t)
			d.authRepo.EXPECT().
				GetByEmail(gomock.Any(), gomock.Eq("a@b.c")).
				Return(tc.stored, tc.repoErr).
				Times(1)

			res, err := d.svc.Login(context.Background(), "a@b.c", tc.password)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				if res != nil {
					t.Error("a token was issued despite the failure")
				}
				return
			}
			if err != nil {
				t.Fatalf("Login: %v", err)
			}
			if res.Id != userID {
				t.Errorf("Id = %v, want %v", res.Id, userID)
			}

			claims, err := d.jwt.Verify(res.Token)
			if err != nil {
				t.Fatalf("issued token does not verify: %v", err)
			}
			if claims.Subject != userID.String() {
				t.Errorf("token subject = %q, want %q", claims.Subject, userID.String())
			}
			if claims.Role != "user" {
				t.Errorf("token role = %q, want user", claims.Role)
			}
		})
	}
}

// A user must never be able to mint a token for a role they do not hold.
func TestLogin_TokenCarriesStoredRoleNotRequestedRole(t *testing.T) {
	d := newAuthDeps(t)
	hasher := password.NewHasher("pepper", bcrypt.MinCost)
	h, _ := hasher.GenerateHash("pw")
	userID := uuid.New()

	d.authRepo.EXPECT().
		GetByEmail(gomock.Any(), gomock.Any()).
		Return(domainauth.User{
			Id: userID, Email: "a@b.c", Fullname: "A B",
			Role: "user", IsVerified: true, PasswordHash: h,
		}, nil).
		Times(1)

	res, err := d.svc.Login(context.Background(), "a@b.c", "pw")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	claims, err := d.jwt.Verify(res.Token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.Role == "admin" {
		t.Fatal("privilege escalation: token minted with admin role")
	}
}

// ===========================================================================
// Logout / blocklist
// ===========================================================================

func TestLogout_BlocklistsTokenForItsRemainingLifetime(t *testing.T) {
	d := newAuthDeps(t)
	tokenStr := "the.jwt.string"
	exp := time.Now().Add(30 * time.Minute)

	d.cache.EXPECT().
		Set(gomock.Any(), gomock.Eq("token_blocklist:"+tokenStr), gomock.Eq("1"), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _ string, ttl time.Duration) error {
			// TTL must cover the rest of the token's life, and must not be
			// unbounded (that would leak Redis memory forever).
			if ttl <= 0 {
				t.Errorf("blocklist TTL = %v, want > 0", ttl)
			}
			if ttl > 31*time.Minute {
				t.Errorf("blocklist TTL = %v, want <= the token's remaining lifetime", ttl)
			}
			return nil
		}).
		Times(1)

	claims := jwtpkg.Payload{RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(exp)}}
	if err := d.svc.Logout(context.Background(), tokenStr, claims); err != nil {
		t.Fatalf("Logout: %v", err)
	}
}

// An already-expired token needs no blocklist entry — writing one with a
// negative TTL would make Redis store it forever.
func TestLogout_ExpiredTokenIsNotWrittenToCache(t *testing.T) {
	d := newAuthDeps(t)
	claims := jwtpkg.Payload{
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute))},
	}
	// cache.Set is not expected; gomock fails the test if it is called.
	if err := d.svc.Logout(context.Background(), "expired.jwt", claims); err != nil {
		t.Fatalf("Logout: %v", err)
	}
}

// If the blocklist write fails, logout must report failure — silently
// succeeding would tell the user they are logged out while the token stays live.
func TestLogout_CacheFailureIsReported(t *testing.T) {
	d := newAuthDeps(t)
	boom := errors.New("redis: connection refused")

	d.cache.EXPECT().
		Set(gomock.Any(), gomock.Eq("token_blocklist:tok"), gomock.Eq("1"), gomock.Any()).
		Return(boom).
		Times(1)

	claims := jwtpkg.Payload{
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	}
	if err := d.svc.Logout(context.Background(), "tok", claims); !errors.Is(err, boom) {
		t.Errorf("err = %v, want the cache error", err)
	}
}

// ===========================================================================
// Verify
// ===========================================================================

func TestVerify_LooksUpByHashNotRawToken(t *testing.T) {
	d := newAuthDeps(t)
	d.passthroughTx(1)

	raw := "11111111-2222-3333-4444-555555555555"
	userID := uuid.New()

	// The contract: the raw token from the email is hashed before the lookup.
	d.verifierRepo.EXPECT().
		GetByHash(gomock.Any(), gomock.Eq(hashOf(raw))).
		Return(domainauth.Verifier{Id: 42, UserId: userID, Token: hashOf(raw)}, nil).
		Times(1)

	d.authRepo.EXPECT().SetVerified(gomock.Any(), gomock.Eq(userID)).Return(nil).Times(1)
	d.verifierRepo.EXPECT().Delete(gomock.Any(), gomock.Eq(int64(42))).Return(nil).Times(1)

	if err := d.svc.Verify(context.Background(), raw); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

// Double-verify: the first call consumes the row, so the second lookup misses.
func TestVerify_SecondUseOfSameTokenIsRejected(t *testing.T) {
	d := newAuthDeps(t)
	raw := "already-used-token"

	d.verifierRepo.EXPECT().
		GetByHash(gomock.Any(), gomock.Eq(hashOf(raw))).
		Return(domainauth.Verifier{}, sql.ErrNoRows).
		Times(1)

	// SetVerified must not be reached.
	err := d.svc.Verify(context.Background(), raw)
	if !errors.Is(err, domainauth.ErrInvalidToken) {
		t.Errorf("err = %v, want ErrInvalidToken", err)
	}
}

// Marking the user verified and consuming the token must be atomic: if the
// delete fails, the whole thing must roll back.
func TestVerify_TokenDeleteFailureAbortsTransaction(t *testing.T) {
	d := newAuthDeps(t)
	d.passthroughTx(1)

	raw := "tok"
	userID := uuid.New()
	deleteErr := errors.New("delete failed")

	d.verifierRepo.EXPECT().GetByHash(gomock.Any(), gomock.Eq(hashOf(raw))).
		Return(domainauth.Verifier{Id: 9, UserId: userID}, nil).Times(1)
	d.authRepo.EXPECT().SetVerified(gomock.Any(), gomock.Eq(userID)).Return(nil).Times(1)
	d.verifierRepo.EXPECT().Delete(gomock.Any(), gomock.Eq(int64(9))).Return(deleteErr).Times(1)

	if err := d.svc.Verify(context.Background(), raw); !errors.Is(err, deleteErr) {
		t.Errorf("err = %v, want the delete error to surface so the tx rolls back", err)
	}
}

// A malformed/unknown token must not be treated as a backend outage and vice versa.
func TestVerify_BackendErrorIsDistinctFromInvalidToken(t *testing.T) {
	d := newAuthDeps(t)
	raw := "tok"

	d.verifierRepo.EXPECT().GetByHash(gomock.Any(), gomock.Eq(hashOf(raw))).
		Return(domainauth.Verifier{}, errors.New("connection reset by peer")).Times(1)

	err := d.svc.Verify(context.Background(), raw)
	if errors.Is(err, domainauth.ErrInvalidToken) {
		t.Error("a database outage was reported to the caller as an invalid token")
	}
	if !errors.Is(err, domainauth.ErrTokenFetchFailed) {
		t.Errorf("err = %v, want it to wrap ErrTokenFetchFailed", err)
	}
}

// ===========================================================================
// ResendVerify
// ===========================================================================

func TestResendVerify_ReplacesOldTokenAndStoresOnlyTheHash(t *testing.T) {
	d := newAuthDeps(t)
	userID := uuid.New()

	d.authRepo.EXPECT().GetByEmail(gomock.Any(), gomock.Eq("u@example.com")).
		Return(domainauth.User{Id: userID, Email: "u@example.com", IsVerified: false}, nil).Times(1)

	d.verifierRepo.EXPECT().GetById(gomock.Any(), gomock.Eq(userID)).
		Return(domainauth.Verifier{Id: 3, UserId: userID, Token: "old-hash"}, nil).Times(1)

	// the superseded token must be revoked, not left usable alongside the new one
	d.verifierRepo.EXPECT().Delete(gomock.Any(), gomock.Eq(int64(3))).Return(nil).Times(1)

	var stored string
	d.verifierRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, v domainauth.Verifier) error {
			if v.UserId != userID {
				t.Errorf("verifier.UserId = %v, want %v", v.UserId, userID)
			}
			stored = v.Token
			return nil
		}).Times(1)

	var emailed string
	d.queue.EXPECT().PublishEmail(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, msg queue.EmailMessage) error {
			if msg.Name != "resend-verify" {
				t.Errorf("email job type %q, want resend-verify", msg.Name)
			}
			emailed = msg.Token
			return nil
		}).Times(1)

	if err := d.svc.ResendVerify(context.Background(), "u@example.com"); err != nil {
		t.Fatalf("ResendVerify: %v", err)
	}
	if stored != hashOf(emailed) {
		t.Errorf("stored token = %q, want sha256(emailed) = %q", stored, hashOf(emailed))
	}
}

func TestResendVerify_AlreadyVerifiedUserIsRejected(t *testing.T) {
	d := newAuthDeps(t)
	d.authRepo.EXPECT().GetByEmail(gomock.Any(), gomock.Any()).
		Return(domainauth.User{Id: uuid.New(), IsVerified: true}, nil).Times(1)

	// no token churn, no mail
	err := d.svc.ResendVerify(context.Background(), "u@example.com")
	if !errors.Is(err, domainauth.ErrUserAlreadyVerified) {
		t.Errorf("err = %v, want ErrUserAlreadyVerified", err)
	}
}

// ===========================================================================
// ResetPasswordPost
// ===========================================================================

// EXPECTED TO FAIL: internal/app/auth/reset_password.go accepts confirmPass as a
// parameter and never reads it. The mismatch check lives only in the HTTP
// handler's `eqfield=Password` tag, so any non-HTTP caller (a CLI, a worker, a
// future gRPC transport) silently resets the password to `pass`.
func TestResetPasswordPost_MismatchedConfirmationIsRejected(t *testing.T) {
	d := newAuthDeps(t)
	userID := uuid.New()

	d.reseterRepo.EXPECT().
		GetByToken(gomock.Any(), gomock.Eq("reset-tok")).
		Return(domainauth.Reseter{Id: 1, UserId: userID, Token: "reset-tok"}, nil).
		Times(1)
	d.authRepo.EXPECT().
		GetById(gomock.Any(), gomock.Eq(userID)).
		Return(domainauth.User{Id: userID, Email: "u@example.com"}, nil).
		AnyTimes()
	d.tnx.EXPECT().
		With(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
			return fn(ctx)
		}).
		AnyTimes()

	// The password must not be touched when the confirmation does not match.
	d.authRepo.EXPECT().
		UpdatePassword(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id uuid.UUID, _ string) error {
			t.Errorf("password for user %v was reset even though "+
				"confirmPass (%q) did not match pass (%q)", id, "TYPO-different!", "newpassword1!")
			return nil
		}).
		AnyTimes()
	d.reseterRepo.EXPECT().DeleteById(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	d.queue.EXPECT().PublishEmail(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	err := d.svc.ResetPasswordPost(context.Background(), "reset-tok", "newpassword1!", "TYPO-different!")
	if err == nil {
		t.Error("mismatched password confirmation was accepted; " +
			"ResetPasswordPost ignores its confirmPass argument entirely")
	}
}

func TestResetPasswordPost_HappyPathUpdatesHashAndConsumesToken(t *testing.T) {
	d := newAuthDeps(t)
	d.passthroughTx(1)
	userID := uuid.New()

	d.reseterRepo.EXPECT().GetByToken(gomock.Any(), gomock.Eq("reset-tok")).
		Return(domainauth.Reseter{Id: 11, UserId: userID, Token: "reset-tok"}, nil).Times(1)
	d.authRepo.EXPECT().GetById(gomock.Any(), gomock.Eq(userID)).
		Return(domainauth.User{Id: userID, Email: "u@example.com"}, nil).Times(1)

	d.authRepo.EXPECT().
		UpdatePassword(gomock.Any(), gomock.Eq(userID), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, hash string) error {
			if hash == "newpassword1!" {
				t.Error("the new password was stored in plaintext")
			}
			hasher := password.NewHasher("pepper", bcrypt.MinCost)
			if !hasher.CompareHashAndPassword(hash, "newpassword1!") {
				t.Error("stored hash does not verify against the new password")
			}
			return nil
		}).Times(1)

	// the reset token must be single-use
	d.reseterRepo.EXPECT().DeleteById(gomock.Any(), gomock.Eq(int64(11))).Return(nil).Times(1)

	d.queue.EXPECT().PublishEmail(gomock.Any(), gomock.Eq(queue.EmailMessage{
		To:   "u@example.com",
		Name: "reset-password",
	})).Return(nil).Times(1)

	if err := d.svc.ResetPasswordPost(context.Background(), "reset-tok", "newpassword1!", "newpassword1!"); err != nil {
		t.Fatalf("ResetPasswordPost: %v", err)
	}
}

// If the password update cannot be committed, the reset token must survive so
// the user can retry — and no "your password was changed" mail may go out.
func TestResetPasswordPost_UpdateFailureSendsNoNotification(t *testing.T) {
	d := newAuthDeps(t)
	d.passthroughTx(1)
	userID := uuid.New()
	updateErr := errors.New("update failed")

	d.reseterRepo.EXPECT().GetByToken(gomock.Any(), gomock.Any()).
		Return(domainauth.Reseter{Id: 11, UserId: userID}, nil).Times(1)
	d.authRepo.EXPECT().GetById(gomock.Any(), gomock.Any()).
		Return(domainauth.User{Id: userID, Email: "u@example.com"}, nil).Times(1)
	d.authRepo.EXPECT().UpdatePassword(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(updateErr).Times(1)

	// DeleteById and PublishEmail are not expected.
	if err := d.svc.ResetPasswordPost(context.Background(), "tok", "pw!", "pw!"); !errors.Is(err, updateErr) {
		t.Errorf("err = %v, want the update error", err)
	}
}

// EXPECTED TO FAIL (information disclosure): ResetPasswordGet echoes the stored
// reseter.Token straight back to the caller. Combined with the raw-token
// storage bug above, the endpoint hands back a directly usable credential.
// A validity probe should return only whether the token is usable.
func TestResetPasswordGet_DoesNotEchoStoredCredential(t *testing.T) {
	d := newAuthDeps(t)

	d.reseterRepo.EXPECT().
		GetByToken(gomock.Any(), gomock.Eq("submitted-token")).
		Return(domainauth.Reseter{Id: 1, UserId: uuid.New(), Token: "STORED-SECRET-VALUE"}, nil).
		Times(1)

	got, err := d.svc.ResetPasswordGet(context.Background(), "submitted-token")
	if err != nil {
		t.Fatalf("ResetPasswordGet: %v", err)
	}
	if got == "STORED-SECRET-VALUE" {
		t.Errorf("the stored reset credential was returned to the caller: %q", got)
	}
}

func TestResetPasswordGet_InvalidTokenIsRejected(t *testing.T) {
	d := newAuthDeps(t)
	d.reseterRepo.EXPECT().GetByToken(gomock.Any(), gomock.Eq("bogus")).
		Return(domainauth.Reseter{}, sql.ErrNoRows).Times(1)

	if _, err := d.svc.ResetPasswordGet(context.Background(), "bogus"); !errors.Is(err, domainauth.ErrReseterTokenFatchFailed) {
		t.Errorf("err = %v, want it to wrap ErrReseterTokenFatchFailed", err)
	}
}
