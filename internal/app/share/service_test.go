package share_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	appshare "github.com/labib0x9/ffgif/internal/app/share"
	domainauth "github.com/labib0x9/ffgif/internal/domain/auth"
	authmocks "github.com/labib0x9/ffgif/internal/domain/auth/mocks"
	domainmedia "github.com/labib0x9/ffgif/internal/domain/media"
	mediamocks "github.com/labib0x9/ffgif/internal/domain/media/mocks"
	domainshare "github.com/labib0x9/ffgif/internal/domain/share"
	sharemocks "github.com/labib0x9/ffgif/internal/domain/share/mocks"
	"github.com/labib0x9/ffgif/internal/port/queue"
	queuemocks "github.com/labib0x9/ffgif/internal/port/queue/mocks"
	tokenpkg "github.com/labib0x9/ffgif/pkg/token"
)

type shareDeps struct {
	authRepo  *authmocks.MockAuthRepository
	gifRepo   *mediamocks.MockGifRepository
	shareRepo *sharemocks.MockShareRepository
	queue     *queuemocks.MockQueue
	svc       appshare.Service
}

func newShareDeps(t *testing.T) *shareDeps {
	t.Helper()
	ctrl := gomock.NewController(t)
	d := &shareDeps{
		authRepo:  authmocks.NewMockAuthRepository(ctrl),
		gifRepo:   mediamocks.NewMockGifRepository(ctrl),
		shareRepo: sharemocks.NewMockShareRepository(ctrl),
		queue:     queuemocks.NewMockQueue(ctrl),
	}
	d.svc = appshare.NewService(d.authRepo, d.gifRepo, d.shareRepo, d.queue)
	return d
}

// shareMatcher asserts every field of the share.Share handed to the repository,
// so a swapped owner/recipient or a dropped expiry cannot slip through.
type shareMatcher struct {
	gifKey, ownerID, sharedWith string
	expiresAt                   time.Time
	got                         domainshare.Share
}

func (m *shareMatcher) Matches(x any) bool {
	s, ok := x.(domainshare.Share)
	if !ok {
		return false
	}
	m.got = s
	if s.GifKey != m.gifKey || s.OwnerID != m.ownerID || s.SharedWith != m.sharedWith {
		return false
	}
	return s.ExpiresAt != nil && s.ExpiresAt.Equal(m.expiresAt)
}

func (m *shareMatcher) String() string {
	return "share.Share{GifKey:" + m.gifKey + ", OwnerID:" + m.ownerID +
		", SharedWith:" + m.sharedWith + ", ExpiresAt:" + m.expiresAt.String() + "}"
}

// ===========================================================================
// Create — IDOR / BOLA surface
// ===========================================================================

// The brief lists this as a known hole ("Create never checks sharedBy is the
// gif's owner"). It is NOT: internal/app/share/create.go:30 compares the
// GetOwner result against sharedBy. This test pins that check down so a future
// refactor cannot quietly remove it.
func TestCreate_UserCannotShareAnotherUsersGif(t *testing.T) {
	d := newShareDeps(t)
	recipientID := uuid.New()

	d.authRepo.EXPECT().
		GetByEmail(gomock.Any(), gomock.Eq("friend@example.com")).
		Return(domainauth.User{Id: recipientID, Email: "friend@example.com"}, nil).
		Times(1)

	// the gif belongs to victim-user, not to attacker-user
	d.gifRepo.EXPECT().
		GetOwner(gomock.Any(), gomock.Eq("victims-gif-key")).
		Return("victim-user", nil).
		Times(1)

	// shareRepo.Create must NOT be reached — gomock fails on an unexpected call.
	err := d.svc.Create(context.Background(), "attacker-user", "victims-gif-key",
		"friend@example.com", time.Now().Add(time.Hour))

	if !errors.Is(err, domainmedia.ErrGifOwnerMismatch) {
		t.Errorf("err = %v, want ErrGifOwnerMismatch — a non-owner shared someone else's GIF", err)
	}
}

