package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"go.uber.org/mock/gomock"

	"github.com/labib0x9/ffgif/config"
	portcache "github.com/labib0x9/ffgif/internal/port/cache"
	cachemocks "github.com/labib0x9/ffgif/internal/port/cache/mocks"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
)

const mwSecret = "test-secret-key-1234"

type mwHarness struct {
	cache *cachemocks.MockCache
	jwt   *jwtpkg.Jwt
	mw    *middleware.Middlewares
}

func newMwHarness(t *testing.T) *mwHarness {
	t.Helper()
	ctrl := gomock.NewController(t)
	h := &mwHarness{
		cache: cachemocks.NewMockCache(ctrl),
		jwt:   jwtpkg.NewJwt([]byte(mwSecret)),
	}
	h.mw = middleware.NewMiddlewares(&config.Config{}, h.cache, *h.jwt)
	return h
}

// protected wraps a handler that records whether the request got through.
func (h *mwHarness) protected(reached *int32) http.Handler {
	return h.mw.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(reached, 1)
		w.WriteHeader(http.StatusOK)
	}))
}

func doAuth(handler http.Handler, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// ===========================================================================
// Blocklist — the fail-open hole
// ===========================================================================

// EXPECTED TO FAIL: internal/transport/http/middleware/auth.go decides
// blocklist membership with
//
//	if _, err := m.cache.Get(r.Context(), key); err == nil { /* blocked */ }
//
// Only a nil error means "blocked". Every other outcome — including
// "connection refused", a timeout, or an auth failure against Redis — is read
// as "not blocklisted" and the request is let through. During a Redis outage
// every token the users logged out with silently becomes valid again, for as
// long as the outage lasts.
//
// Contract: a cache MISS means not-blocklisted; a cache FAULT must fail closed.
func TestAuth_BlocklistFailsClosedWhenRedisIsDown(t *testing.T) {
	outages := []struct {
		name string
		err  error
	}{
		{"connection refused", errors.New("dial tcp 127.0.0.1:6379: connect: connection refused")},
		{"i/o timeout", errors.New("read tcp 127.0.0.1:6379: i/o timeout")},
		{"auth failure", errors.New("NOAUTH Authentication required")},
	}

	for _, tc := range outages {
		t.Run(tc.name, func(t *testing.T) {
			h := newMwHarness(t)
			token, err := h.jwt.Create("John Doe", "user-1", "john@example.com", "user")
			if err != nil {
				t.Fatalf("Create token: %v", err)
			}

			// The token IS on the blocklist (the user logged out), but Redis is
			// unreachable so the middleware cannot see that.
			h.cache.EXPECT().
				Get(gomock.Any(), gomock.Eq("token_blocklist:"+token)).
				Return("", tc.err).
				Times(1)

			var reached int32
			rec := doAuth(h.protected(&reached), token)

			if atomic.LoadInt32(&reached) != 0 {
				t.Errorf("a revoked token reached the protected handler because the "+
					"blocklist lookup failed with %q; the middleware fails OPEN", tc.err)
			}
			if rec.Code == http.StatusOK {
				t.Errorf("status = %d, want 401 or 503 — a blocklist backend fault "+
					"must not be treated as 'not blocklisted'", rec.Code)
			}
		})
	}
}

// The counterpart: a genuine miss is the normal case and must let the request
// through. This is what distinguishes "fail closed" from "reject everything".
func TestAuth_CacheMissMeansNotBlocklisted(t *testing.T) {
	h := newMwHarness(t)
	token, _ := h.jwt.Create("John Doe", "user-1", "john@example.com", "user")

	h.cache.EXPECT().
		Get(gomock.Any(), gomock.Eq("token_blocklist:"+token)).
		Return("", portcache.ErrCacheMiss).
		Times(1)

	var reached int32
	rec := doAuth(h.protected(&reached), token)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 for a token that is simply not blocklisted", rec.Code)
	}
	if atomic.LoadInt32(&reached) != 1 {
		t.Error("the request did not reach the protected handler")
	}
}

func TestAuth_BlocklistedTokenIsRejected(t *testing.T) {
	h := newMwHarness(t)
	token, _ := h.jwt.Create("John Doe", "user-1", "john@example.com", "user")

	h.cache.EXPECT().
		Get(gomock.Any(), gomock.Eq("token_blocklist:"+token)).
		Return("1", nil).
		Times(1)

	var reached int32
	rec := doAuth(h.protected(&reached), token)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	if atomic.LoadInt32(&reached) != 0 {
		t.Error("a blocklisted token reached the protected handler")
	}
	if !strings.Contains(rec.Header().Get("WWW-Authenticate"), "blocklist") {
		t.Errorf("WWW-Authenticate = %q, want it to mention the blocklist",
			rec.Header().Get("WWW-Authenticate"))
	}
}

// ===========================================================================
// Token validation
// ===========================================================================

