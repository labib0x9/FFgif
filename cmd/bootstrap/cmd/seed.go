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

// seededGif mirrors the actual `gifs` table columns
// (key, user_id, name, status, persist, download, file_size_bytes, width, height, duration_seconds, is_public, url, thumbnail_key, created_at).
type seededGif struct {
	key             string
	userID          uuid.UUID
	name            string
	status          string
	persist         bool
	download        int
	fileSizeBytes   int64
	width           int
	height          int
	durationSeconds float64
	isPublic        bool
	url             string
	thumbnailKey    string
	createdAt       time.Time
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

	// 1. MinIO: upload seed.gif once — the only real object created.
	seedMinioStorage(ctx, cnf)

	const totalUsers = 1000
	const defaultPassword = "Password123!"

	slog.Info("Starting seed...", "users", totalUsers)

	hasher := password.NewHasher(cnf.HashPepper, cnf.BcryptCost)
	passHash, err := hasher.GenerateHash(defaultPassword)
	if err != nil {
		return fmt.Errorf("failed to generate password hash: %w", err)
	}

	// 2. Users & profiles
	users, err := seedUsersAndProfiles(ctx, db, totalUsers, passHash)
	if err != nil {
		return fmt.Errorf("failed to seed users: %w", err)
	}

	// 3. Gifs — 40 to 60 per user
	gifs, userGifStats, err := seedGifs(ctx, db, users)
	if err != nil {
		return fmt.Errorf("failed to seed gifs: %w", err)
	}

	// 4. Quota reflecting actual gif usage
	if err := seedQuotas(ctx, db, users, userGifStats); err != nil {
		return fmt.Errorf("failed to seed quotas: %w", err)
	}

	// 5. Friendships (social graph)
	acceptedFriendPairs, err := seedFriendships(ctx, db, users)
	if err != nil {
		return fmt.Errorf("failed to seed friendships: %w", err)
	}

	// 6. Shares between friends
	if err := seedShares(ctx, db, acceptedFriendPairs, gifs); err != nil {
		return fmt.Errorf("failed to seed shares: %w", err)
	}

	// 7. Public share tokens
	if err := seedShareTokens(ctx, db, gifs); err != nil {
		return fmt.Errorf("failed to seed share tokens: %w", err)
	}

	slog.Info("Seed completed successfully!",
		"users", len(users),
		"gifs", len(gifs),
	)
	return nil
}

func seedMinioStorage(ctx context.Context, cnf *config.Config) {
	gifPath := "seed.gif"
	if _, err := os.Stat(gifPath); os.IsNotExist(err) {
		candidates := []string{"/app/seed.gif", "../seed.gif", "../../seed.gif"}
		found := false
		for _, cand := range candidates {
			if _, err := os.Stat(cand); err == nil {
				gifPath = cand
				found = true
				break
			}
		}
		if !found {
			slog.Warn("seed.gif file not found on disk, skipping storage upload", "path", gifPath)
			return
		}
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

	_, err = client.FPutObject(ctx, cnf.Minio.StorageBucket, sharedGifObjectKey, gifPath, minio_go.PutObjectOptions{
		ContentType: "image/gif",
	})
	if err != nil {
		slog.Warn("Failed to upload seed.gif to MinIO storage bucket", "err", err)
	} else {
		slog.Info("Successfully uploaded seed.gif to MinIO storage bucket", "bucket", cnf.Minio.StorageBucket, "key", sharedGifObjectKey)
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

// sharedGifObjectKey is the single real object seedMinioStorage uploads to the
// storage bucket. Every seeded gif's thumbnail_url points at it (so
// GetGifThumbnail resolves for any seeded row); each gif's own `key` stays a
// unique dummy value since it's the primary key and the real download object
// key — we are not uploading 40-60k copies of the same file.
const sharedGifObjectKey = "seed.gif"

func seedGifs(ctx context.Context, db *sql.DB, users []seededUser) ([]seededGif, map[uuid.UUID]*userStats, error) {
	const seedGifSizeBytes = int64(3120532) // exact size of seed.gif, used for quota accounting and metadata

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

			statusRoll := mrand.Float64()
			var status string
			var persist bool
			var isPublic bool

			switch {
			case statusRoll < 0.70:
				status = "private"
				persist = true
				isPublic = false
			case statusRoll < 0.90:
				status = "public"
				persist = true
				isPublic = true
			default:
				status = "recent"
				persist = false
				isPublic = false
			}

			download := mrand.IntN(85)
			createdAfterUser := time.Duration(mrand.IntN(int(time.Since(u.createdAt).Seconds()))) * time.Second
			createdAt := u.createdAt.Add(createdAfterUser)

			allGifs = append(allGifs, seededGif{
				key:             key,
				userID:          u.id,
				name:            name,
				status:          status,
				persist:         persist,
				download:        download,
				fileSizeBytes:   seedGifSizeBytes,
				width:           480,
				height:          270,
				durationSeconds: 3.20,
				isPublic:        isPublic,
				url:             key,
				thumbnailKey:    sharedGifObjectKey,
				createdAt:       createdAt,
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
		gifArgs := make([]interface{}, 0, len(batch)*14)

		for _, g := range batch {
			gOffset := len(gifArgs)
			gifValues = append(gifValues, fmt.Sprintf(
				"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				gOffset+1, gOffset+2, gOffset+3, gOffset+4, gOffset+5,
				gOffset+6, gOffset+7, gOffset+8, gOffset+9, gOffset+10,
				gOffset+11, gOffset+12, gOffset+13, gOffset+14,
			))
			gifArgs = append(gifArgs,
				g.key, g.userID, g.name, g.status, g.persist,
				g.download, g.fileSizeBytes, g.width, g.height, g.durationSeconds,
				g.isPublic, g.url, g.thumbnailKey, g.createdAt,
			)
		}

		query := fmt.Sprintf(`
			INSERT INTO gifs (
				key, user_id, name, status, persist,
				download, file_size_bytes, width, height, duration_seconds,
				is_public, url, thumbnail_key, created_at
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
		if g.status != "public" || len(tokens) >= 500 {
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

func init() {
	rootCmd.AddCommand(seedCmd)
}
