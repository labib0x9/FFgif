package cmd

import (
	"context"
	crand "crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	mrand "math/rand/v2"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labib0x9/ffgif/config"
	"github.com/labib0x9/ffgif/internal/infra/minio"
	"github.com/labib0x9/ffgif/pkg/password"
	_ "github.com/lib/pq"
	minio_go "github.com/minio/minio-go/v7"
	"github.com/spf13/cobra"
)

var seed bool

var seedCmd = &cobra.Command{
	Use:     "seed",
	Aliases: []string{"seed"},
	Short:   "seeds demo data",
	RunE:    seedSetupfunc,
}

func seedSetupfunc(cmd *cobra.Command, args []string) error {
	return setupSeed()
}

type seededUser struct {
	id        uuid.UUID
	username  string
	fullname  string
	email     string
	createdAt time.Time
}

type seededGif struct {
	key       string
	userID    uuid.UUID
	name      string
	status    string
	persist   bool
	download  int
	sizeBytes int64
	width     int
	height    int
	duration  float64
	isPublic  bool
	thumbKey  string
	createdAt time.Time
}

var firstNames = []string{
	"James", "Mary", "John", "Patricia", "Robert", "Jennifer", "Michael", "Linda",
	"William", "Elizabeth", "David", "Barbara", "Richard", "Susan", "Joseph", "Jessica",
	"Thomas", "Sarah", "Charles", "Karen", "Christopher", "Nancy", "Daniel", "Lisa",
	"Matthew", "Betty", "Anthony", "Margaret", "Mark", "Sandra", "Donald", "Ashley",
	"Steven", "Kimberly", "Paul", "Emily", "Andrew", "Donna", "Joshua", "Michelle",
	"Kenneth", "Dorothy", "Kevin", "Carol", "Brian", "Amanda", "George", "Melissa",
	"Edward", "Deborah", "Alex", "Sam", "Jordan", "Taylor", "Morgan", "Casey",
}

var lastNames = []string{
	"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis",
	"Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson",
	"Thomas", "Taylor", "Moore", "Jackson", "Martin", "Lee", "Perez", "Thompson",
	"White", "Harris", "Sanchez", "Clark", "Ramirez", "Lewis", "Robinson", "Walker",
	"Young", "Allen", "King", "Wright", "Scott", "Torres", "Nguyen", "Hill", "Flores",
}

var gifAdjectives = []string{
	"Funny", "Epic", "Cute", "Crazy", "Cool", "Awesome", "Super", "Happy",
	"Dancing", "Gaming", "Retro", "Neon", "Cyber", "Speed", "Pixel", "Vintage",
	"Space", "Ocean", "Fire", "Spark", "Magic", "Hyper", "Vibrant", "Smooth",
}

var gifNouns = []string{
	"Moment", "Clip", "Dance", "Jump", "Vibe", "Roll", "Spin", "Flip",
	"Scene", "Explosion", "Victory", "Fail", "Celebration", "Wave", "Glow",
	"Drift", "Sunset", "Flight", "Combo", "Dodge", "Party", "Chill", "Burst",
}

var videoNames = []string{
	"screen_recording.mp4", "gameplay_highlight.mp4", "vacation_clip.mp4",
	"meme_compilation.mp4", "tutorial_capture.mp4", "reaction_short.mp4",
	"sports_highlight.mp4", "drone_footage.mp4", "music_video_clip.mp4",
}

