package user_test

// import (
// 	"context"
// 	"errors"
// 	"testing"

// 	"github.com/google/uuid"
// 	appuser "github.com/labib0x9/ffgif/internal/app/user"
// 	domainauth "github.com/labib0x9/ffgif/internal/domain/auth"
// 	domainuser "github.com/labib0x9/ffgif/internal/domain/user"
// 	"github.com/labib0x9/ffgif/pkg/jwt"
// 	"github.com/labib0x9/ffgif/pkg/password"
// )

// // --- Mocks ---

// type mockUserRepo struct {
// 	getProfileFunc     func(ctx context.Context, id string) (domainuser.ProfileResp, error)
// 	updateProfileFunc  func(ctx context.Context, profile domainuser.ProfileResp, id string) (domainuser.ProfileResp, error)
// 	setProfileFunc     func(ctx context.Context, profile domainuser.Profile) error
// 	changePasswordFunc func(ctx context.Context, userId string, hash string) error
// }

// func (m *mockUserRepo) GetProfile(ctx context.Context, id string) (domainuser.ProfileResp, error) {
// 	if m.getProfileFunc != nil {
// 		return m.getProfileFunc(ctx, id)
// 	}
// 	return domainuser.ProfileResp{}, nil
// }
// func (m *mockUserRepo) UpdateProfile(ctx context.Context, profile domainuser.ProfileResp, id string) (domainuser.ProfileResp, error) {
// 	if m.updateProfileFunc != nil {
// 		return m.updateProfileFunc(ctx, profile, id)
// 	}
// 	return profile, nil
// }
// func (m *mockUserRepo) SetProfile(ctx context.Context, profile domainuser.Profile) error {
// 	if m.setProfileFunc != nil {
// 		return m.setProfileFunc(ctx, profile)
// 	}
// 	return nil
// }
// func (m *mockUserRepo) ChangePassword(ctx context.Context, userId string, hash string) error {
// 	if m.changePasswordFunc != nil {
// 		return m.changePasswordFunc(ctx, userId, hash)
// 	}
// 	return nil
// }

// type mockQuotaRepo struct {
// 	getByIdFunc func(ctx context.Context, id string) (*domainuser.Quota, error)
// }

// func (m *mockQuotaRepo) Create(ctx context.Context, quota domainuser.Quota) error { return nil }
// func (m *mockQuotaRepo) GetById(ctx context.Context, userId string) (*domainuser.Quota, error) {
// 	if m.getByIdFunc != nil {
// 		return m.getByIdFunc(ctx, userId)
// 	}
// 	return &domainuser.Quota{UserID: uuid.New(), TotalBytes: 1024 * 1024, GifCount: 5}, nil
// }

// type mockAuthRepo struct {
// 	getByIdFunc        func(ctx context.Context, id uuid.UUID) (domainauth.User, error)
// 	deleteByIdFunc     func(ctx context.Context, id uuid.UUID) error
// 	updatePasswordFunc func(ctx context.Context, id uuid.UUID, passHash string) error
// }

// func (m *mockAuthRepo) GetByEmail(ctx context.Context, email string) (domainauth.User, error) {
// 	return domainauth.User{}, nil
// }
// func (m *mockAuthRepo) GetById(ctx context.Context, id uuid.UUID) (domainauth.User, error) {
// 	if m.getByIdFunc != nil {
// 		return m.getByIdFunc(ctx, id)
// 	}
// 	return domainauth.User{}, nil
// }
// func (m *mockAuthRepo) Create(ctx context.Context, user domainauth.User) (domainauth.User, error) {
// 	return user, nil
// }
// func (m *mockAuthRepo) DeleteById(ctx context.Context, id uuid.UUID) error {
// 	if m.deleteByIdFunc != nil {
// 		return m.deleteByIdFunc(ctx, id)
// 	}
// 	return nil
// }
// func (m *mockAuthRepo) DeleteByEmail(ctx context.Context, email string) error { return nil }
// func (m *mockAuthRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passHash string) error {
// 	if m.updatePasswordFunc != nil {
// 		return m.updatePasswordFunc(ctx, id, passHash)
// 	}
// 	return nil
// }
// func (m *mockAuthRepo) SetVerified(ctx context.Context, userId uuid.UUID) error { return nil }
// func (m *mockAuthRepo) Upgrade(ctx context.Context, id string, user domainauth.User) (domainauth.User, error) {
// 	return user, nil
// }

