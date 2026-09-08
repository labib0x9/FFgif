package share_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	appshare "github.com/labib0x9/ffgif/internal/app/share"
	domainauth "github.com/labib0x9/ffgif/internal/domain/auth"
	authMocks "github.com/labib0x9/ffgif/internal/domain/auth/mocks"
	domainmedia "github.com/labib0x9/ffgif/internal/domain/media"
	mediaMocks "github.com/labib0x9/ffgif/internal/domain/media/mocks"
	domainshare "github.com/labib0x9/ffgif/internal/domain/share"
	shareMocks "github.com/labib0x9/ffgif/internal/domain/share/mocks"
	"github.com/labib0x9/ffgif/internal/port/queue"
	queueMocks "github.com/labib0x9/ffgif/internal/port/queue/mocks"
)

type shareTestDeps struct {
	ctrl      *gomock.Controller
	authRepo  *authMocks.MockAuthRepository
	gifRepo   *mediaMocks.MockGifRepository
	shareRepo *shareMocks.MockShareRepository
	queue     *queueMocks.MockQueue
	svc       appshare.Service
}

func newShareTestDeps(t *testing.T) *shareTestDeps {
	ctrl := gomock.NewController(t)
	deps := &shareTestDeps{
		ctrl:      ctrl,
		authRepo:  authMocks.NewMockAuthRepository(ctrl),
		gifRepo:   mediaMocks.NewMockGifRepository(ctrl),
		shareRepo: shareMocks.NewMockShareRepository(ctrl),
		queue:     queueMocks.NewMockQueue(ctrl),
	}
	deps.svc = appshare.NewService(
		deps.authRepo,
		deps.gifRepo,
		deps.shareRepo,
		deps.queue,
	)
	return deps
}