func setupSeed() error {
	cnf := config.GetConfig()
	dbSource := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cnf.PostgreSQL.User,
		cnf.PostgreSQL.Pass,
		cnf.PostgreSQL.Addr,
		cnf.PostgreSQL.Port,
		cnf.PostgreSQL.DatabaseName,
		cnf.PostgreSQL.SslMode,
	)

	db, err := sql.Open("postgres", dbSource)
	if err != nil {
		return fmt.Errorf("failed to open db connection: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping db: %w", err)
	}

	// 1. MinIO: Upload seed.gif to storage bucket
	seedMinioStorage(ctx, cnf)

	const totalUsers = 1000
	const defaultPassword = "Password123!"

	slog.Info("Starting comprehensive simulation seed...", "users", totalUsers)

	hasher := password.NewHasher(cnf.HashPepper, cnf.BcryptCost)
	passHash, err := hasher.GenerateHash(defaultPassword)
	if err != nil {
		return fmt.Errorf("failed to generate password hash: %w", err)
	}

	// 2. Generate and seed 1,000 users & profiles
	users, err := seedUsersAndProfiles(ctx, db, totalUsers, passHash)
	if err != nil {
		return fmt.Errorf("failed to seed users: %w", err)
	}

	// 3. Generate and seed GIFs (40-60 randomly per user) using seed.gif characteristics
	gifs, userGifStats, err := seedGifs(ctx, db, users)
	if err != nil {
		return fmt.Errorf("failed to seed gifs: %w", err)
	}

	// 4. Seed user quotas reflecting their actual GIF usage
	if err := seedQuotas(ctx, db, users, userGifStats); err != nil {
		return fmt.Errorf("failed to seed quotas: %w", err)
	}

	// 5. Seed video conversion jobs (completed + in-progress/failed)
	if err := seedJobs(ctx, db, users, gifs); err != nil {
		return fmt.Errorf("failed to seed jobs: %w", err)
	}

	// 6. Seed friendships (social graph between users)
	acceptedFriendPairs, err := seedFriendships(ctx, db, users)
	if err != nil {
		return fmt.Errorf("failed to seed friendships: %w", err)
	}

	// 7. Seed shares (sharing GIFs with friends)
	if err := seedShares(ctx, db, acceptedFriendPairs, gifs); err != nil {
		return fmt.Errorf("failed to seed shares: %w", err)
	}

	// 8. Seed public share tokens
	if err := seedShareTokens(ctx, db, gifs); err != nil {
		return fmt.Errorf("failed to seed share tokens: %w", err)
	}

	// 9. Seed last_upload metadata
	if err := seedLastUploads(ctx, db, users); err != nil {
		return fmt.Errorf("failed to seed last uploads: %w", err)
	}

	// 10. Seed anonymous users, anon profiles, and anon quotas
	if err := seedAnonymousSessions(ctx, db); err != nil {
		return fmt.Errorf("failed to seed anonymous sessions: %w", err)
	}

	// 11. Seed pending verifiers & password resetters
	if err := seedAuthTokens(ctx, db, users); err != nil {
		return fmt.Errorf("failed to seed auth tokens: %w", err)
	}

	slog.Info("Full simulation seed completed successfully!",
		"users", len(users),
		"gifs", len(gifs),
	)
	return nil
}

func seedMinioStorage(ctx context.Context, cnf *config.Config) {
	gifPath := "seed.gif"
	if _, err := os.Stat(gifPath); os.IsNotExist(err) {
		slog.Warn("seed.gif file not found on disk, skipping storage upload", "path", gifPath)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			slog.Warn("MinIO storage unavailable during seeding, skipping direct storage uploads", "err", r)
		}
	}()

	client := minio.NewMinio(cnf.Minio)
	exists, err := client.BucketExists(ctx, cnf.Minio.StorageBucket)
	if err != nil || !exists {
		if err := client.MakeBucket(ctx, cnf.Minio.StorageBucket, minio_go.MakeBucketOptions{Region: "us-east-1"}); err != nil {
			slog.Warn("Could not ensure storage bucket exists in MinIO", "err", err)
			return
		}
	}

	_, err = client.FPutObject(ctx, cnf.Minio.StorageBucket, "seed.gif", gifPath, minio_go.PutObjectOptions{
		ContentType: "image/gif",
	})
	if err != nil {
		slog.Warn("Failed to upload seed.gif to MinIO storage bucket", "err", err)
	} else {
		slog.Info("Successfully uploaded seed.gif to MinIO storage bucket", "bucket", cnf.Minio.StorageBucket)
	}
}

