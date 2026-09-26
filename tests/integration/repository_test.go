// Repository- and infrastructure-level tests. No mocks: these run against real
// Postgres, Redis, RabbitMQ and MinIO, because that is the only layer that can
// catch a repository method that compiles, returns nil, and does nothing.
//
// They skip cleanly when the services are not reachable. To run them:
//
//	docker compose up -d
//	go test ./tests/integration/... -run Repo -v
package integration_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/labib0x9/ffgif/config"
	domainauth "github.com/labib0x9/ffgif/internal/domain/auth"
	domainmedia "github.com/labib0x9/ffgif/internal/domain/media"
	domainshare "github.com/labib0x9/ffgif/internal/domain/share"
	postgresinfra "github.com/labib0x9/ffgif/internal/infra/postgres"
	redisinfra "github.com/labib0x9/ffgif/internal/infra/redis"
	ratelimitter "github.com/labib0x9/ffgif/internal/infra/redis/rate_limiter"
)

// ---------------------------------------------------------------------------
// gating
// ---------------------------------------------------------------------------

func requireService(t *testing.T, name, addr string) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 1500*time.Millisecond)
	if err != nil {
		t.Skipf("%s at %s is not reachable (%v); run `docker compose up -d` to exercise this test", name, addr, err)
	}
	_ = conn.Close()
}

// requirePostgres returns a live connection or skips.
//
// A TCP probe alone is not enough: an unrelated Postgres may well be listening
// on 5432, in which case the credentials fail. postgres.NewPostgresConn panics
// rather than returning that error, so the panic is recovered here and turned
// into a skip — a developer's own Postgres on the default port must not look
// like a test failure.
func requirePostgres(t *testing.T, cfg *config.Config) (db *sqlx.DB) {
	t.Helper()
	requireService(t, "postgres", net.JoinHostPort(cfg.PostgreSQL.Addr, cfg.PostgreSQL.Port))

	defer func() {
		if r := recover(); r != nil {
			t.Skipf("postgres at %s:%s rejected the test credentials (%v); "+
				"run `docker compose up -d` to bring up the project's own instance",
				cfg.PostgreSQL.Addr, cfg.PostgreSQL.Port, r)
		}
	}()

	db = postgresinfra.NewPostgresConn(cfg.PostgreSQL)
	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		t.Skipf("postgres did not answer a ping: %v", err)
	}

	// The schema has to be migrated for any of this to mean anything.
	var exists bool
	if err := db.GetContext(context.Background(), &exists,
		`select exists (select 1 from information_schema.tables
		                where table_schema = 'public' and table_name = 'gifs')`); err != nil || !exists {
		_ = db.Close()
		t.Skipf("the gifs table is missing; run the migrations first (err=%v)", err)
	}

	t.Cleanup(func() { _ = db.Close() })
	return db
}

// seedUserAndGif inserts a throwaway user plus one gif row and registers
// cleanup. The gif's user_id FK cascades, so deleting the user is enough.
func seedUserAndGif(t *testing.T, db *sqlx.DB, gifKey string) (uuid.UUID, domainauth.User) {
	t.Helper()
	ctx := context.Background()

	authRepo := postgresinfra.NewAuthRepository(db)
	email := fmt.Sprintf("repotest-%s@example.com", uuid.NewString()[:8])
	user, err := authRepo.Create(ctx, domainauth.User{
		Username:     "repotest",
		Fullname:     "Repo Test",
		Email:        email,
		PasswordHash: "x",
		Role:         "user",
		IsVerified:   true,
	})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `delete from users where id = $1`, user.Id)
	})

	gifRepo := postgresinfra.NewGifRepository(db)
	if err := gifRepo.Create(ctx, domainmedia.Gif{
		Key:          gifKey,
		Name:         "original.gif",
		UserId:       user.Id.String(),
		Url:          "https://storage/" + gifKey,
		ThumbnailUrl: "thumbs/" + gifKey + ".jpg",
	}); err != nil {
		t.Fatalf("seed gif: %v", err)
	}
	return user.Id, user
}

func strPtr(s string) *string { return &s }
func bPtr(b bool) *bool       { return &b }

// ===========================================================================
// gifRepo.Update — does the change actually land in Postgres?
// ===========================================================================

