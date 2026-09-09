package user_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"

	appuser "github.com/labib0x9/ffgif/internal/app/user"
	domainauth "github.com/labib0x9/ffgif/internal/domain/auth"
	authmocks "github.com/labib0x9/ffgif/internal/domain/auth/mocks"
	domainmedia "github.com/labib0x9/ffgif/internal/domain/media"
	domainuser "github.com/labib0x9/ffgif/internal/domain/user"
	usermocks "github.com/labib0x9/ffgif/internal/domain/user/mocks"
	dbmocks "github.com/labib0x9/ffgif/internal/port/db/mocks"
	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
	"github.com/labib0x9/ffgif/pkg/apperr"
	"github.com/labib0x9/ffgif/pkg/password"
)

type userDeps struct {
	userRepo  *usermocks.MockUserRepository
	quotaRepo *usermocks.MockQuotaRepository
	authRepo  *authmocks.MockAuthRepository
	tnx       *dbmocks.MockTxManager
	hasher    *password.Hasher
	svc       appuser.Service
}

func newUserDeps(t *testing.T) *userDeps {
	t.Helper()
	ctrl := gomock.NewController(t)
	d := &userDeps{
		userRepo:  usermocks.NewMockUserRepository(ctrl),
		quotaRepo: usermocks.NewMockQuotaRepository(ctrl),
		authRepo:  authmocks.NewMockAuthRepository(ctrl),
		tnx:       dbmocks.NewMockTxManager(ctrl),
		hasher:    password.NewHasher("pepper", bcrypt.MinCost),
	}
	d.svc = appuser.NewService(
		d.userRepo, d.quotaRepo, d.authRepo, d.tnx,
		*jwtpkg.NewJwt([]byte("test-secret-key-1234")), *d.hasher,
	)
	return d
}

func (d *userDeps) passthroughRC(times int) {
	d.tnx.EXPECT().
		WithRC(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
			return fn(ctx)
		}).
		Times(times)
}

// ===========================================================================
// ChangePassword
// ===========================================================================

func TestChangePassword_StoresANewHashOfTheNewPassword(t *testing.T) {
	d := newUserDeps(t)
	id := uuid.New()
	current, err := d.hasher.GenerateHash("old-password!")
	if err != nil {
		t.Fatalf("GenerateHash: %v", err)
	}

	d.authRepo.EXPECT().
		GetById(gomock.Any(), gomock.Eq(id)).
		Return(domainauth.User{Id: id, PasswordHash: current}, nil).
		Times(1)

	d.userRepo.EXPECT().
		ChangePassword(gomock.Any(), gomock.Eq(id.String()), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, hash string) error {
			if hash == "new-password!" {
				t.Error("the new password was stored in plaintext")
			}
			if hash == current {
				t.Error("the stored hash did not change")
			}
			if !d.hasher.CompareHashAndPassword(hash, "new-password!") {
				t.Error("the stored hash does not verify against the new password")
			}
			return nil
		}).
		Times(1)

	if err := d.svc.ChangePassword(context.Background(), id.String(),
		"old-password!", "new-password!", "new-password!"); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}
}