func seedUsersAndProfiles(ctx context.Context, db *sql.DB, count int, passHash string) ([]seededUser, error) {
	users := make([]seededUser, 0, count)
	now := time.Now()

	for i := 1; i <= count; i++ {
		fn := firstNames[mrand.IntN(len(firstNames))]
		ln := lastNames[mrand.IntN(len(lastNames))]

		username := fmt.Sprintf("%s_%s%d", strings.ToLower(fn), strings.ToLower(ln), i)
		if len(username) > 20 {
			username = username[:20]
		}
		email := fmt.Sprintf("%s.%s%d@ffgif.local", strings.ToLower(fn), strings.ToLower(ln), i)
		if len(email) > 50 {
			email = fmt.Sprintf("u%d_%s@ffgif.local", i, strings.ToLower(ln))
		}
		fullname := fmt.Sprintf("%s %s", fn, ln)
		createdPastDays := mrand.IntN(90) + 1
		createdAt := now.AddDate(0, 0, -createdPastDays).Add(time.Duration(mrand.IntN(86400)) * time.Second)

		users = append(users, seededUser{
			id:        uuid.New(),
			username:  username,
			fullname:  fullname,
			email:     email,
			createdAt: createdAt,
		})
	}

	const batchSize = 250
	for i := 0; i < len(users); i += batchSize {
		end := i + batchSize
		if end > len(users) {
			end = len(users)
		}
		batch := users[i:end]

		userValues := make([]string, 0, len(batch))
		userArgs := make([]interface{}, 0, len(batch)*8)

		profileValues := make([]string, 0, len(batch))
		profileArgs := make([]interface{}, 0, len(batch)*3)

		for _, u := range batch {
			uOffset := len(userArgs)
			userValues = append(userValues, fmt.Sprintf(
				"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				uOffset+1, uOffset+2, uOffset+3, uOffset+4, uOffset+5, uOffset+6, uOffset+7, uOffset+8,
			))
			userArgs = append(userArgs, u.id, u.username, u.fullname, u.email, passHash, true, "user", u.createdAt)

			pOffset := len(profileArgs)
			pic := fmt.Sprintf("https://api.dicebear.com/7.x/bottts/svg?seed=%s", u.username)
			profileValues = append(profileValues, fmt.Sprintf(
				"($%d, $%d, $%d)",
				pOffset+1, pOffset+2, pOffset+3,
			))
			profileArgs = append(profileArgs, u.id, pic, u.createdAt)
		}

		userQuery := fmt.Sprintf(`
			INSERT INTO users (id, username, fullname, email, password_hash, is_verified, role, created_at)
			VALUES %s
			ON CONFLICT (email) DO NOTHING;
		`, strings.Join(userValues, ", "))
		if _, err := db.ExecContext(ctx, userQuery, userArgs...); err != nil {
			return nil, fmt.Errorf("insert users batch error: %w", err)
		}

		profileQuery := fmt.Sprintf(`
			INSERT INTO profiles (user_id, profile_pic, created_at)
			VALUES %s
			ON CONFLICT (user_id) DO NOTHING;
		`, strings.Join(profileValues, ", "))
		if _, err := db.ExecContext(ctx, profileQuery, profileArgs...); err != nil {
			return nil, fmt.Errorf("insert profiles batch error: %w", err)
		}
	}

	slog.Info("Users and profiles seeded", "count", len(users))
	return users, nil
}

type userStats struct {
	gifCount  int
	usedBytes int64
}

