package user_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	appuser "github.com/labib0x9/ffgif/internal/app/user"
	domainauth "github.com/labib0x9/ffgif/internal/domain/auth"
	authMocks "github.com/labib0x9/ffgif/internal/domain/auth/mocks"
	domainmedia "github.com/labib0x9/ffgif/internal/domain/media"
	domainuser "github.com/labib0x9/ffgif/internal/domain/user"
	userMocks "github.com/labib0x9/ffgif/internal/domain/user/mocks"
	dbMocks "github.com/labib0x9/ffgif/internal/port/db/mocks"
	"github.com/labib0x9/ffgif/pkg/apperr"
	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
	"github.com/labib0x9/ffgif/pkg/password"
)

type userTestDeps struct {
	ctrl      *gomock.Controller
	userRepo  *userMocks.MockUserRepository
	quotaRepo *userMocks.MockQuotaRepository
	authRepo  *authMocks.MockAuthRepository
	txManager *dbMocks.MockTxManager
	jwtSvc    *jwtpkg.Jwt
	hasher    *password.Hasher
	svc       appuser.Service
}

func newUserTestDeps(t *testing.T) *userTestDeps {
	ctrl := gomock.NewController(t)
	hasher := password.NewHasher("test-pepper", 4)
	jwtSvc := jwtpkg.NewJwt([]byte("test-secret-32-bytes-long-key-1234"))

	deps := &userTestDeps{
		ctrl:      ctrl,
		userRepo:  userMocks.NewMockUserRepository(ctrl),
		quotaRepo: userMocks.NewMockQuotaRepository(ctrl),
		authRepo:  authMocks.NewMockAuthRepository(ctrl),
		txManager: dbMocks.NewMockTxManager(ctrl),
		jwtSvc:    jwtSvc,
		hasher:    hasher,
	}

	deps.svc = appuser.NewService(
		deps.userRepo,
		deps.quotaRepo,
		deps.authRepo,
		deps.txManager,
		*deps.jwtSvc,
		*deps.hasher,
	)
	return deps
}