// // --- Tests ---

// func TestUserService_GetProfile_Success(t *testing.T) {
// 	expectedProfile := domainuser.ProfileResp{
// 		Username: "john_doe",
// 		Email:    "john@example.com",
// 		Fullname: "John Doe",
// 	}

// 	userRepo := &mockUserRepo{
// 		getProfileFunc: func(ctx context.Context, id string) (domainuser.ProfileResp, error) {
// 			return expectedProfile, nil
// 		},
// 	}

// 	svc := appuser.NewService(
// 		userRepo,
// 		&mockQuotaRepo{},
// 		&mockAuthRepo{},
// 		*jwt.NewJwt([]byte("secret")),
// 		*password.NewHasher("pepper", 10),
// 	)

// 	profile, err := svc.GetProfile(context.Background(), "user-123")
// 	if err != nil {
// 		t.Fatalf("expected GetProfile to succeed, got: %v", err)
// 	}

// 	if profile.Username != expectedProfile.Username {
// 		t.Errorf("expected username %s, got %s", expectedProfile.Username, profile.Username)
// 	}
// }

// func TestUserService_ChangePassword_Success(t *testing.T) {
// 	hasher := password.NewHasher("pepper", 10)
// 	oldHash, _ := hasher.GenerateHash("oldPass123")
// 	userId := uuid.New()
// 	updated := false

// 	userRepo := &mockUserRepo{
// 		changePasswordFunc: func(ctx context.Context, userId string, hash string) error {
// 			updated = true
// 			if !hasher.CompareHashAndPassword(hash, "oldPass123") {
// 				t.Errorf("expected updated password hash to match oldPass123")
// 			}
// 			return nil
// 		},
// 	}

// 	authRepo := &mockAuthRepo{
// 		getByIdFunc: func(ctx context.Context, id uuid.UUID) (domainauth.User, error) {
// 			return domainauth.User{Id: id, PasswordHash: oldHash}, nil
// 		},
// 	}

// 	svc := appuser.NewService(
// 		userRepo,
// 		&mockQuotaRepo{},
// 		authRepo,
// 		*jwt.NewJwt([]byte("secret")),
// 		*hasher,
// 	)

// 	err := svc.ChangePassword(context.Background(), userId.String(), "oldPass123", "oldPass123", "oldPass123")
// 	if err != nil {
// 		t.Fatalf("expected ChangePassword to succeed, got: %v", err)
// 	}

// 	if !updated {
// 		t.Errorf("expected password to be updated in database")
// 	}
// }

// func TestUserService_ChangePassword_Mismatch(t *testing.T) {
// 	hasher := password.NewHasher("pepper", 10)
// 	svc := appuser.NewService(
// 		&mockUserRepo{},
// 		&mockQuotaRepo{},
// 		&mockAuthRepo{},
// 		*jwt.NewJwt([]byte("secret")),
// 		*hasher,
// 	)

// 	err := svc.ChangePassword(context.Background(), uuid.New().String(), "old", "newPass1", "differentPass2")
// 	if !errors.Is(err, domainuser.ErrPasswordMismatched) {
// 		t.Errorf("expected ErrPasswordMismatched, got %v", err)
// 	}
// }

// func TestUserService_GetQuota_Success(t *testing.T) {
// 	userId := uuid.New()
// 	quotaRepo := &mockQuotaRepo{
// 		getByIdFunc: func(ctx context.Context, id string) (*domainuser.Quota, error) {
// 			return &domainuser.Quota{UserID: userId, TotalBytes: 500000, GifCount: 5}, nil
// 		},
// 	}

// 	svc := appuser.NewService(
// 		&mockUserRepo{},
// 		quotaRepo,
// 		&mockAuthRepo{},
// 		*jwt.NewJwt([]byte("secret")),
// 		*password.NewHasher("pepper", 10),
// 	)

// 	quota, err := svc.GetQuota(context.Background(), userId.String())
// 	if err != nil {
// 		t.Fatalf("expected GetQuota to succeed, got: %v", err)
// 	}

// 	if quota.TotalBytes != 500000 {
// 		t.Errorf("expected TotalBytes 500000, got %d", quota.TotalBytes)
// 	}
// }
