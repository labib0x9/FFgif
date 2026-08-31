package share

// import (
// 	"context"
// 	"testing"
// 	"time"

// 	"github.com/google/uuid"
// 	domainauth "github.com/labib0x9/ffgif/internal/domain/auth"
// 	domainmedia "github.com/labib0x9/ffgif/internal/domain/media"
// 	domainshare "github.com/labib0x9/ffgif/internal/domain/share"
// )

// type mockAuthRepo struct{}

// func (m *mockAuthRepo) GetByEmail(ctx context.Context, email string) (domainauth.User, error) {
// 	return domainauth.User{Id: uuid.New(), Email: email}, nil
// }
// func (m *mockAuthRepo) GetById(ctx context.Context, id uuid.UUID) (domainauth.User, error) {
// 	return domainauth.User{Id: id}, nil
// }
// func (m *mockAuthRepo) Create(ctx context.Context, user domainauth.User) (domainauth.User, error) {
// 	return user, nil
// }
// func (m *mockAuthRepo) DeleteById(ctx context.Context, id uuid.UUID) error { return nil }
// func (m *mockAuthRepo) DeleteByEmail(ctx context.Context, email string) error { return nil }
// func (m *mockAuthRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passHash string) error {
// 	return nil
// }
// func (m *mockAuthRepo) SetVerified(ctx context.Context, userId uuid.UUID) error { return nil }
// func (m *mockAuthRepo) Upgrade(ctx context.Context, id string, user domainauth.User) (domainauth.User, error) {
// 	return user, nil
// }

// type mockGifRepo struct{}

// func (m *mockGifRepo) Create(ctx context.Context, gif domainmedia.Gif) error { return nil }
// func (m *mockGifRepo) Get(ctx context.Context, user_id string, status string) ([]domainmedia.GifResp, error) {
// 	return nil, nil
// }
// func (m *mockGifRepo) GetByKey(ctx context.Context, key string) (domainmedia.GifResp, error) {
// 	return domainmedia.GifResp{Key: key}, nil
// }
// func (m *mockGifRepo) GetRecents(ctx context.Context, user_id string) ([]domainmedia.GifResp, error) {
// 	return nil, nil
// }
// func (m *mockGifRepo) Delete(ctx context.Context, key string) error { return nil }
// func (m *mockGifRepo) Update(ctx context.Context, key string, gif domainmedia.GifResp) error {
// 	return nil
// }
// func (m *mockGifRepo) SaveRecent(ctx context.Context, key string) error { return nil }

// type mockShareRepo struct {
// 	createdShare bool
// }

// func (m *mockShareRepo) Create(ctx context.Context, s domainshare.Share) error {
// 	m.createdShare = true
// 	return nil
// }
// func (m *mockShareRepo) Get(ctx context.Context, user string) ([]domainshare.GifResp, error) {
// 	return []domainshare.GifResp{}, nil
// }
// func (m *mockShareRepo) Delete(ctx context.Context, id string) error               { return nil }
// func (m *mockShareRepo) Update(ctx context.Context, id string, s domainshare.Share) error { return nil }
// func (m *mockShareRepo) GetById(ctx context.Context, id string) (domainshare.GifResp, error) {
// 	return domainshare.GifResp{}, nil
// }

// func TestShareService_Create(t *testing.T) {
// 	shareRepo := &mockShareRepo{}
// 	svc := &service{
// 		authRepo:  &mockAuthRepo{},
// 		gifRepo:   &mockGifRepo{},
// 		shareRepo: shareRepo,
// 	}

// 	err := svc.Create(context.Background(), "user-1", "gif-key-123", "friend@example.com", time.Now().Add(24*time.Hour))
// 	if err != nil {
// 		t.Fatalf("expected Create to succeed, got: %v", err)
// 	}

// 	if !shareRepo.createdShare {
// 		t.Errorf("expected share to be recorded in repository")
// 	}
// }

// func TestShareService_Get(t *testing.T) {
// 	svc := &service{
// 		authRepo:  &mockAuthRepo{},
// 		gifRepo:   &mockGifRepo{},
// 		shareRepo: &mockShareRepo{},
// 	}

// 	shares, err := svc.Get(context.Background(), "user-1")
// 	if err != nil {
// 		t.Fatalf("expected Get to succeed, got: %v", err)
// 	}

// 	if shares == nil {
// 		t.Errorf("expected non-nil slice of shares")
// 	}
// }