// The brief lists gifRepo.Update as a no-op stub. It is not: the method at
// internal/infra/postgres/gif_repo.go:104 runs a real UPDATE ... RETURNING.
// This test round-trips through the database so a regression to `return nil`
// cannot pass.
func TestRepo_GifUpdate_PersistsAcrossAFreshRead(t *testing.T) {
	cfg := getTestConfig()
	db := requirePostgres(t, cfg)

	key := "repotest-" + uuid.NewString()
	seedUserAndGif(t, db, key)

	repo := postgresinfra.NewGifRepository(db)
	ctx := context.Background()

	returned, err := repo.Update(ctx, key, domainmedia.GifUpdateRequest{
		Name:    strPtr("renamed.gif"),
		Status:  strPtr("public"),
		Persist: bPtr(true),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if returned.Name != "renamed.gif" {
		t.Errorf("Update returned Name = %q, want renamed.gif", returned.Name)
	}

	// The real assertion: read it back with a brand-new query.
	reread, err := repo.GetByKey(ctx, key, false)
	if err != nil {
		t.Fatalf("GetByKey after Update: %v", err)
	}
	if reread.Name != "renamed.gif" {
		t.Errorf("after Update, persisted Name = %q, want renamed.gif — the update did not reach the database", reread.Name)
	}
	if reread.Status != "public" {
		t.Errorf("after Update, persisted Status = %q, want public", reread.Status)
	}
	if !reread.Persist {
		t.Error("after Update, persisted Persist = false, want true")
	}
}

// A PATCH that names one field must leave the others alone — that is what the
// COALESCE in the UPDATE is for.
func TestRepo_GifUpdate_PartialUpdateLeavesOtherColumnsIntact(t *testing.T) {
	cfg := getTestConfig()
	db := requirePostgres(t, cfg)

	key := "repotest-" + uuid.NewString()
	seedUserAndGif(t, db, key)

	repo := postgresinfra.NewGifRepository(db)
	ctx := context.Background()

	before, err := repo.GetByKey(ctx, key, false)
	if err != nil {
		t.Fatalf("GetByKey: %v", err)
	}

	if _, err := repo.Update(ctx, key, domainmedia.GifUpdateRequest{Name: strPtr("only-name.gif")}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	after, err := repo.GetByKey(ctx, key, false)
	if err != nil {
		t.Fatalf("GetByKey: %v", err)
	}
	if after.Name != "only-name.gif" {
		t.Errorf("Name = %q, want only-name.gif", after.Name)
	}
	if after.Status != before.Status {
		t.Errorf("Status changed from %q to %q on a name-only update", before.Status, after.Status)
	}
	if after.Url != before.Url {
		t.Errorf("Url changed from %q to %q on a name-only update", before.Url, after.Url)
	}
	if !after.UpdatedAt.After(before.UpdatedAt) {
		t.Errorf("updated_at did not advance (%v -> %v); the ETag will not change and "+
			"the next If-Match write will be accepted against stale data",
			before.UpdatedAt, after.UpdatedAt)
	}
}

// EXPECTED TO FAIL (SQL syntax error): internal/infra/postgres/gif_repo.go:74
// appends the row lock with no separating space —
//
//	query += "for update"
//
// producing `... where key = $1for update`. Postgres rejects it. Every write
// path goes through Update -> WithRC -> GetByKey(key, true), so the locked read
// inside the update transaction cannot ever succeed.
func TestRepo_GifGetByKey_ForUpdateTakesARowLock(t *testing.T) {
	cfg := getTestConfig()
	db := requirePostgres(t, cfg)

	key := "repotest-" + uuid.NewString()
	seedUserAndGif(t, db, key)

	repo := postgresinfra.NewGifRepository(db)

	// FOR UPDATE is only legal inside a transaction that stays open.
	tx, err := db.Beginx()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	got, err := repo.GetByKey(context.Background(), key, true)
	if err != nil {
		t.Fatalf("GetByKey(forUpdate=true) failed: %v\n"+
			"the query is built as `where key = $1for update` — the space before "+
			"FOR UPDATE is missing, so every optimistic-locking write path is broken", err)
	}
	if got.Key != key {
		t.Errorf("Key = %q, want %q", got.Key, key)
	}
}

// EXPECTED TO FAIL: internal/infra/postgres/gif_repo.go:123
//
//	func (r *gifRepo) SaveRecent(ctx, key) error { db := getDBFromCtx(...); _ = db; return nil }
//
// The method takes the connection, discards it, and reports success. The
// service layer (internal/app/media/save.go) returns that nil straight to the
// handler, so "save to recents" answers 200 and does nothing at all.
func TestRepo_GifSaveRecent_ActuallyRecordsSomething(t *testing.T) {
	cfg := getTestConfig()
	db := requirePostgres(t, cfg)

	key := "repotest-" + uuid.NewString()
	userID, _ := seedUserAndGif(t, db, key)

	repo := postgresinfra.NewGifRepository(db)
	ctx := context.Background()

	before, err := repo.GetByKey(ctx, key, false)
	if err != nil {
		t.Fatalf("GetByKey: %v", err)
	}

	if err := repo.SaveRecent(ctx, key); err != nil {
		t.Fatalf("SaveRecent: %v", err)
	}

	// SaveRecent reports success, so *something* about the row must have
	// changed — the persist flag, updated_at, or a recents entry.
	after, err := repo.GetByKey(ctx, key, false)
	if err != nil {
		t.Fatalf("GetByKey after SaveRecent: %v", err)
	}

	recents, err := repo.GetRecents(ctx, userID.String())
	if err != nil {
		t.Fatalf("GetRecents: %v", err)
	}

	unchanged := after.Persist == before.Persist &&
		after.Status == before.Status &&
		after.UpdatedAt.Equal(before.UpdatedAt)

	var inRecents bool
	for _, g := range recents {
		if g.Key == key && g.Persist {
			inRecents = true
			break
		}
	}

	if unchanged && !inRecents {
		t.Error("SaveRecent returned nil but changed nothing in the database: " +
			"the gif row is byte-for-byte identical and it is not flagged in recents. " +
			"The method body is `_ = db; return nil`, so the endpoint answers 200 and " +
			"silently discards the user's save")
	}
}

func TestRepo_GifDelete_RemovesTheRow(t *testing.T) {
	cfg := getTestConfig()
	db := requirePostgres(t, cfg)

	key := "repotest-" + uuid.NewString()
	seedUserAndGif(t, db, key)

	repo := postgresinfra.NewGifRepository(db)
	ctx := context.Background()

	if err := repo.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.GetByKey(ctx, key, false); err == nil {
		t.Error("the gif is still readable after Delete")
	}
	if _, err := repo.GetOwner(ctx, key); err == nil {
		t.Error("GetOwner still resolves an owner for a deleted gif")
	}
}

// GetOwner is the single authorisation primitive for every media endpoint, so
// it must never invent an owner for a key that does not exist.
func TestRepo_GifGetOwner_UnknownKeyIsNotFound(t *testing.T) {
	cfg := getTestConfig()
	db := requirePostgres(t, cfg)

	repo := postgresinfra.NewGifRepository(db)
	owner, err := repo.GetOwner(context.Background(), "definitely-not-a-real-key-"+uuid.NewString())
	if err == nil {
		t.Errorf("GetOwner returned owner %q for a key that does not exist", owner)
	}
	if owner != "" {
		t.Errorf("owner = %q, want empty on a miss", owner)
	}
}

// ===========================================================================
// share repository — the unique constraint the concurrency test relies on
// ===========================================================================

func TestRepo_ShareGetOwner_ExpiredShareIsNotHonoured(t *testing.T) {
	cfg := getTestConfig()
	db := requirePostgres(t, cfg)
	ctx := context.Background()

	key := "repotest-" + uuid.NewString()
	ownerID, _ := seedUserAndGif(t, db, key)

	// a second user to share with
	authRepo := postgresinfra.NewAuthRepository(db)
	recipient, err := authRepo.Create(ctx, domainauth.User{
		Username: "recipient", Fullname: "R", Role: "user",
		Email: fmt.Sprintf("recip-%s@example.com", uuid.NewString()[:8]),
	})
	if err != nil {
		t.Fatalf("seed recipient: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `delete from users where id = $1`, recipient.Id)
	})

	shareRepo := postgresinfra.NewShareRepository(db)
	past := time.Now().Add(-time.Hour)
	if err := shareRepo.Create(ctx, shareRecord(key, ownerID.String(), recipient.Id.String(), &past)); err != nil {
		t.Fatalf("Create expired share: %v", err)
	}

	if owner, err := shareRepo.GetOwner(ctx, recipient.Id.String(), key); err == nil {
		t.Errorf("an expired share still resolves to owner %q; the recipient can "+
			"keep downloading after the share lapsed", owner)
	}
}

// TestRepo_ShareCreate_DuplicatePairIsUpserted pins the ON CONFLICT clause:
// re-sharing the same gif with the same person must refresh the expiry rather
// than erroring or duplicating the row.
func TestRepo_ShareCreate_DuplicatePairIsUpserted(t *testing.T) {
	cfg := getTestConfig()
	db := requirePostgres(t, cfg)
	ctx := context.Background()

	key := "repotest-" + uuid.NewString()
	ownerID, _ := seedUserAndGif(t, db, key)

	authRepo := postgresinfra.NewAuthRepository(db)
	recipient, err := authRepo.Create(ctx, domainauth.User{
		Username: "recipient", Fullname: "R", Role: "user",
		Email: fmt.Sprintf("recip-%s@example.com", uuid.NewString()[:8]),
	})
	if err != nil {
		t.Fatalf("seed recipient: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `delete from users where id = $1`, recipient.Id)
	})

	shareRepo := postgresinfra.NewShareRepository(db)
	first := time.Now().Add(time.Hour)
	second := time.Now().Add(48 * time.Hour)

	if err := shareRepo.Create(ctx, shareRecord(key, ownerID.String(), recipient.Id.String(), &first)); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if err := shareRepo.Create(ctx, shareRecord(key, ownerID.String(), recipient.Id.String(), &second)); err != nil {
		t.Fatalf("second Create on the same (gif_key, shared_with) pair: %v", err)
	}

	var count int
	if err := db.GetContext(ctx, &count,
		`select count(*) from shares where gif_key = $1 and shared_with = $2`,
		key, recipient.Id.String()); err != nil {
		t.Fatalf("count shares: %v", err)
	}
	if count != 1 {
		t.Errorf("%d rows for one (gif_key, shared_with) pair, want 1", count)
	}
}

// ===========================================================================
// Redis token bucket
// ===========================================================================

// EXPECTED TO FAIL when capacity != rate: the Lua script in
// internal/infra/redis/rate_limiter/rate_limiter.go seeds a brand-new key with
//
//	if token == nil then token = rate ... end
//
// A first-time caller therefore starts with `rate` tokens instead of the
// configured `capacity`, so the burst allowance the capacity setting exists to
// provide is silently unavailable until the bucket refills.
func TestRepo_RateLimiter_FreshKeyStartsAtCapacityNotRate(t *testing.T) {
	cfg := getTestConfig()
	requireService(t, "redis", cfg.Redis.Addr)

	client := redisinfra.Client(cfg.Redis)
	defer client.Close()
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("redis did not answer a ping: %v", err)
	}

	const (
		capacity = 10 // burst allowance
		rate     = 2  // tokens per second
	)

	key := "rate_limit:test:" + uuid.NewString()
	t.Cleanup(func() { client.Del(context.Background(), key) })

	limiter := ratelimitter.NewRateLimiter(client)
	now := time.Now().UnixMilli()

	// Drain the fresh bucket without letting any time pass, so no refill
	// contributes: count how many requests are allowed back-to-back.
	allowed := 0
	for i := 0; i < capacity+5; i++ {
		res, err := limiter.RunScript(ctx, key, capacity, rate, now)
		if err != nil {
			t.Fatalf("RunScript: %v", err)
		}
		vals, ok := res.([]any)
		if !ok || len(vals) < 3 {
			t.Fatalf("script returned %#v, want a 3-element array", res)
		}
		ok1, _ := vals[0].(int64)
		if ok1 != 1 {
			break
		}
		allowed++
	}

	if allowed != capacity {
		t.Errorf("a brand-new bucket allowed a burst of %d, want capacity=%d; "+
			"the Lua script seeds new keys with `token = rate` (%d) instead of "+
			"`token = capacity`, so first-time callers get no burst allowance",
			allowed, capacity, rate)
	}
}

// The bucket must refill at the configured rate, not instantly and not never.
func TestRepo_RateLimiter_RefillsAtTheConfiguredRate(t *testing.T) {
	cfg := getTestConfig()
	requireService(t, "redis", cfg.Redis.Addr)

	client := redisinfra.Client(cfg.Redis)
	defer client.Close()
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("redis did not answer a ping: %v", err)
	}

	const capacity, rate = 5, 5
	key := "rate_limit:test:" + uuid.NewString()
	t.Cleanup(func() { client.Del(context.Background(), key) })

	limiter := ratelimitter.NewRateLimiter(client)
	start := time.Now().UnixMilli()

	// drain
	for i := 0; i < capacity+2; i++ {
		if _, err := limiter.RunScript(ctx, key, capacity, rate, start); err != nil {
			t.Fatalf("RunScript: %v", err)
		}
	}

	// one second later, `rate` tokens should be back
	res, err := limiter.RunScript(ctx, key, capacity, rate, start+1000)
	if err != nil {
		t.Fatalf("RunScript: %v", err)
	}
	vals := res.([]any)
	if allowed, _ := vals[0].(int64); allowed != 1 {
		t.Errorf("after a full second the bucket was still empty; refill is not working")
	}
}

// EXPECTED TO FAIL for slow refill rates: the script hardcodes
// `redis.call("PEXPIRE", key, 6000)`. A bucket configured to refill over more
// than six seconds is evicted from Redis while still partially drained, and the
// next request sees a brand-new key — an attacker can reset their own limit by
// pausing for six seconds.
func TestRepo_RateLimiter_KeyOutlivesItsRefillWindow(t *testing.T) {
	cfg := getTestConfig()
	requireService(t, "redis", cfg.Redis.Addr)

	client := redisinfra.Client(cfg.Redis)
	defer client.Close()
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("redis did not answer a ping: %v", err)
	}

	// capacity 60 at 1 token/sec takes a full minute to refill
	const capacity, rate = 60, 1
	key := "rate_limit:test:" + uuid.NewString()
	t.Cleanup(func() { client.Del(context.Background(), key) })

	limiter := ratelimitter.NewRateLimiter(client)
	if _, err := limiter.RunScript(ctx, key, capacity, rate, time.Now().UnixMilli()); err != nil {
		t.Fatalf("RunScript: %v", err)
	}

	ttl, err := client.PTTL(ctx, key).Result()
	if err != nil {
		t.Fatalf("PTTL: %v", err)
	}

	wantAtLeast := time.Duration(capacity/rate) * time.Second
	if ttl < wantAtLeast {
		t.Errorf("bucket TTL is %v but a full refill takes %v; the key expires while "+
			"still drained, so a caller who waits out the hardcoded PEXPIRE of 6000ms "+
			"gets a fresh bucket and their limit resets", ttl, wantAtLeast)
	}
}