func TestCreate_PersistsExactOwnerRecipientAndExpiry(t *testing.T) {
	d := newShareDeps(t)
	recipientID := uuid.New()
	expiry := time.Now().Add(24 * time.Hour).UTC()

	d.authRepo.EXPECT().
		GetByEmail(gomock.Any(), gomock.Eq("friend@example.com")).
		Return(domainauth.User{Id: recipientID, Email: "friend@example.com"}, nil).
		Times(1)
	d.gifRepo.EXPECT().
		GetOwner(gomock.Any(), gomock.Eq("gif-key-123")).
		Return("owner-user-1", nil).
		Times(1)

	// The recipient stored must be the resolved user *id*, never the email.
	d.shareRepo.EXPECT().
		Create(gomock.Any(), &shareMatcher{
			gifKey:     "gif-key-123",
			ownerID:    "owner-user-1",
			sharedWith: recipientID.String(),
			expiresAt:  expiry,
		}).
		Return(nil).
		Times(1)

	if err := d.svc.Create(context.Background(), "owner-user-1", "gif-key-123", "friend@example.com", expiry); err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestCreate_RecipientEmailDoesNotExist(t *testing.T) {
	d := newShareDeps(t)

	d.authRepo.EXPECT().
		GetByEmail(gomock.Any(), gomock.Eq("ghost@example.com")).
		Return(domainauth.User{}, sql.ErrNoRows).
		Times(1)

	// GetOwner and Create must not be reached.
	err := d.svc.Create(context.Background(), "owner-1", "gif-1", "ghost@example.com", time.Now().Add(time.Hour))
	if !errors.Is(err, domainauth.ErrUserNotFound) {
		t.Errorf("err = %v, want ErrUserNotFound", err)
	}
}

func TestCreate_MissingGif(t *testing.T) {
	d := newShareDeps(t)

	d.authRepo.EXPECT().GetByEmail(gomock.Any(), gomock.Any()).
		Return(domainauth.User{Id: uuid.New()}, nil).Times(1)
	d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq("no-such-gif")).
		Return("", sql.ErrNoRows).Times(1)

	err := d.svc.Create(context.Background(), "owner-1", "no-such-gif", "f@example.com", time.Now().Add(time.Hour))
	if !errors.Is(err, domainmedia.ErrGifNotFound) {
		t.Errorf("err = %v, want ErrGifNotFound", err)
	}
}

// A DB outage on the ownership lookup must never be read as "not found", which
// a caller could map to a 404 and hide a real fault.
func TestCreate_OwnerLookupOutageIsNotReportedAsNotFound(t *testing.T) {
	d := newShareDeps(t)

	d.authRepo.EXPECT().GetByEmail(gomock.Any(), gomock.Any()).
		Return(domainauth.User{Id: uuid.New()}, nil).Times(1)
	d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Any()).
		Return("", errors.New("connection refused")).Times(1)

	err := d.svc.Create(context.Background(), "owner-1", "gif-1", "f@example.com", time.Now().Add(time.Hour))
	if err == nil {
		t.Fatal("a backend outage was swallowed")
	}
	if errors.Is(err, domainmedia.ErrGifNotFound) {
		t.Error("a backend outage was reported as ErrGifNotFound")
	}
}

// EXPECTED TO FAIL: internal/app/share/create.go never inspects expiresAt. A
// share whose expiry is already in the past — or exactly now() — is written to
// the DB, where `expires_at > now()` immediately filters it out. The caller is
// told the share succeeded but the recipient can never open it.
func TestCreate_RejectsNonFutureExpiry(t *testing.T) {
	cases := []struct {
		name    string
		expires time.Time
	}{
		{"expiry in the past", time.Now().Add(-time.Hour)},
		{"expiry exactly now", time.Now()},
		{"zero-value expiry", time.Time{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := newShareDeps(t)

			d.authRepo.EXPECT().GetByEmail(gomock.Any(), gomock.Any()).
				Return(domainauth.User{Id: uuid.New()}, nil).AnyTimes()
			d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Any()).
				Return("owner-1", nil).AnyTimes()
			d.shareRepo.EXPECT().
				Create(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, s domainshare.Share) error {
					t.Errorf("a share expiring at %v (not in the future) was persisted "+
						"and is invisible to the recipient from the moment it is written", s.ExpiresAt)
					return nil
				}).
				AnyTimes()

			err := d.svc.Create(context.Background(), "owner-1", "gif-1", "f@example.com", tc.expires)
			if err == nil {
				t.Error("Create accepted a non-future expiry")
			}
		})
	}
}

// A unique-constraint violation on (gif_key, shared_with) must surface as an
// error, not a panic — the repo's ON CONFLICT DO UPDATE normally absorbs it,
// but the service must stay correct if that clause is ever dropped.
func TestCreate_DuplicateShareSurfacesRepoError(t *testing.T) {
	d := newShareDeps(t)
	dup := errors.New(`pq: duplicate key value violates unique constraint "shares_gif_key_shared_with_key"`)

	d.authRepo.EXPECT().GetByEmail(gomock.Any(), gomock.Any()).
		Return(domainauth.User{Id: uuid.New()}, nil).Times(1)
	d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Any()).Return("owner-1", nil).Times(1)
	d.shareRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(dup).Times(1)

	if err := d.svc.Create(context.Background(), "owner-1", "gif-1", "f@example.com", time.Now().Add(time.Hour)); !errors.Is(err, dup) {
		t.Errorf("err = %v, want the repo's unique-violation error", err)
	}
}