func seedGifs(ctx context.Context, db *sql.DB, users []seededUser) ([]seededGif, map[uuid.UUID]*userStats, error) {
	const seedGifSizeBytes = int64(3120532) // exact size of seed.gif
	const width = 500
	const height = 750
	const duration = 3.50

	allGifs := make([]seededGif, 0, len(users)*50)
	statsMap := make(map[uuid.UUID]*userStats, len(users))

	for _, u := range users {
		numGifs := mrand.IntN(21) + 40 // 40 to 60 GIFs randomly
		statsMap[u.id] = &userStats{
			gifCount:  numGifs,
			usedBytes: int64(numGifs) * seedGifSizeBytes,
		}

		for g := 0; g < numGifs; g++ {
			adj := gifAdjectives[mrand.IntN(len(gifAdjectives))]
			noun := gifNouns[mrand.IntN(len(gifNouns))]
			name := fmt.Sprintf("%s %s #%d", adj, noun, g+1)

			gifID := uuid.New().String()
			key := fmt.Sprintf("%s_%s_output.gif", u.id, gifID[:8])
			thumbKey := fmt.Sprintf("thumpnail_%s.jpg", gifID[:8])

			statusRoll := mrand.Float64()
			var status string
			var isPublic bool
			var persist bool

			switch {
			case statusRoll < 0.70:
				status = "private"
				isPublic = false
				persist = true
			case statusRoll < 0.90:
				status = "public"
				isPublic = true
				persist = true
			default:
				status = "recent"
				isPublic = false
				persist = false
			}

			download := mrand.IntN(85)
			createdAfterUser := time.Duration(mrand.IntN(int(time.Since(u.createdAt).Seconds()))) * time.Second
			createdAt := u.createdAt.Add(createdAfterUser)

			allGifs = append(allGifs, seededGif{
				key:       key,
				userID:    u.id,
				name:      name,
				status:    status,
				persist:   persist,
				download:  download,
				sizeBytes: seedGifSizeBytes,
				width:     width,
				height:    height,
				duration:  duration,
				isPublic:  isPublic,
				thumbKey:  thumbKey,
				createdAt: createdAt,
			})
		}
	}

	const batchSize = 500
	for i := 0; i < len(allGifs); i += batchSize {
		end := i + batchSize
		if end > len(allGifs) {
			end = len(allGifs)
		}
		batch := allGifs[i:end]

		gifValues := make([]string, 0, len(batch))
		gifArgs := make([]interface{}, 0, len(batch)*13)

		for _, g := range batch {
			gOffset := len(gifArgs)
			gifValues = append(gifValues, fmt.Sprintf(
				"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				gOffset+1, gOffset+2, gOffset+3, gOffset+4, gOffset+5, gOffset+6,
				gOffset+7, gOffset+8, gOffset+9, gOffset+10, gOffset+11, gOffset+12, gOffset+13,
			))
			gifArgs = append(gifArgs,
				g.key, g.userID, g.name, g.status, g.persist, g.download,
				g.sizeBytes, g.width, g.height, g.duration, g.isPublic,
				g.key, g.thumbKey,
			)
		}

		query := fmt.Sprintf(`
			INSERT INTO gifs (
				key, user_id, name, status, persist, download,
				file_size_bytes, width, height, duration_seconds, is_public,
				url, thumbnail_key
			)
			VALUES %s
			ON CONFLICT (key) DO NOTHING;
		`, strings.Join(gifValues, ", "))

		if _, err := db.ExecContext(ctx, query, gifArgs...); err != nil {
			return nil, nil, fmt.Errorf("insert gifs batch error: %w", err)
		}
	}

	slog.Info("Gifs seeded across all users", "totalGifs", len(allGifs))
	return allGifs, statsMap, nil
}

func seedQuotas(ctx context.Context, db *sql.DB, users []seededUser, stats map[uuid.UUID]*userStats) error {
	const batchSize = 250
	for i := 0; i < len(users); i += batchSize {
		end := i + batchSize
		if end > len(users) {
			end = len(users)
		}
		batch := users[i:end]

		quotaValues := make([]string, 0, len(batch))
		quotaArgs := make([]interface{}, 0, len(batch)*4)

		for _, u := range batch {
			st := stats[u.id]
			usedBytes := int64(0)
			gifCount := 0
			if st != nil {
				usedBytes = st.usedBytes
				gifCount = st.gifCount
			}

			qOffset := len(quotaArgs)
			quotaValues = append(quotaValues, fmt.Sprintf(
				"($%d, $%d, $%d, $%d)",
				qOffset+1, qOffset+2, qOffset+3, qOffset+4,
			))
			quotaArgs = append(quotaArgs, u.id, usedBytes, 1073741824, gifCount)
		}

		query := fmt.Sprintf(`
			INSERT INTO quota (user_id, used_bytes, total_bytes, gif_count)
			VALUES %s
			ON CONFLICT (user_id) DO UPDATE SET
				used_bytes = EXCLUDED.used_bytes,
				gif_count = EXCLUDED.gif_count;
		`, strings.Join(quotaValues, ", "))

		if _, err := db.ExecContext(ctx, query, quotaArgs...); err != nil {
			return fmt.Errorf("insert quota batch error: %w", err)
		}
	}

	slog.Info("Quotas seeded matching gif usage", "users", len(users))
	return nil
}

