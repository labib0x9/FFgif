package share_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	appshare "github.com/labib0x9/ffgif/internal/app/share"
	domainauth "github.com/labib0x9/ffgif/internal/domain/auth"
	domainmedia "github.com/labib0x9/ffgif/internal/domain/media"
	domainshare "github.com/labib0x9/ffgif/internal/domain/share"
	"github.com/labib0x9/ffgif/internal/port/queue"
	amqp "github.com/rabbitmq/amqp091-go"
)

type mockAuthRepo struct {
	getByEmailFunc func(ctx context.Context, email string) (domainauth.User, error)
}

func (m *mockAuthRepo) GetByEmail(ctx context.Context, email string) (domainauth.User, error) {
	if m.getByEmailFunc != nil {
		return m.getByEmailFunc(ctx, email)
	}
	return domainauth.User{Id: uuid.New(), Email: email}, nil
}
func (m *mockAuthRepo) GetById(ctx context.Context, id uuid.UUID) (domainauth.User, error) {
	return domainauth.User{Id: id}, nil
}
func (m *mockAuthRepo) Create(ctx context.Context, user domainauth.User) (domainauth.User, error) {
	return user, nil
}
func (m *mockAuthRepo) DeleteById(ctx context.Context, id uuid.UUID) error    { return nil }
func (m *mockAuthRepo) DeleteByEmail(ctx context.Context, email string) error { return nil }
func (m *mockAuthRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passHash string) error {
	return nil
}
func (m *mockAuthRepo) SetVerified(ctx context.Context, userId uuid.UUID) error { return nil }
func (m *mockAuthRepo) Upgrade(ctx context.Context, id string, user domainauth.User) (domainauth.User, error) {
	return user, nil
}

type mockGifRepo struct {
	getByKeyFunc func(ctx context.Context, key string, forUpdate bool) (domainmedia.GifResponse, error)
	getOwnerFunc func(ctx context.Context, key string) (string, error)
}

func (m *mockGifRepo) Create(ctx context.Context, gif domainmedia.Gif) error { return nil }
func (m *mockGifRepo) Get(ctx context.Context, user_id string, status string) ([]domainmedia.GifResponse, error) {
	return nil, nil
}
func (m *mockGifRepo) GetByKey(ctx context.Context, key string, forUpdate bool) (domainmedia.GifResponse, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key, forUpdate)
	}
	return domainmedia.GifResponse{Key: key}, nil
}
func (m *mockGifRepo) GetRecents(ctx context.Context, user_id string) ([]domainmedia.GifResponse, error) {
	return nil, nil
}
func (m *mockGifRepo) Delete(ctx context.Context, key string) error { return nil }
func (m *mockGifRepo) Update(ctx context.Context, key string, req domainmedia.GifUpdateRequest) (domainmedia.GifResponse, error) {
	return domainmedia.GifResponse{}, nil
}
func (m *mockGifRepo) SaveRecent(ctx context.Context, key string) error { return nil }
func (m *mockGifRepo) GetOwner(ctx context.Context, key string) (string, error) {
	if m.getOwnerFunc != nil {
		return m.getOwnerFunc(ctx, key)
	}
	return "", nil
}

type mockShareRepo struct {
	createdShare      *domainshare.Share
	createdTokenShare *domainshare.ShareByToken
	createFunc        func(ctx context.Context, s domainshare.Share) error
	createByTokenFunc func(ctx context.Context, gif domainshare.ShareByToken) error
	getFunc           func(ctx context.Context, user string) ([]domainshare.GifResponse, error)
	getByTokenFunc    func(ctx context.Context, token string) (domainshare.GifTokenResponse, error)
	getOwnerFunc      func(ctx context.Context, user string, key string) (string, error)
	deleteFunc        func(ctx context.Context, key, shareWithId string) error
}

func (m *mockShareRepo) Create(ctx context.Context, s domainshare.Share) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, s)
	}
	m.createdShare = &s
	return nil
}
func (m *mockShareRepo) CreateByToken(ctx context.Context, gif domainshare.ShareByToken) error {
	if m.createByTokenFunc != nil {
		return m.createByTokenFunc(ctx, gif)
	}
	m.createdTokenShare = &gif
	return nil
}
func (m *mockShareRepo) Get(ctx context.Context, user string) ([]domainshare.GifResponse, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, user)
	}
	return []domainshare.GifResponse{}, nil
}
func (m *mockShareRepo) GetByToken(ctx context.Context, token string) (domainshare.GifTokenResponse, error) {
	if m.getByTokenFunc != nil {
		return m.getByTokenFunc(ctx, token)
	}
	return domainshare.GifTokenResponse{}, nil
}
func (m *mockShareRepo) GetOwner(ctx context.Context, user string, key string) (string, error) {
	if m.getOwnerFunc != nil {
		return m.getOwnerFunc(ctx, user, key)
	}
	return "", nil
}
func (m *mockShareRepo) Delete(ctx context.Context, key, shareWithId string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, key, shareWithId)
	}
	return nil
}