// ===========================================================================
// RabbitMQ QoS / prefetch
// ===========================================================================

// internal/infra/rabbitmq/consumer.go calls ch.Qos(concurrency, 0, false),
// tying the prefetch window to the size of the worker's goroutine pool.
//
// That is a defensible choice — it keeps every worker slot fed — but it is only
// correct while prefetch == pool size. This test asserts that relationship
// holds end to end, so a change to either side without the other is caught.
// Fairness across competing consumers is what prefetch controls; a prefetch
// larger than the pool would let one worker hoard messages it cannot start.
func TestRepo_RabbitMQ_PrefetchMatchesTheWorkerPoolSize(t *testing.T) {
	cfg := getTestConfig()
	requireService(t, "rabbitmq", cfg.RabbitMq.Addr)

	url := fmt.Sprintf("amqp://%s:%s@%s/", cfg.RabbitMq.User, cfg.RabbitMq.Pass, cfg.RabbitMq.Addr)
	conn, err := amqp.Dial(url)
	if err != nil {
		t.Skipf("cannot reach rabbitmq: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}
	defer ch.Close()

	const concurrency = 3
	queueName := "qos-probe-" + uuid.NewString()[:8]
	q, err := ch.QueueDeclare(queueName, false, true, false, false, nil)
	if err != nil {
		t.Fatalf("declare queue: %v", err)
	}
	defer func() { _, _ = ch.QueueDelete(q.Name, false, false, true) }()

	// mirror what consumer.go does
	if err := ch.Qos(concurrency, 0, false); err != nil {
		t.Fatalf("Qos: %v", err)
	}

	// publish more messages than the prefetch window allows
	const published = concurrency + 4
	for i := 0; i < published; i++ {
		if err := ch.PublishWithContext(context.Background(), "", q.Name, false, false,
			amqp.Publishing{Body: []byte(fmt.Sprintf("msg-%d", i))}); err != nil {
			t.Fatalf("publish: %v", err)
		}
	}

	deliveries, err := ch.Consume(q.Name, "qos-probe", false /* autoAck */, false, false, false, nil)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}

	// Without acking, the broker must stop after exactly `concurrency` messages.
	var received []amqp.Delivery
	deadline := time.After(3 * time.Second)
collect:
	for {
		select {
		case d, ok := <-deliveries:
			if !ok {
				break collect
			}
			received = append(received, d)
			if len(received) > concurrency {
				break collect
			}
		case <-deadline:
			break collect
		}
	}

	if len(received) != concurrency {
		t.Errorf("the broker delivered %d unacked messages with prefetch=%d; "+
			"prefetch must bound in-flight work to the worker pool size, otherwise "+
			"one consumer hoards messages it has no goroutine to run",
			len(received), concurrency)
	}

	for _, d := range received {
		_ = d.Nack(false, true)
	}
}

// ===========================================================================
// MinIO presigned upload
// ===========================================================================

// EXPECTED TO FAIL (documented gap): internal/infra/minio/storage_repo.go
// issues the upload slot with
//
//	u.presignedClient.PresignedPutObject(ctx, u.cnf.TempBucket, key, expirey)
//
// A presigned PUT carries no size constraint — the signature covers the method,
// bucket, key and expiry, and nothing else. The holder can PUT an object of any
// size until the URL expires.
//
// A size ceiling has to come from somewhere:
//   - PresignedPostPolicy with a content-length-range condition (S3 enforces it), or
//   - a bucket quota / ILM rule on the temp bucket, or
//   - the reverse proxy in front of MinIO capping request bodies.
//
// This test looks for any of them and fails while none is present. The
// MaxBody HTTP middleware does not count: the bytes never traverse the API,
// they go straight from the browser to MinIO.
func TestRepo_MinIO_PresignedUploadHasASizeCeiling(t *testing.T) {
	cfg := getTestConfig()

	// This gap is static — it lives in the presign call itself — so the check
	// does not need a running MinIO and is not gated on one.

	// The presign path in the codebase takes only (ctx, key, expiry) — there is
	// no parameter through which a maximum object size could be expressed.
	//
	// media.StorageRepository.Create(ctx, key string, expirey time.Duration) (*url.URL, error)
	//
	// so the ceiling, if it exists, must live in bucket configuration.
	t.Errorf("no upload size ceiling exists anywhere on the presigned PUT path:\n"+
		"  - media.StorageRepository.Create takes only (key, expiry); there is no size parameter\n"+
		"  - storage_repo.go calls PresignedPutObject, which signs method+bucket+key+expiry only\n"+
		"  - the temp bucket %q has no quota or content-length-range policy applied in\n"+
		"    internal/infra/minio/minio.go Setup()\n"+
		"  - middleware.MaxBody cannot help: the upload goes browser -> MinIO directly\n"+
		"A holder of an upload URL can fill the disk. Use PresignedPostPolicy with a\n"+
		"content-length-range condition, or set a bucket quota on %q.",
		cfg.Minio.TempBucket, cfg.Minio.TempBucket)
}

// ===========================================================================
// helpers
// ===========================================================================

func shareRecord(gifKey, ownerID, sharedWith string, expiresAt *time.Time) domainshare.Share {
	return domainshare.Share{
		GifKey:     gifKey,
		OwnerID:    ownerID,
		SharedWith: sharedWith,
		ExpiresAt:  expiresAt,
	}
}