func seedJobs(ctx context.Context, db *sql.DB, users []seededUser, gifs []seededGif) error {
	type jobEntry struct {
		id        uuid.UUID
		userID    uuid.UUID
		jobType   string
		status    string
		progress  int
		errMsg    string
		resultKey string
		createdAt time.Time
	}

	jobs := make([]jobEntry, 0, len(gifs)+50)

	// Create a completed conversion job for each GIF
	for _, g := range gifs {
		jobs = append(jobs, jobEntry{
			id:        uuid.New(),
			userID:    g.userID,
			jobType:   "video_convert",
			status:    "done",
			progress:  100,
			errMsg:    "",
			resultKey: g.key,
			createdAt: g.createdAt.Add(-15 * time.Second),
		})
	}

	// Add realistic failed and queued jobs for a small subset of users
	for i := 0; i < 30 && i < len(users); i++ {
		jobs = append(jobs, jobEntry{
			id:        uuid.New(),
			userID:    users[i].id,
			jobType:   "video_convert",
			status:    "failed",
			progress:  45,
			errMsg:    "unsupported video codec or corrupt frame index",
			resultKey: "",
			createdAt: time.Now().Add(-2 * time.Hour),
		})
	}
	for i := 30; i < 50 && i < len(users); i++ {
		jobs = append(jobs, jobEntry{
			id:        uuid.New(),
			userID:    users[i].id,
			jobType:   "video_convert",
			status:    "queued",
			progress:  0,
			errMsg:    "",
			resultKey: "",
			createdAt: time.Now().Add(-5 * time.Minute),
		})
	}

	const batchSize = 500
	for i := 0; i < len(jobs); i += batchSize {
		end := i + batchSize
		if end > len(jobs) {
			end = len(jobs)
		}
		batch := jobs[i:end]

		jobValues := make([]string, 0, len(batch))
		jobArgs := make([]interface{}, 0, len(batch)*7)

		for _, j := range batch {
			jOffset := len(jobArgs)
			jobValues = append(jobValues, fmt.Sprintf(
				"($%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				jOffset+1, jOffset+2, jOffset+3, jOffset+4, jOffset+5, jOffset+6, jOffset+7,
			))
			jobArgs = append(jobArgs, j.id, j.userID, j.jobType, j.status, j.progress, j.errMsg, j.resultKey)
		}

		query := fmt.Sprintf(`
			INSERT INTO jobs (id, user_id, type, status, progress, error_message, result_key)
			VALUES %s
			ON CONFLICT (id) DO NOTHING;
		`, strings.Join(jobValues, ", "))

		if _, err := db.ExecContext(ctx, query, jobArgs...); err != nil {
			return fmt.Errorf("insert jobs batch error: %w", err)
		}
	}

	slog.Info("Conversion jobs seeded", "count", len(jobs))
	return nil
}

type friendPair struct {
	userA uuid.UUID
	userB uuid.UUID
}

func seedFriendships(ctx context.Context, db *sql.DB, users []seededUser) ([]friendPair, error) {
	type friendshipRecord struct {
		requesterID uuid.UUID
		addresseeID uuid.UUID
		status      string
		createdAt   time.Time
	}

	records := make([]friendshipRecord, 0, len(users)*3)
	pairSet := make(map[string]bool)
	acceptedPairs := make([]friendPair, 0, len(users)*2)

	for idx, u := range users {
		numFriends := mrand.IntN(4) + 2 // 2 to 5 friends
		for f := 0; f < numFriends; f++ {
			targetIdx := mrand.IntN(len(users))
			if targetIdx == idx {
				continue
			}
			target := users[targetIdx]

			key := fmt.Sprintf("%s:%s", u.id, target.id)
			revKey := fmt.Sprintf("%s:%s", target.id, u.id)
			if pairSet[key] || pairSet[revKey] {
				continue
			}
			pairSet[key] = true

			status := "accepted"
			if mrand.Float64() < 0.15 {
				status = "pending"
			} else {
				acceptedPairs = append(acceptedPairs, friendPair{userA: u.id, userB: target.id})
			}

			records = append(records, friendshipRecord{
				requesterID: u.id,
				addresseeID: target.id,
				status:      status,
				createdAt:   time.Now().Add(-time.Duration(mrand.IntN(60*24)) * time.Hour),
			})
		}
	}

	const batchSize = 250
	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}
		batch := records[i:end]

		fValues := make([]string, 0, len(batch))
		fArgs := make([]interface{}, 0, len(batch)*4)

		for _, r := range batch {
			offset := len(fArgs)
			fValues = append(fValues, fmt.Sprintf(
				"($%d, $%d, $%d, $%d)",
				offset+1, offset+2, offset+3, offset+4,
			))
			fArgs = append(fArgs, r.requesterID, r.addresseeID, r.status, r.createdAt)
		}

		query := fmt.Sprintf(`
			INSERT INTO friendships (requester_id, addressee_id, status, created_at)
			VALUES %s
			ON CONFLICT (requester_id, addressee_id) DO NOTHING;
		`, strings.Join(fValues, ", "))

		if _, err := db.ExecContext(ctx, query, fArgs...); err != nil {
			return nil, fmt.Errorf("insert friendships batch error: %w", err)
		}
	}

	slog.Info("Friendships seeded", "count", len(records))
	return acceptedPairs, nil
}