type mockQueue struct {
	publishedEmail   *queue.EmailMessage
	publishEmailFunc func(ctx context.Context, msg queue.EmailMessage) error
}

func (m *mockQueue) PublishEmail(ctx context.Context, msg queue.EmailMessage) error {
	if m.publishEmailFunc != nil {
		return m.publishEmailFunc(ctx, msg)
	}
	m.publishedEmail = &msg
	return nil
}
func (m *mockQueue) PublishVideo(ctx context.Context, msg queue.VideoMessage) error           { return nil }
func (m *mockQueue) PublishSaveVideo(ctx context.Context, msg queue.SaveVideoMessage) error       { return nil }
func (m *mockQueue) PublishRetrySaveVideo(ctx context.Context, msg queue.SaveVideoMessage) error  { return nil }
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

func TestShareService_Create_Success(t *testing.T) {
	recipientID := uuid.New()
	authRepo := &mockAuthRepo{
		getByEmailFunc: func(ctx context.Context, email string) (domainauth.User, error) {
			return domainauth.User{Id: recipientID, Email: email}, nil
		},
	}
	gifRepo := &mockGifRepo{
		getOwnerFunc: func(ctx context.Context, key string) (string, error) {
			return "owner-user-1", nil
		},
	}
	shareRepo := &mockShareRepo{}
	q := &mockQueue{}

	svc := appshare.NewService(authRepo, gifRepo, shareRepo, q)

	expiry := time.Now().Add(24 * time.Hour)
	err := svc.Create(context.Background(), "owner-user-1", "gif-key-123", "friend@example.com", expiry)
	if err != nil {
		t.Fatalf("expected Create to succeed, got: %v", err)
	}

	if shareRepo.createdShare == nil {
		t.Fatal("expected share record to be created in repository")
	}
	if shareRepo.createdShare.OwnerID != "owner-user-1" {
		t.Errorf("expected ownerID owner-user-1, got %s", shareRepo.createdShare.OwnerID)
	}
	if shareRepo.createdShare.SharedWith != recipientID.String() {
		t.Errorf("expected sharedWith %s, got %s", recipientID.String(), shareRepo.createdShare.SharedWith)
	}
}