func TestChangePassword_Rejections(t *testing.T) {
	id := uuid.New()

	tests := []struct {
		name        string
		userID      string
		current     string
		newPass     string
		confirm     string
		storedFor   string // plaintext the stored hash was made from
		repoErr     error
		wantErr     error
		wantAttempt bool // whether userRepo.ChangePassword may be called
	}{
		{
			name:      "wrong current password",
			userID:    id.String(),
			current:   "not-my-password",
			newPass:   "new-password!",
			confirm:   "new-password!",
			storedFor: "old-password!",
			wantErr:   domainauth.ErrInvalidCredential,
		},
		{
			name:      "new and confirmation do not match",
			userID:    id.String(),
			current:   "old-password!",
			newPass:   "new-password!",
			confirm:   "typo-password!",
			storedFor: "old-password!",
			wantErr:   apperr.ErrPasswordMismatched,
		},
		{
			name:      "empty current password",
			userID:    id.String(),
			current:   "",
			newPass:   "new-password!",
			confirm:   "new-password!",
			storedFor: "old-password!",
			wantErr:   domainauth.ErrInvalidCredential,
		},
		{
			name:      "unknown user",
			userID:    id.String(),
			current:   "old-password!",
			newPass:   "new-password!",
			confirm:   "new-password!",
			storedFor: "old-password!",
			repoErr:   sql.ErrNoRows,
			wantErr:   domainauth.ErrUserNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := newUserDeps(t)
			hash, err := d.hasher.GenerateHash(tc.storedFor)
			if err != nil {
				t.Fatalf("GenerateHash: %v", err)
			}

			d.authRepo.EXPECT().
				GetById(gomock.Any(), gomock.Eq(id)).
				Return(domainauth.User{Id: id, PasswordHash: hash}, tc.repoErr).
				Times(1)

			// userRepo.ChangePassword must not be reached for any of these.
			got := d.svc.ChangePassword(context.Background(), tc.userID, tc.current, tc.newPass, tc.confirm)
			if !errors.Is(got, tc.wantErr) {
				t.Errorf("err = %v, want %v", got, tc.wantErr)
			}
		})
	}
}

func TestChangePassword_MalformedUserIdNeverTouchesTheDatabase(t *testing.T) {
	d := newUserDeps(t)
	// authRepo.GetById must not be called for an unparseable id.
	if err := d.svc.ChangePassword(context.Background(), "not-a-uuid",
		"old", "new", "new"); err == nil {
		t.Error("a malformed user id was accepted")
	}
}

// EXPECTED TO FAIL: internal/app/user/change_password.go compares the new
// password only against confirmPass. Setting the password to its current value
// is accepted and rewrites the row with a fresh bcrypt hash, so the user
// believes they rotated a credential that in fact did not change. A rotation
// endpoint must refuse a no-op change.
func TestChangePassword_RefusesToReuseTheCurrentPassword(t *testing.T) {
	d := newUserDeps(t)
	id := uuid.New()
	hash, err := d.hasher.GenerateHash("same-password!")
	if err != nil {
		t.Fatalf("GenerateHash: %v", err)
	}

	d.authRepo.EXPECT().
		GetById(gomock.Any(), gomock.Eq(id)).
		Return(domainauth.User{Id: id, PasswordHash: hash}, nil).
		Times(1)

	d.userRepo.EXPECT().
		ChangePassword(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, newHash string) error {
			t.Error("the password was 'changed' to the value it already had; " +
				"the user is told the rotation succeeded but the credential is unchanged")
			return nil
		}).
		AnyTimes()

	if err := d.svc.ChangePassword(context.Background(), id.String(),
		"same-password!", "same-password!", "same-password!"); err == nil {
		t.Error("ChangePassword accepted the current password as the new password")
	}
}

func TestChangePassword_RepositoryFailureIsReported(t *testing.T) {
	d := newUserDeps(t)
	id := uuid.New()
	hash, _ := d.hasher.GenerateHash("old-password!")

	d.authRepo.EXPECT().GetById(gomock.Any(), gomock.Eq(id)).
		Return(domainauth.User{Id: id, PasswordHash: hash}, nil).Times(1)
	d.userRepo.EXPECT().ChangePassword(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errors.New("update failed")).Times(1)

	err := d.svc.ChangePassword(context.Background(), id.String(), "old-password!", "new!", "new!")
	if !errors.Is(err, apperr.ErrTableUpdateFailed) {
		t.Errorf("err = %v, want it to wrap ErrTableUpdateFailed", err)
	}
}

// ===========================================================================
// DeleteUser
// ===========================================================================

func TestDeleteUser_RequiresTheCorrectPassword(t *testing.T) {
	d := newUserDeps(t)
	id := uuid.New()
	hash, _ := d.hasher.GenerateHash("my-password!")

	d.authRepo.EXPECT().GetById(gomock.Any(), gomock.Eq(id)).
		Return(domainauth.User{Id: id, PasswordHash: hash}, nil).Times(1)

	// DeleteById must not be reached with the wrong password.
	err := d.svc.DeleteUser(context.Background(), id.String(), "wrong-password")
	if !errors.Is(err, domainauth.ErrInvalidCredential) {
		t.Errorf("err = %v, want ErrInvalidCredential", err)
	}
}