func TestShareService_Create(t *testing.T) {
	t.Run("success: owner shares gif with registered user", func(t *testing.T) {
		d := newShareTestDeps(t)
		ctx := context.Background()

		ownerID := uuid.New().String()
		recipientID := uuid.New()
		recipientEmail := "friend@example.com"
		gifKey := "my-gif-key.gif"
		expiresAt := time.Now().Add(24 * time.Hour)

		d.authRepo.EXPECT().GetByEmail(gomock.Any(), gomock.Eq(recipientEmail)).
			Return(domainauth.User{Id: recipientID, Email: recipientEmail}, nil).Times(1)

		d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq(gifKey)).
			Return(ownerID, nil).Times(1)

		d.shareRepo.EXPECT().Create(gomock.Any(), gomock.Cond(func(x any) bool {
			s, ok := x.(domainshare.Share)
			return ok && s.GifKey == gifKey && s.OwnerID == ownerID && s.SharedWith == recipientID.String() && s.ExpiresAt.Equal(expiresAt)
		})).Return(nil).Times(1)

		err := d.svc.Create(ctx, ownerID, gifKey, recipientEmail, expiresAt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("non-owner attempts to share: returns ErrGifOwnerMismatch", func(t *testing.T) {
		d := newShareTestDeps(t)
		ctx := context.Background()

		actualOwnerID := uuid.New().String()
		attackerID := uuid.New().String()
		recipientID := uuid.New()
		recipientEmail := "friend@example.com"
		gifKey := "target-gif.gif"
		expiresAt := time.Now().Add(24 * time.Hour)

		d.authRepo.EXPECT().GetByEmail(gomock.Any(), gomock.Eq(recipientEmail)).
			Return(domainauth.User{Id: recipientID, Email: recipientEmail}, nil).Times(1)

		d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq(gifKey)).
			Return(actualOwnerID, nil).Times(1)

		err := d.svc.Create(ctx, attackerID, gifKey, recipientEmail, expiresAt)
		if !errors.Is(err, domainmedia.ErrGifOwnerMismatch) {
			t.Fatalf("expected ErrGifOwnerMismatch when non-owner tries to share, got %v", err)
		}
	})

	t.Run("recipient email not found: returns ErrUserNotFound", func(t *testing.T) {
		d := newShareTestDeps(t)
		ctx := context.Background()

		d.authRepo.EXPECT().GetByEmail(gomock.Any(), "nonexistent@example.com").
			Return(domainauth.User{}, sql.ErrNoRows).Times(1)

		err := d.svc.Create(ctx, "owner-id", "gif-key", "nonexistent@example.com", time.Now().Add(1*time.Hour))
		if !errors.Is(err, domainauth.ErrUserNotFound) {
			t.Fatalf("expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("gif not found: returns ErrGifNotFound", func(t *testing.T) {
		d := newShareTestDeps(t)
		ctx := context.Background()

		recipientID := uuid.New()
		d.authRepo.EXPECT().GetByEmail(gomock.Any(), "friend@example.com").
			Return(domainauth.User{Id: recipientID, Email: "friend@example.com"}, nil).Times(1)

		d.gifRepo.EXPECT().GetOwner(gomock.Any(), "missing-gif").
			Return("", sql.ErrNoRows).Times(1)

		err := d.svc.Create(ctx, "owner-id", "missing-gif", "friend@example.com", time.Now().Add(1*time.Hour))
		if !errors.Is(err, domainmedia.ErrGifNotFound) {
			t.Fatalf("expected ErrGifNotFound, got %v", err)
		}
	})
}

func TestShareService_CreateByToken(t *testing.T) {
	t.Run("success: owner generates public/token share and publishes email", func(t *testing.T) {
		d := newShareTestDeps(t)
		ctx := context.Background()

		ownerID := uuid.New().String()
		gifKey := "public-share-gif.gif"
		email := "guest@example.com"
		expiresAt := time.Now().Add(48 * time.Hour)

		d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq(gifKey)).
			Return(ownerID, nil).Times(1)

		d.shareRepo.EXPECT().CreateByToken(gomock.Any(), gomock.Cond(func(x any) bool {
			s, ok := x.(domainshare.ShareByToken)
			return ok && s.GifKey == gifKey && s.Email == email && s.Token != "" && s.ExpiresAt.Equal(expiresAt)
		})).Return(nil).Times(1)

		d.queue.EXPECT().PublishEmail(gomock.Any(), gomock.Cond(func(x any) bool {
			m, ok := x.(queue.EmailMessage)
			return ok && m.To == email && m.Name == "share" && m.Token != ""
		})).Return(nil).Times(1)

		token, err := d.svc.CreateByToken(ctx, ownerID, gifKey, email, expiresAt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token == "" {
			t.Errorf("expected non-empty share token")
		}
	})

	t.Run("non-owner attempts public share: returns ErrGifOwnerMismatch", func(t *testing.T) {
		d := newShareTestDeps(t)
		ctx := context.Background()

		ownerID := "owner-user"
		attackerID := "attacker-user"

		d.gifRepo.EXPECT().GetOwner(gomock.Any(), "some-gif.gif").
			Return(ownerID, nil).Times(1)

		_, err := d.svc.CreateByToken(ctx, attackerID, "some-gif.gif", "guest@example.com", time.Now().Add(1*time.Hour))
		if !errors.Is(err, domainmedia.ErrGifOwnerMismatch) {
			t.Fatalf("expected ErrGifOwnerMismatch, got %v", err)
		}
	})
}

func TestShareService_Delete(t *testing.T) {
	t.Run("success: revokes share for user", func(t *testing.T) {
		d := newShareTestDeps(t)
		ctx := context.Background()

		ownerID := "owner-id"
		gifKey := "my-gif.gif"
		shareWithID := "shared-user-id"

		d.shareRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq(shareWithID), gomock.Eq(gifKey)).
			Return(ownerID, nil).Times(1)

		d.shareRepo.EXPECT().Delete(gomock.Any(), gomock.Eq(gifKey), gomock.Eq(shareWithID)).
			Return(nil).Times(1)

		err := d.svc.Delete(ctx, ownerID, gifKey, shareWithID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("unauthorized: non-owner cannot revoke share", func(t *testing.T) {
		d := newShareTestDeps(t)
		ctx := context.Background()

		actualOwnerID := "owner-id"
		attackerID := "attacker-id"
		gifKey := "my-gif.gif"
		shareWithID := "shared-user-id"

		d.shareRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq(shareWithID), gomock.Eq(gifKey)).
			Return(actualOwnerID, nil).Times(1)

		err := d.svc.Delete(ctx, attackerID, gifKey, shareWithID)
		if !errors.Is(err, domainshare.ErrNotAuthorized) {
			t.Fatalf("expected ErrNotAuthorized, got %v", err)
		}
	})
}

func TestShareService_Get(t *testing.T) {
	t.Run("success: lists shared gifs for user", func(t *testing.T) {
		d := newShareTestDeps(t)
		ctx := context.Background()

		userID := "user-123"
		expected := []domainshare.GifResponse{
			{ID: "1", GifKey: "key-1", Name: "gif1.gif"},
			{ID: "2", GifKey: "key-2", Name: "gif2.gif"},
		}

		d.shareRepo.EXPECT().Get(gomock.Any(), gomock.Eq(userID)).
			Return(expected, nil).Times(1)

		res, err := d.svc.Get(ctx, userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res) != 2 {
			t.Errorf("expected 2 shared gifs, got %d", len(res))
		}
	})
}

func TestShareService_GetByToken(t *testing.T) {
	t.Run("success: retrieves shared gif by token", func(t *testing.T) {
		d := newShareTestDeps(t)
		ctx := context.Background()

		token := "share-token-xyz"
		expected := domainshare.GifTokenResponse{
			GifKey: "shared-key",
			Name:   "awesome.gif",
			Url:    "https://storage/shared-key.gif",
		}

		d.shareRepo.EXPECT().GetByToken(gomock.Any(), gomock.Eq(token)).
			Return(expected, nil).Times(1)

		res, err := d.svc.GetByToken(ctx, token)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.GifKey != "shared-key" {
			t.Errorf("expected GifKey 'shared-key', got '%s'", res.GifKey)
		}
	})
}