func TestShareService_Create_RecipientNotFound(t *testing.T) {
	authRepo := &mockAuthRepo{
		getByEmailFunc: func(ctx context.Context, email string) (domainauth.User, error) {
			return domainauth.User{}, errors.New("user not found")
		},
	}
	svc := appshare.NewService(authRepo, &mockGifRepo{}, &mockShareRepo{}, &mockQueue{})

	err := svc.Create(context.Background(), "owner-1", "gif-123", "missing@example.com", time.Now())
	if !errors.Is(err, domainauth.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestShareService_Create_GifNotFound(t *testing.T) {
	gifRepo := &mockGifRepo{
		getOwnerFunc: func(ctx context.Context, key string) (string, error) {
			return "", sql.ErrNoRows
		},
	}
	svc := appshare.NewService(&mockAuthRepo{}, gifRepo, &mockShareRepo{}, &mockQueue{})

	err := svc.Create(context.Background(), "owner-1", "missing-gif", "friend@example.com", time.Now())
	if !errors.Is(err, domainmedia.ErrGifNotFound) {
		t.Errorf("expected ErrGifNotFound, got %v", err)
	}
}

func TestShareService_Create_RepoError(t *testing.T) {
	gifRepo := &mockGifRepo{
		getOwnerFunc: func(ctx context.Context, key string) (string, error) {
			return "owner-1", nil
		},
	}
	shareRepo := &mockShareRepo{
		createFunc: func(ctx context.Context, s domainshare.Share) error {
			return errors.New("db insert failure")
		},
	}
	svc := appshare.NewService(&mockAuthRepo{}, gifRepo, shareRepo, &mockQueue{})

	err := svc.Create(context.Background(), "owner-1", "gif-1", "friend@example.com", time.Now())
	if err == nil {
		t.Fatal("expected error from share repo, got nil")
	}
}

func TestShareService_Get_Success(t *testing.T) {
	shareRepo := &mockShareRepo{
		getFunc: func(ctx context.Context, user string) ([]domainshare.GifResponse, error) {
			return []domainshare.GifResponse{
				{Name: "Shared Gif 1", GifKey: "key-1", OwnerID: "owner-1"},
			}, nil
		},
	}
	svc := appshare.NewService(&mockAuthRepo{}, &mockGifRepo{}, shareRepo, &mockQueue{})

	shares, err := svc.Get(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("expected Get to succeed, got: %v", err)
	}
	if len(shares) != 1 {
		t.Errorf("expected 1 share, got %d", len(shares))
	}
}

func TestShareService_Delete_Success(t *testing.T) {
	deleted := false
	shareRepo := &mockShareRepo{
		getOwnerFunc: func(ctx context.Context, user, key string) (string, error) {
			return "owner-1", nil
		},
		deleteFunc: func(ctx context.Context, key, shareWithId string) error {
			deleted = true
			return nil
		},
	}
	svc := appshare.NewService(&mockAuthRepo{}, &mockGifRepo{}, shareRepo, &mockQueue{})

	err := svc.Delete(context.Background(), "owner-1", "gif-1", "user-2")
	if err != nil {
		t.Fatalf("expected Delete to succeed, got: %v", err)
	}
	if !deleted {
		t.Fatal("expected share repo Delete to be called")
	}
}

func TestShareService_Delete_NotFound(t *testing.T) {
	shareRepo := &mockShareRepo{
		getOwnerFunc: func(ctx context.Context, user, key string) (string, error) {
			return "", sql.ErrNoRows
		},
	}
	svc := appshare.NewService(&mockAuthRepo{}, &mockGifRepo{}, shareRepo, &mockQueue{})

	err := svc.Delete(context.Background(), "owner-1", "gif-1", "user-2")
	if !errors.Is(err, domainshare.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestShareService_Delete_NotAuthorized(t *testing.T) {
	shareRepo := &mockShareRepo{
		getOwnerFunc: func(ctx context.Context, user, key string) (string, error) {
			return "actual-owner-id", nil
		},
	}
	svc := appshare.NewService(&mockAuthRepo{}, &mockGifRepo{}, shareRepo, &mockQueue{})

	err := svc.Delete(context.Background(), "attacker-user-id", "gif-1", "user-2")
	if !errors.Is(err, domainshare.ErrNotAuthorized) {
		t.Errorf("expected ErrNotAuthorized, got %v", err)
	}
}

func TestShareService_Delete_GetOwnerError(t *testing.T) {
	shareRepo := &mockShareRepo{
		getOwnerFunc: func(ctx context.Context, user, key string) (string, error) {
			return "", errors.New("db connection failure")
		},
	}
	svc := appshare.NewService(&mockAuthRepo{}, &mockGifRepo{}, shareRepo, &mockQueue{})

	err := svc.Delete(context.Background(), "owner-1", "gif-1", "user-2")
	if err == nil || errors.Is(err, domainshare.ErrNotFound) {
		t.Errorf("expected db error, got: %v", err)
	}
}

func TestShareService_CreateByToken_Success(t *testing.T) {
	gifRepo := &mockGifRepo{
		getOwnerFunc: func(ctx context.Context, key string) (string, error) {
			return "owner-1", nil
		},
	}
	shareRepo := &mockShareRepo{}
	q := &mockQueue{}

	svc := appshare.NewService(&mockAuthRepo{}, gifRepo, shareRepo, q)

	expiry := time.Now().Add(24 * time.Hour)
	token, err := svc.CreateByToken(context.Background(), "owner-1", "gif-1", "friend@example.com", expiry)
	if err != nil {
		t.Fatalf("expected CreateByToken to succeed, got %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if shareRepo.createdTokenShare == nil {
		t.Fatal("expected token share to be created in repo")
	}
	if q.publishedEmail == nil {
		t.Fatal("expected email to be published")
	}
	if q.publishedEmail.To != "friend@example.com" {
		t.Errorf("expected email To friend@example.com, got %s", q.publishedEmail.To)
	}
}

func TestShareService_CreateByToken_GifNotFound(t *testing.T) {
	gifRepo := &mockGifRepo{
		getOwnerFunc: func(ctx context.Context, key string) (string, error) {
			return "", sql.ErrNoRows
		},
	}
	svc := appshare.NewService(&mockAuthRepo{}, gifRepo, &mockShareRepo{}, &mockQueue{})

	_, err := svc.CreateByToken(context.Background(), "owner-1", "missing-gif", "friend@example.com", time.Now())
	if !errors.Is(err, domainmedia.ErrGifNotFound) {
		t.Errorf("expected ErrGifNotFound, got %v", err)
	}
}

func TestShareService_CreateByToken_OwnerMismatch(t *testing.T) {
	gifRepo := &mockGifRepo{
		getOwnerFunc: func(ctx context.Context, key string) (string, error) {
			return "actual-owner", nil
		},
	}
	svc := appshare.NewService(&mockAuthRepo{}, gifRepo, &mockShareRepo{}, &mockQueue{})

	_, err := svc.CreateByToken(context.Background(), "attacker", "gif-1", "friend@example.com", time.Now())
	if !errors.Is(err, domainmedia.ErrGifOwnerMismatch) {
		t.Errorf("expected ErrGifOwnerMismatch, got %v", err)
	}
}

func TestShareService_GetByToken_Success(t *testing.T) {
	shareRepo := &mockShareRepo{
		getByTokenFunc: func(ctx context.Context, token string) (domainshare.GifTokenResponse, error) {
			return domainshare.GifTokenResponse{
				GifKey: "gif-1",
				Name:   "sample.gif",
			}, nil
		},
	}
	svc := appshare.NewService(&mockAuthRepo{}, &mockGifRepo{}, shareRepo, &mockQueue{})

	resp, err := svc.GetByToken(context.Background(), "valid-token")
	if err != nil {
		t.Fatalf("expected GetByToken to succeed, got %v", err)
	}
	if resp.GifKey != "gif-1" {
		t.Errorf("expected GifKey gif-1, got %s", resp.GifKey)
	}
}