func seedShares(ctx context.Context, db *sql.DB, pairs []friendPair, gifs []seededGif) error {
	userGifs := make(map[uuid.UUID][]seededGif)
	for _, g := range gifs {
		userGifs[g.userID] = append(userGifs[g.userID], g)
	}

	type shareRecord struct {
		gifKey     string
		ownerID    uuid.UUID
		sharedWith uuid.UUID
		expiresAt  *time.Time
		createdAt  time.Time
	}

	shares := make([]shareRecord, 0, len(pairs)*2)
	shareSet := make(map[string]bool)

	for _, p := range pairs {
		ownerGifs := userGifs[p.userA]
		if len(ownerGifs) == 0 {
			continue
		}
		sampleGif := ownerGifs[mrand.IntN(len(ownerGifs))]
		pairKey := fmt.Sprintf("%s:%s", sampleGif.key, p.userB)
		if shareSet[pairKey] {
			continue
		}
		shareSet[pairKey] = true

		var expiresAt *time.Time
		if mrand.Float64() < 0.60 {
			exp := time.Now().Add(7 * 24 * time.Hour)
			expiresAt = &exp
		}

		shares = append(shares, shareRecord{
			gifKey:     sampleGif.key,
			ownerID:    p.userA,
			sharedWith: p.userB,
			expiresAt:  expiresAt,
			createdAt:  time.Now().Add(-time.Duration(mrand.IntN(14*24)) * time.Hour),
		})
	}

	const batchSize = 250
	for i := 0; i < len(shares); i += batchSize {
		end := i + batchSize
		if end > len(shares) {
			end = len(shares)
		}
		batch := shares[i:end]

		sValues := make([]string, 0, len(batch))
		sArgs := make([]interface{}, 0, len(batch)*5)

		for _, s := range batch {
			offset := len(sArgs)
			sValues = append(sValues, fmt.Sprintf(
				"($%d, $%d, $%d, $%d, $%d)",
				offset+1, offset+2, offset+3, offset+4, offset+5,
			))
			sArgs = append(sArgs, s.gifKey, s.ownerID, s.sharedWith, s.expiresAt, s.createdAt)
		}

		query := fmt.Sprintf(`
			INSERT INTO shares (gif_key, owner_id, shared_with, expires_at, created_at)
			VALUES %s
			ON CONFLICT (gif_key, shared_with) DO NOTHING;
		`, strings.Join(sValues, ", "))

		if _, err := db.ExecContext(ctx, query, sArgs...); err != nil {
			return fmt.Errorf("insert shares batch error: %w", err)
		}
	}

	slog.Info("Shared GIFs seeded", "count", len(shares))
	return nil
}