func TestUserService_GetProfile(t *testing.T) {
	t.Run("success: returns user profile", func(t *testing.T) {
		d := newUserTestDeps(t)
		ctx := context.Background()

		uid := uuid.New().String()
		expectedProfile := domainuser.ProfileResponse{
			Username:   "johndoe",
			Fullname:   "John Doe",
			Email:      "john@example.com",
			ProfilePic: "https://avatar.com/pic.png",
			IsVerified: true,
		}

		d.userRepo.EXPECT().GetProfile(gomock.Any(), gomock.Eq(uid), gomock.Eq(false)).
			Return(expectedProfile, nil).Times(1)

		profile, err := d.svc.GetProfile(ctx, uid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if profile.Username != "johndoe" || profile.Email != "john@example.com" {
			t.Errorf("profile fields mismatch: got %+v", profile)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		d := newUserTestDeps(t)
		ctx := context.Background()

		uid := uuid.New().String()
		d.userRepo.EXPECT().GetProfile(gomock.Any(), gomock.Eq(uid), gomock.Eq(false)).
			Return(domainuser.ProfileResponse{}, sql.ErrNoRows).Times(1)

		_, err := d.svc.GetProfile(ctx, uid)
		if err == nil {
			t.Fatalf("expected error for non-existent profile")
		}
	})
}

func TestUserService_GetQuota(t *testing.T) {
	t.Run("success: returns user quota", func(t *testing.T) {
		d := newUserTestDeps(t)
		ctx := context.Background()

		uid := uuid.New().String()
		expectedQuota := &domainuser.Quota{
			UsedBytes:  1024,
			TotalBytes: 50 * 1024 * 1024,
			GifCount:   2,
			GitCount:   10,
		}

		d.quotaRepo.EXPECT().GetById(gomock.Any(), gomock.Eq(uid)).
			Return(expectedQuota, nil).Times(1)

		quota, err := d.svc.GetQuota(ctx, uid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if quota.UsedBytes != 1024 || quota.GifCount != 2 {
			t.Errorf("quota fields mismatch: got %+v", quota)
		}
	})
}

func TestUserService_UpdateProfile(t *testing.T) {
	t.Run("success: optimistic locking matches ETag and updates profile", func(t *testing.T) {
		d := newUserTestDeps(t)
		ctx := context.Background()

		uid := uuid.New().String()
		now := time.Now()
		lastUpdatedAt := now.Format(time.RFC3339Nano)

		newName := "New Name"
		req := domainuser.ProfileUpdateRequest{Fullname: &newName}

		d.txManager.EXPECT().WithRC(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
			return fn(ctx)
		}).Times(1)

		d.userRepo.EXPECT().GetProfile(gomock.Any(), gomock.Eq(uid), gomock.Eq(true)).
			Return(domainuser.ProfileResponse{
				Fullname:  "Old Name",
				UpdatedAt: now,
			}, nil).Times(1)

		d.userRepo.EXPECT().UpdateProfile(gomock.Any(), gomock.Eq(req), gomock.Eq(uid)).
			Return(domainuser.ProfileResponse{
				Fullname:  "New Name",
				UpdatedAt: now.Add(1 * time.Second),
			}, nil).Times(1)

		updated, err := d.svc.UpdateProfile(ctx, req, uid, lastUpdatedAt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.Fullname != "New Name" {
			t.Errorf("expected updated fullname 'New Name', got '%s'", updated.Fullname)
		}
	})

	t.Run("ETag mismatch: returns ErrETagValidationFailed", func(t *testing.T) {
		d := newUserTestDeps(t)
		ctx := context.Background()

		uid := uuid.New().String()
		now := time.Now()
		staleUpdatedAt := now.Add(-1 * time.Hour).Format(time.RFC3339Nano)

		newName := "New Name"
		req := domainuser.ProfileUpdateRequest{Fullname: &newName}

		d.txManager.EXPECT().WithRC(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
			return fn(ctx)
		}).Times(1)

		d.userRepo.EXPECT().GetProfile(gomock.Any(), gomock.Eq(uid), gomock.Eq(true)).
			Return(domainuser.ProfileResponse{
				Fullname:  "Current Name",
				UpdatedAt: now,
			}, nil).Times(1)

		_, err := d.svc.UpdateProfile(ctx, req, uid, staleUpdatedAt)
		if !errors.Is(err, domainmedia.ErrETagValidationFailed) {
			t.Fatalf("expected ErrETagValidationFailed, got %v", err)
		}
	})
}

func TestUserService_ChangePassword(t *testing.T) {
	t.Run("success: validates current password and updates hash", func(t *testing.T) {
		d := newUserTestDeps(t)
		ctx := context.Background()

		rawUid := uuid.New()
		uidStr := rawUid.String()
		oldHash, _ := d.hasher.GenerateHash("oldPass123!")

		d.authRepo.EXPECT().GetById(gomock.Any(), gomock.Eq(rawUid)).
			Return(domainauth.User{
				Id:           rawUid,
				PasswordHash: oldHash,
			}, nil).Times(1)

		d.userRepo.EXPECT().ChangePassword(gomock.Any(), gomock.Eq(uidStr), gomock.Cond(func(x any) bool {
			hash, ok := x.(string)
			return ok && d.hasher.CompareHashAndPassword(hash, "newPass123!")
		})).Return(nil).Times(1)

		err := d.svc.ChangePassword(ctx, uidStr, "oldPass123!", "newPass123!", "newPass123!")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("password mismatch: returns ErrPasswordMismatched", func(t *testing.T) {
		d := newUserTestDeps(t)
		ctx := context.Background()

		rawUid := uuid.New()
		uidStr := rawUid.String()
		oldHash, _ := d.hasher.GenerateHash("oldPass123!")

		d.authRepo.EXPECT().GetById(gomock.Any(), gomock.Eq(rawUid)).
			Return(domainauth.User{
				Id:           rawUid,
				PasswordHash: oldHash,
			}, nil).Times(1)

		err := d.svc.ChangePassword(ctx, uidStr, "oldPass123!", "newPass123!", "differentPass!")
		if !errors.Is(err, apperr.ErrPasswordMismatched) {
			t.Fatalf("expected ErrPasswordMismatched, got %v", err)
		}
	})

	t.Run("wrong current password: returns ErrInvalidCredential", func(t *testing.T) {
		d := newUserTestDeps(t)
		ctx := context.Background()

		rawUid := uuid.New()
		uidStr := rawUid.String()
		oldHash, _ := d.hasher.GenerateHash("oldPass123!")

		d.authRepo.EXPECT().GetById(gomock.Any(), gomock.Eq(rawUid)).
			Return(domainauth.User{
				Id:           rawUid,
				PasswordHash: oldHash,
			}, nil).Times(1)

		err := d.svc.ChangePassword(ctx, uidStr, "wrongCurrentPass!", "newPass123!", "newPass123!")
		if !errors.Is(err, domainauth.ErrInvalidCredential) {
			t.Fatalf("expected ErrInvalidCredential, got %v", err)
		}
	})

	// EXPECTED TO FAIL / Design Weakness: Changing password to the EXACT same old password should be rejected.
	t.Run("same old and new password: should reject reusing old password", func(t *testing.T) {
		d := newUserTestDeps(t)
		ctx := context.Background()

		rawUid := uuid.New()
		uidStr := rawUid.String()
		oldHash, _ := d.hasher.GenerateHash("samePass123!")

		d.authRepo.EXPECT().GetById(gomock.Any(), gomock.Eq(rawUid)).
			Return(domainauth.User{
				Id:           rawUid,
				PasswordHash: oldHash,
			}, nil).Times(1)

		d.userRepo.EXPECT().ChangePassword(gomock.Any(), gomock.Eq(uidStr), gomock.Any()).
			Return(nil).AnyTimes()

		err := d.svc.ChangePassword(ctx, uidStr, "samePass123!", "samePass123!", "samePass123!")
		if err == nil {
			t.Errorf("BUG/WEAKNESS: ChangePassword accepted the same old password as the new password")
		}
	})
}

func TestUserService_DeleteUser(t *testing.T) {
	t.Run("success: verifies password and deletes user", func(t *testing.T) {
		d := newUserTestDeps(t)
		ctx := context.Background()

		rawUid := uuid.New()
		uidStr := rawUid.String()
		passHash, _ := d.hasher.GenerateHash("correctPass123!")

		d.authRepo.EXPECT().GetById(gomock.Any(), gomock.Eq(rawUid)).
			Return(domainauth.User{
				Id:           rawUid,
				PasswordHash: passHash,
			}, nil).Times(1)

		d.authRepo.EXPECT().DeleteById(gomock.Any(), gomock.Eq(rawUid)).Return(nil).Times(1)

		err := d.svc.DeleteUser(ctx, uidStr, "correctPass123!")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("wrong password: returns ErrInvalidCredential", func(t *testing.T) {
		d := newUserTestDeps(t)
		ctx := context.Background()

		rawUid := uuid.New()
		uidStr := rawUid.String()
		passHash, _ := d.hasher.GenerateHash("correctPass123!")

		d.authRepo.EXPECT().GetById(gomock.Any(), gomock.Eq(rawUid)).
			Return(domainauth.User{
				Id:           rawUid,
				PasswordHash: passHash,
			}, nil).Times(1)

		err := d.svc.DeleteUser(ctx, uidStr, "wrongPass!")
		if !errors.Is(err, domainauth.ErrInvalidCredential) {
			t.Fatalf("expected ErrInvalidCredential, got %v", err)
		}
	})
}