// Two goroutines sharing the same gif with the same recipient must both come
// back cleanly (one succeeding, one either succeeding via upsert or returning
// the constraint error) and must never race or panic. Run with -race.
func TestCreate_ConcurrentDuplicatePairIsSafe(t *testing.T) {
	d := newShareDeps(t)
	const n = 16
	recipientID := uuid.New()

	d.authRepo.EXPECT().GetByEmail(gomock.Any(), gomock.Eq("f@example.com")).
		Return(domainauth.User{Id: recipientID}, nil).Times(n)
	d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq("gif-1")).
		Return("owner-1", nil).Times(n)

	var mu sync.Mutex
	seen := 0
	d.shareRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, s domainshare.Share) error {
			mu.Lock()
			defer mu.Unlock()
			seen++
			if seen > 1 {
				// simulate the unique constraint firing on every writer but the first
				return errors.New(`pq: duplicate key value violates unique constraint`)
			}
			return nil
		}).
		Times(n)

	expiry := time.Now().Add(time.Hour)
	errCh := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errCh <- d.svc.Create(context.Background(), "owner-1", "gif-1", "f@example.com", expiry)
		}()
	}
	wg.Wait()
	close(errCh)

	ok := 0
	for err := range errCh {
		if err == nil {
			ok++
		}
	}
	if ok != 1 {
		t.Errorf("%d of %d concurrent duplicate shares succeeded, want exactly 1", ok, n)
	}
}

// ===========================================================================
// CreateByToken
// ===========================================================================

func TestCreateByToken_NonOwnerIsRejected(t *testing.T) {
	d := newShareDeps(t)

	d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq("victims-gif")).
		Return("victim-user", nil).Times(1)

	// CreateByToken and PublishEmail must not be reached.
	_, err := d.svc.CreateByToken(context.Background(), "attacker", "victims-gif",
		"attacker@example.com", time.Now().Add(time.Hour))
	if !errors.Is(err, domainmedia.ErrGifOwnerMismatch) {
		t.Errorf("err = %v, want ErrGifOwnerMismatch", err)
	}
}

// EXPECTED TO FAIL: internal/app/share/create_by_token.go does
// `_token, _ := token.GenerateToken()` and stores the RAW token in
// share_tokens.token. That column is a bearer credential granting access to the
// GIF, so a DB read (backup, log, replica) yields working share links.
// The lookup should store sha256(token) and hash the incoming token in
// GetByToken, exactly as the signup/verify flow already does.
func TestCreateByToken_StoresHashNotRawBearerToken(t *testing.T) {
	d := newShareDeps(t)

	d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Eq("gif-1")).Return("owner-1", nil).Times(1)

	var stored string
	d.shareRepo.EXPECT().
		CreateByToken(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, s domainshare.ShareByToken) error {
			if s.GifKey != "gif-1" {
				t.Errorf("GifKey = %q, want gif-1", s.GifKey)
			}
			if s.Email != "friend@example.com" {
				t.Errorf("Email = %q, want friend@example.com", s.Email)
			}
			stored = s.Token
			return nil
		}).
		Times(1)

	d.queue.EXPECT().PublishEmail(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	returned, err := d.svc.CreateByToken(context.Background(), "owner-1", "gif-1",
		"friend@example.com", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("CreateByToken: %v", err)
	}

	if stored == returned {
		t.Errorf("the raw share token is stored verbatim in share_tokens.token (%q); "+
			"anyone who can read the table gets working share links", stored)
	}
	if want := tokenpkg.GetTokenHash(returned); stored != want {
		t.Errorf("stored token = %q, want sha256(issued token) = %q", stored, want)
	}
}

func TestCreateByToken_EmailsTheTokenToTheRecipient(t *testing.T) {
	d := newShareDeps(t)

	d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Any()).Return("owner-1", nil).Times(1)
	d.shareRepo.EXPECT().CreateByToken(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	var msg queue.EmailMessage
	d.queue.EXPECT().
		PublishEmail(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, m queue.EmailMessage) error {
			msg = m
			return nil
		}).
		Times(1)

	tok, err := d.svc.CreateByToken(context.Background(), "owner-1", "gif-1",
		"friend@example.com", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("CreateByToken: %v", err)
	}
	if msg.To != "friend@example.com" {
		t.Errorf("email To = %q, want friend@example.com", msg.To)
	}
	if msg.Name != "share" {
		t.Errorf("email job type = %q, want share", msg.Name)
	}
	if msg.Token != tok {
		t.Errorf("emailed token %q != returned token %q", msg.Token, tok)
	}
}