func seedShareTokens(ctx context.Context, db *sql.DB, gifs []seededGif) error {
	type tokenRecord struct {
		gifKey    string
		token     string
		email     string
		expiresAt time.Time
		createdAt time.Time
	}

	tokens := make([]tokenRecord, 0, 500)
	tokenSet := make(map[string]bool)

	for _, g := range gifs {
		if !g.isPublic || len(tokens) >= 500 {
			continue
		}
		if mrand.Float64() > 0.08 {
			continue
		}

		bytes := make([]byte, 16)
		_, _ = crand.Read(bytes)
		tokenStr := hex.EncodeToString(bytes)

		recipientEmail := fmt.Sprintf("recipient_%s@external.org", tokenStr[:8])
		pairKey := fmt.Sprintf("%s:%s", g.key, recipientEmail)
		if tokenSet[pairKey] {
			continue
		}
		tokenSet[pairKey] = true

		tokens = append(tokens, tokenRecord{
			gifKey:    g.key,
			token:     tokenStr,
			email:     recipientEmail,
			expiresAt: time.Now().Add(3 * 24 * time.Hour),
			createdAt: time.Now().Add(-time.Duration(mrand.IntN(48)) * time.Hour),
		})
	}

	const batchSize = 250
	for i := 0; i < len(tokens); i += batchSize {
		end := i + batchSize
		if end > len(tokens) {
			end = len(tokens)
		}
		batch := tokens[i:end]

		tValues := make([]string, 0, len(batch))
		tArgs := make([]interface{}, 0, len(batch)*5)

		for _, t := range batch {
			offset := len(tArgs)
			tValues = append(tValues, fmt.Sprintf(
				"($%d, $%d, $%d, $%d, $%d)",
				offset+1, offset+2, offset+3, offset+4, offset+5,
			))
			tArgs = append(tArgs, t.gifKey, t.token, t.email, t.expiresAt, t.createdAt)
		}

		query := fmt.Sprintf(`
			INSERT INTO share_tokens (gif_key, token, email, expires_at, created_at)
			VALUES %s
			ON CONFLICT (gif_key, email) DO NOTHING;
		`, strings.Join(tValues, ", "))

		if _, err := db.ExecContext(ctx, query, tArgs...); err != nil {
			return fmt.Errorf("insert share tokens batch error: %w", err)
		}
	}

	slog.Info("Share tokens seeded", "count", len(tokens))
	return nil
}

func seedLastUploads(ctx context.Context, db *sql.DB, users []seededUser) error {
	type uploadRecord struct {
		userID      uuid.UUID
		fileKey     string
		fileName    string
		contentType string
		sizeBytes   int64
		durationSec float64
		thumbURL    string
		uploadedAt  time.Time
	}

	records := make([]uploadRecord, 0, int(float64(len(users))*0.75))

	for _, u := range users {
		if mrand.Float64() > 0.75 {
			continue
		}

		uploadID := uuid.New().String()[:8]
		fileKey := fmt.Sprintf("raw_video_%s.mp4", uploadID)
		fileName := videoNames[mrand.IntN(len(videoNames))]
		sizeBytes := int64(mrand.IntN(25000000) + 5000000)   // 5MB to 30MB
		durationSec := float64(mrand.IntN(150)+30) / 10.0     // 3.0s to 18.0s
		thumbURL := fmt.Sprintf("thumb_%s.jpg", uploadID)
		uploadedAt := time.Now().Add(-time.Duration(mrand.IntN(72)) * time.Hour)

		records = append(records, uploadRecord{
			userID:      u.id,
			fileKey:     fileKey,
			fileName:    fileName,
			contentType: "video/mp4",
			sizeBytes:   sizeBytes,
			durationSec: durationSec,
			thumbURL:    thumbURL,
			uploadedAt:  uploadedAt,
		})
	}

	const batchSize = 250
	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}
		batch := records[i:end]

		uValues := make([]string, 0, len(batch))
		uArgs := make([]interface{}, 0, len(batch)*8)

		for _, r := range batch {
			offset := len(uArgs)
			uValues = append(uValues, fmt.Sprintf(
				"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				offset+1, offset+2, offset+3, offset+4, offset+5, offset+6, offset+7, offset+8,
			))
			uArgs = append(uArgs, r.userID, r.fileKey, r.fileName, r.contentType, r.sizeBytes, r.durationSec, r.thumbURL, r.uploadedAt)
		}

		query := fmt.Sprintf(`
			INSERT INTO last_upload (user_id, file_key, file_name, content_type, size_bytes, duration_sec, thumbnail_url, uploaded_at)
			VALUES %s
			ON CONFLICT (user_id) DO NOTHING;
		`, strings.Join(uValues, ", "))

		if _, err := db.ExecContext(ctx, query, uArgs...); err != nil {
			return fmt.Errorf("insert last_upload batch error: %w", err)
		}
	}

	slog.Info("Last uploads seeded", "count", len(records))
	return nil
}