func TestAuth_ValidTokenPopulatesTheAuthContext(t *testing.T) {
	h := newMwHarness(t)
	token, _ := h.jwt.Create("John Doe", "user-uuid-1", "john@example.com", "user")

	h.cache.EXPECT().
		Get(gomock.Any(), gomock.Eq("token_blocklist:"+token)).
		Return("", portcache.ErrCacheMiss).
		Times(1)

	var gotUser, gotToken string
	handler := h.mw.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = httputil.GetUserId(r.Context())
		gotToken, _ = httputil.GetAuthorizationHeader(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	if rec := doAuth(handler, token); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if gotUser != "user-uuid-1" {
		t.Errorf("user id in context = %q, want user-uuid-1", gotUser)
	}
	if gotToken != token {
		t.Error("the raw token was not put in the request context")
	}
}

// Every one of these must be refused before the blocklist is even consulted,
// so the mocked cache expects no calls at all.
func TestAuth_MalformedAndForgedTokensAreRefused(t *testing.T) {
	h := newMwHarness(t)
	valid, _ := h.jwt.Create("John Doe", "user-1", "john@example.com", "user")

	// signed with a different key
	foreign := jwtpkg.NewJwt([]byte("attacker-controlled-key"))
	forged, _ := foreign.Create("John Doe", "user-1", "john@example.com", "admin")

	// alg=none
	noneTok, err := gojwt.NewWithClaims(gojwt.SigningMethodNone, jwtpkg.Payload{
		Fullname: "John Doe", Email: "john@example.com", Role: "admin",
		RegisteredClaims: gojwt.RegisteredClaims{
			Subject:   "user-1",
			ExpiresAt: gojwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}).SignedString(gojwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("build alg=none token: %v", err)
	}

	// already expired
	expired, err := gojwt.NewWithClaims(gojwt.SigningMethodHS256, jwtpkg.Payload{
		Fullname: "John Doe", Email: "john@example.com", Role: "user",
		RegisteredClaims: gojwt.RegisteredClaims{
			Subject:   "user-1",
			IssuedAt:  gojwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: gojwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}).SignedString([]byte(mwSecret))
	if err != nil {
		t.Fatalf("build expired token: %v", err)
	}

	// payload tampered: flip a character in the claims segment, signature untouched
	parts := strings.Split(valid, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected token shape: %q", valid)
	}
	tamperedPayload := parts[0] + "." + mutate(parts[1]) + "." + parts[2]
	tamperedSig := parts[0] + "." + parts[1] + "." + mutate(parts[2])

	tests := []struct {
		name  string
		token string
	}{
		{"tampered payload", tamperedPayload},
		{"tampered signature", tamperedSig},
		{"signed with a foreign key", forged},
		{"alg=none", noneTok},
		{"expired", expired},
		{"not a jwt at all", "gibberish"},
		{"empty bearer value", ""},
		{"only two segments", parts[0] + "." + parts[1]},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var reached int32
			rec := doAuth(h.protected(&reached), tc.token)

			if atomic.LoadInt32(&reached) != 0 {
				t.Errorf("a %s token reached the protected handler", tc.name)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", rec.Code)
			}
		})
	}
}

func TestAuth_AuthorizationHeaderShape(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{"missing header", ""},
		{"no scheme", "abc.def.ghi"},
		{"wrong scheme", "Basic dXNlcjpwYXNz"},
		{"lowercase bearer is not accepted", "bearer abc.def.ghi"},
		{"scheme only", "Bearer"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newMwHarness(t)
			var reached int32
			handler := h.protected(&reached)

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", rec.Code)
			}
			if atomic.LoadInt32(&reached) != 0 {
				t.Error("the request reached the protected handler")
			}
		})
	}
}

// ===========================================================================
// Concurrency: logout racing against in-flight requests
// ===========================================================================

// N requests hit the middleware while the same token is being revoked. Whatever
// the interleaving, every request that is told the token is blocklisted must be
// refused, and no request may be refused while the cache still reports a miss.
// Run with -race.
func TestAuth_ConcurrentRequestsDuringLogout(t *testing.T) {
	h := newMwHarness(t)
	token, _ := h.jwt.Create("John Doe", "user-1", "john@example.com", "user")

	const n = 32
	var revoked atomic.Bool
	var blockedAnswers, missAnswers int64

	h.cache.EXPECT().
		Get(gomock.Any(), gomock.Eq("token_blocklist:"+token)).
		DoAndReturn(func(_ context.Context, _ string) (string, error) {
			if revoked.Load() {
				atomic.AddInt64(&blockedAnswers, 1)
				return "1", nil
			}
			atomic.AddInt64(&missAnswers, 1)
			return "", portcache.ErrCacheMiss
		}).
		Times(n)

	var reached int32
	handler := h.protected(&reached)

	var wg sync.WaitGroup
	codes := make([]int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i == n/2 {
				revoked.Store(true) // the logout lands mid-flight
			}
			codes[i] = doAuth(handler, token).Code
		}(i)
	}
	wg.Wait()

	var ok, unauthorized int
	for _, c := range codes {
		switch c {
		case http.StatusOK:
			ok++
		case http.StatusUnauthorized:
			unauthorized++
		default:
			t.Errorf("unexpected status %d", c)
		}
	}
	if int64(ok) != atomic.LoadInt64(&missAnswers) {
		t.Errorf("%d requests were allowed but the blocklist reported a miss %d times",
			ok, atomic.LoadInt64(&missAnswers))
	}
	if int64(unauthorized) != atomic.LoadInt64(&blockedAnswers) {
		t.Errorf("%d requests were refused but the blocklist reported blocked %d times",
			unauthorized, atomic.LoadInt64(&blockedAnswers))
	}
	if int32(ok) != atomic.LoadInt32(&reached) {
		t.Errorf("%d requests got 200 but %d reached the handler", ok, atomic.LoadInt32(&reached))
	}
}

// mutate flips one character of a base64url segment so the segment stays
// well-formed but no longer matches what was signed.
func mutate(seg string) string {
	if seg == "" {
		return "A"
	}
	b := []byte(seg)
	if b[0] == 'A' {
		b[0] = 'B'
	} else {
		b[0] = 'A'
	}
	return string(b)
}