func TestDeleteUser_DeletesTheCallersOwnIdOnly(t *testing.T) {
	d := newUserDeps(t)
	id := uuid.New()
	hash, _ := d.hasher.GenerateHash("my-password!")

	d.authRepo.EXPECT().GetById(gomock.Any(), gomock.Eq(id)).
		Return(domainauth.User{Id: id, PasswordHash: hash}, nil).Times(1)

	// The id deleted must be the one derived from the caller's own token,
	// never anything taken from the request body.
	d.authRepo.EXPECT().DeleteById(gomock.Any(), gomock.Eq(id)).Return(nil).Times(1)

	if err := d.svc.DeleteUser(context.Background(), id.String(), "my-password!"); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
}

// If the delete itself fails the caller must be told, not left believing their
// account is gone while it is still live.
func TestDeleteUser_DeleteFailureIsReported(t *testing.T) {
	d := newUserDeps(t)
	id := uuid.New()
	hash, _ := d.hasher.GenerateHash("my-password!")

	d.authRepo.EXPECT().GetById(gomock.Any(), gomock.Eq(id)).
		Return(domainauth.User{Id: id, PasswordHash: hash}, nil).Times(1)
	d.authRepo.EXPECT().DeleteById(gomock.Any(), gomock.Eq(id)).
		Return(errors.New("fk violation: gifs still reference this user")).Times(1)

	err := d.svc.DeleteUser(context.Background(), id.String(), "my-password!")
	if !errors.Is(err, apperr.ErrTableUpdateFailed) {
		t.Errorf("err = %v, want it to wrap ErrTableUpdateFailed", err)
	}
}

// A lookup failure mid-request must abort before anything is deleted.
func TestDeleteUser_LookupFailureLeavesTheAccountIntact(t *testing.T) {
	d := newUserDeps(t)
	id := uuid.New()

	d.authRepo.EXPECT().GetById(gomock.Any(), gomock.Eq(id)).
		Return(domainauth.User{}, errors.New("connection reset")).Times(1)

	// DeleteById must not be reached.
	if err := d.svc.DeleteUser(context.Background(), id.String(), "my-password!"); err == nil {
		t.Error("want an error when the user cannot be loaded")
	}
}

func TestDeleteUser_MalformedIdNeverTouchesTheDatabase(t *testing.T) {
	d := newUserDeps(t)
	if err := d.svc.DeleteUser(context.Background(), "not-a-uuid", "pw"); err == nil {
		t.Error("a malformed user id was accepted")
	}
}

// ===========================================================================
// UpdateProfile
// ===========================================================================

func TestUpdateProfile_ForwardsTheRequestAndHonoursTheETag(t *testing.T) {
	d := newUserDeps(t)
	d.passthroughRC(1)
	now := time.Now().UTC()
	etag := now.Format(time.RFC3339Nano)

	newName := "Updated Name"
	req := domainuser.ProfileUpdateRequest{Fullname: &newName}

	d.userRepo.EXPECT().
		GetProfile(gomock.Any(), gomock.Eq("user-1"), gomock.Eq(true)).
		Return(domainuser.ProfileResponse{Username: "u", UpdatedAt: now}, nil).
		Times(1)

	d.userRepo.EXPECT().
		UpdateProfile(gomock.Any(), gomock.Any(), gomock.Eq("user-1")).
		DoAndReturn(func(_ context.Context, r domainuser.ProfileUpdateRequest, _ string) (domainuser.ProfileResponse, error) {
			if r.Fullname == nil || *r.Fullname != "Updated Name" {
				t.Errorf("Fullname = %v, want Updated Name", r.Fullname)
			}
			if r.Email != nil {
				t.Errorf("Email = %q, want nil for a field the client did not send", *r.Email)
			}
			return domainuser.ProfileResponse{Fullname: "Updated Name"}, nil
		}).
		Times(1)

	got, err := d.svc.UpdateProfile(context.Background(), req, "user-1", etag)
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if got.Fullname != "Updated Name" {
		t.Errorf("Fullname = %q, want Updated Name", got.Fullname)
	}
}