func seedAnonymousSessions(ctx context.Context, db *sql.DB) error {
	const anonCount = 25
	anonUsers := make([]uuid.UUID, 0, anonCount)

	userValues := make([]string, 0, anonCount)
	userArgs := make([]interface{}, 0, anonCount*3)

	profileValues := make([]string, 0, anonCount)
	profileArgs := make([]interface{}, 0, anonCount)

	quotaValues := make([]string, 0, anonCount)
	quotaArgs := make([]interface{}, 0, anonCount)

	for i := 1; i <= anonCount; i++ {
		anonID := uuid.New()
		anonUsers = append(anonUsers, anonID)
		username := fmt.Sprintf("anon_%s", anonID.String()[:8])
		fullname := fmt.Sprintf("Guest User %d", i)

		uOffset := len(userArgs)
		userValues = append(userValues, fmt.Sprintf("($%d, $%d, $%d)", uOffset+1, uOffset+2, uOffset+3))
		userArgs = append(userArgs, anonID, username, fullname)

		pOffset := len(profileArgs)
		profileValues = append(profileValues, fmt.Sprintf("($%d)", pOffset+1))
		profileArgs = append(profileArgs, anonID)

		qOffset := len(quotaArgs)
		quotaValues = append(quotaValues, fmt.Sprintf("($%d)", qOffset+1))
		quotaArgs = append(quotaArgs, anonID)
	}

	userQuery := fmt.Sprintf(`
		INSERT INTO anon_users (id, username, fullname)
		VALUES %s
		ON CONFLICT (id) DO NOTHING;
	`, strings.Join(userValues, ", "))
	if _, err := db.ExecContext(ctx, userQuery, userArgs...); err != nil {
		return fmt.Errorf("insert anon_users error: %w", err)
	}

	profileQuery := fmt.Sprintf(`
		INSERT INTO anon_profiles (user_id)
		VALUES %s
		ON CONFLICT (user_id) DO NOTHING;
	`, strings.Join(profileValues, ", "))
	if _, err := db.ExecContext(ctx, profileQuery, profileArgs...); err != nil {
		return fmt.Errorf("insert anon_profiles error: %w", err)
	}

	quotaQuery := fmt.Sprintf(`
		INSERT INTO anon_quota (user_id)
		VALUES %s
		ON CONFLICT (user_id) DO NOTHING;
	`, strings.Join(quotaValues, ", "))
	if _, err := db.ExecContext(ctx, quotaQuery, quotaArgs...); err != nil {
		return fmt.Errorf("insert anon_quota error: %w", err)
	}

	slog.Info("Anonymous sessions seeded", "count", anonCount)
	return nil
}

func seedAuthTokens(ctx context.Context, db *sql.DB, users []seededUser) error {
	if len(users) < 30 {
		return nil
	}

	for i := 0; i < 15; i++ {
		bytes := make([]byte, 20)
		_, _ = crand.Read(bytes)
		tokenHash := hex.EncodeToString(bytes)

		_, _ = db.ExecContext(ctx, `
			INSERT INTO verifier (user_id, token_hash, expire_at)
			VALUES ($1, $2, NOW() + INTERVAL '30 minutes')
			ON CONFLICT (user_id) DO NOTHING;
		`, users[i].id, tokenHash)
	}

	for i := 15; i < 30; i++ {
		bytes := make([]byte, 20)
		_, _ = crand.Read(bytes)
		tokenHash := hex.EncodeToString(bytes)

		_, _ = db.ExecContext(ctx, `
			INSERT INTO reseter (user_id, token_hash, used, expire_at)
			VALUES ($1, $2, FALSE, NOW() + INTERVAL '15 minutes')
			ON CONFLICT (user_id) DO NOTHING;
		`, users[i].id, tokenHash)
	}

	slog.Info("Auth verifier and reseter tokens seeded")
	return nil
}

func init() {
	rootCmd.AddCommand(seedCmd)
}