// If the row cannot be written there is nothing to share, so no mail may go out.
func TestCreateByToken_RepoFailureSendsNoEmail(t *testing.T) {
	d := newShareDeps(t)

	d.gifRepo.EXPECT().GetOwner(gomock.Any(), gomock.Any()).Return("owner-1", nil).Times(1)
	d.shareRepo.EXPECT().CreateByToken(gomock.Any(), gomock.Any()).
		Return(errors.New("insert failed")).Times(1)

	// PublishEmail is not expected.
	if _, err := d.svc.CreateByToken(context.Background(), "owner-1", "gif-1",
		"f@example.com", time.Now().Add(time.Hour)); err == nil {
		t.Fatal("want an error when the share row cannot be written")
	}
}

// ===========================================================================
// Delete
// ===========================================================================

func TestDelete(t *testing.T) {
	tests := []struct {
		name         string
		ownerFromDB  string
		ownerErr     error
		callerID     string
		wantErr      error
		wantDelete   bool
		deleteReturn error
	}{
		{
			name:        "owner deletes their own share",
			ownerFromDB: "owner-1",
			callerID:    "owner-1",
			wantDelete:  true,
		},
		{
			name:        "a third party cannot delete someone else's share",
			ownerFromDB: "owner-1",
			callerID:    "attacker",
			wantErr:     domainshare.ErrNotAuthorized,
		},
		{
			name:     "deleting an already-deleted share is reported as not found",
			ownerErr: sql.ErrNoRows,
			callerID: "owner-1",
			wantErr:  domainshare.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := newShareDeps(t)

			d.shareRepo.EXPECT().
				GetOwner(gomock.Any(), gomock.Eq("recipient-1"), gomock.Eq("gif-1")).
				Return(tc.ownerFromDB, tc.ownerErr).
				Times(1)

			if tc.wantDelete {
				d.shareRepo.EXPECT().
					Delete(gomock.Any(), gomock.Eq("gif-1"), gomock.Eq("recipient-1")).
					Return(tc.deleteReturn).
					Times(1)
			}

			err := d.svc.Delete(context.Background(), tc.callerID, "gif-1", "recipient-1")
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("Delete: %v", err)
			}
		})
	}
}

// A lookup outage must not be collapsed into "not found", which would make the
// caller believe the share is already gone.
func TestDelete_LookupOutageIsNotReportedAsNotFound(t *testing.T) {
	d := newShareDeps(t)

	d.shareRepo.EXPECT().
		GetOwner(gomock.Any(), gomock.Any(), gomock.Any()).
		Return("", errors.New("connection refused")).
		Times(1)

	err := d.svc.Delete(context.Background(), "owner-1", "gif-1", "recipient-1")
	if err == nil {
		t.Fatal("a backend outage was swallowed")
	}
	if errors.Is(err, domainshare.ErrNotFound) {
		t.Error("a backend outage was reported as ErrNotFound")
	}
}

// ===========================================================================
// Get / GetByToken
// ===========================================================================

func TestGet_PassesTheCallersOwnIdToTheRepository(t *testing.T) {
	d := newShareDeps(t)
	want := []domainshare.GifResponse{{GifKey: "k1", OwnerID: "owner-1", Name: "one.gif"}}

	// The user id must be forwarded verbatim; scoping is the repository's job.
	d.shareRepo.EXPECT().
		Get(gomock.Any(), gomock.Eq("user-1")).
		Return(want, nil).
		Times(1)

	got, err := d.svc.Get(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got) != 1 || got[0].GifKey != "k1" {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestGetByToken_ForwardsTokenAndPropagatesMiss(t *testing.T) {
	t.Run("hit", func(t *testing.T) {
		d := newShareDeps(t)
		d.shareRepo.EXPECT().
			GetByToken(gomock.Any(), gomock.Eq("share-token")).
			Return(domainshare.GifTokenResponse{GifKey: "gif-1", Name: "one.gif"}, nil).
			Times(1)

		resp, err := d.svc.GetByToken(context.Background(), "share-token")
		if err != nil {
			t.Fatalf("GetByToken: %v", err)
		}
		if resp.GifKey != "gif-1" {
			t.Errorf("GifKey = %q, want gif-1", resp.GifKey)
		}
	})

	t.Run("expired or unknown token", func(t *testing.T) {
		d := newShareDeps(t)
		d.shareRepo.EXPECT().
			GetByToken(gomock.Any(), gomock.Eq("expired-token")).
			Return(domainshare.GifTokenResponse{}, sql.ErrNoRows).
			Times(1)

		resp, err := d.svc.GetByToken(context.Background(), "expired-token")
		if err == nil {
			t.Fatal("an expired share token was accepted")
		}
		if resp.Url != "" {
			t.Errorf("a GIF url (%q) was handed out for an expired token", resp.Url)
		}
	})
}