func TestUpdateProfile_StaleETagAbortsTheWrite(t *testing.T) {
	d := newUserDeps(t)
	d.passthroughRC(1)

	d.userRepo.EXPECT().
		GetProfile(gomock.Any(), gomock.Eq("user-1"), gomock.Eq(true)).
		Return(domainuser.ProfileResponse{UpdatedAt: time.Now().UTC()}, nil).
		Times(1)

	// UpdateProfile must not be reached.
	_, err := d.svc.UpdateProfile(context.Background(),
		domainuser.ProfileUpdateRequest{}, "user-1", "1999-01-01T00:00:00Z")
	if !errors.Is(err, domainmedia.ErrETagValidationFailed) {
		t.Errorf("err = %v, want ErrETagValidationFailed", err)
	}
}

func TestUpdateProfile_RowLockIsTakenBeforeTheRead(t *testing.T) {
	d := newUserDeps(t)
	d.passthroughRC(1)
	now := time.Now().UTC()

	// forUpdate must be true so the read-modify-write is serialised
	d.userRepo.EXPECT().
		GetProfile(gomock.Any(), gomock.Any(), gomock.Eq(true)).
		Return(domainuser.ProfileResponse{UpdatedAt: now}, nil).
		Times(1)
	d.userRepo.EXPECT().
		UpdateProfile(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(domainuser.ProfileResponse{}, nil).
		Times(1)

	if _, err := d.svc.UpdateProfile(context.Background(),
		domainuser.ProfileUpdateRequest{}, "user-1", now.Format(time.RFC3339Nano)); err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
}

// ===========================================================================
// GetProfile / GetQuota
// ===========================================================================

func TestGetProfile(t *testing.T) {
	t.Run("existing user", func(t *testing.T) {
		d := newUserDeps(t)
		// a plain read must not take a row lock
		d.userRepo.EXPECT().
			GetProfile(gomock.Any(), gomock.Eq("user-1"), gomock.Eq(false)).
			Return(domainuser.ProfileResponse{Username: "alice", Email: "a@example.com"}, nil).
			Times(1)

		got, err := d.svc.GetProfile(context.Background(), "user-1")
		if err != nil {
			t.Fatalf("GetProfile: %v", err)
		}
		if got.Username != "alice" {
			t.Errorf("Username = %q, want alice", got.Username)
		}
	})

	t.Run("unknown user", func(t *testing.T) {
		d := newUserDeps(t)
		d.userRepo.EXPECT().GetProfile(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(domainuser.ProfileResponse{}, sql.ErrNoRows).Times(1)

		if _, err := d.svc.GetProfile(context.Background(), "nobody"); !errors.Is(err, domainauth.ErrUserNotFound) {
			t.Errorf("err = %v, want ErrUserNotFound", err)
		}
	})

	t.Run("backend outage is not a missing user", func(t *testing.T) {
		d := newUserDeps(t)
		d.userRepo.EXPECT().GetProfile(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(domainuser.ProfileResponse{}, errors.New("connection refused")).Times(1)

		_, err := d.svc.GetProfile(context.Background(), "user-1")
		if errors.Is(err, domainauth.ErrUserNotFound) {
			t.Error("a backend outage was reported as ErrUserNotFound")
		}
	})
}

func TestGetQuota_ScopesToTheRequestedUser(t *testing.T) {
	d := newUserDeps(t)
	id := uuid.New()

	d.quotaRepo.EXPECT().
		GetById(gomock.Any(), gomock.Eq(id.String())).
		Return(&domainuser.Quota{UserID: id, UsedBytes: 1024, TotalBytes: 4096, GifCount: 3, GitCount: 50}, nil).
		Times(1)

	got, err := d.svc.GetQuota(context.Background(), id.String())
	if err != nil {
		t.Fatalf("GetQuota: %v", err)
	}
	if got.UsedBytes != 1024 || got.TotalBytes != 4096 {
		t.Errorf("got %+v, want UsedBytes=1024 TotalBytes=4096", got)
	}
	if got.UserID != id {
		t.Errorf("UserID = %v, want %v", got.UserID, id)
	}
}
