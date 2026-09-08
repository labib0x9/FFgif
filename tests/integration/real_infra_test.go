package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labib0x9/ffgif/config"
	authapp "github.com/labib0x9/ffgif/internal/app/auth"
	mediaapp "github.com/labib0x9/ffgif/internal/app/media"
	shareapp "github.com/labib0x9/ffgif/internal/app/share"
	userapp "github.com/labib0x9/ffgif/internal/app/user"
	domainauth "github.com/labib0x9/ffgif/internal/domain/auth"
	domainmedia "github.com/labib0x9/ffgif/internal/domain/media"
	domainuser "github.com/labib0x9/ffgif/internal/domain/user"
	"github.com/labib0x9/ffgif/internal/infra/ffmpeg"
	minioinfra "github.com/labib0x9/ffgif/internal/infra/minio"
	postgresinfra "github.com/labib0x9/ffgif/internal/infra/postgres"
	rabbitmqinfra "github.com/labib0x9/ffgif/internal/infra/rabbitmq"
	redisinfra "github.com/labib0x9/ffgif/internal/infra/redis"
	"github.com/labib0x9/ffgif/internal/infra/redis/cache"
	ratelimitter "github.com/labib0x9/ffgif/internal/infra/redis/rate_limiter"
	authhandler "github.com/labib0x9/ffgif/internal/transport/http/handlers/auth"
	mediahandler "github.com/labib0x9/ffgif/internal/transport/http/handlers/media"
	sharehandler "github.com/labib0x9/ffgif/internal/transport/http/handlers/share"
	"github.com/labib0x9/ffgif/internal/transport/http/handlers/static"
	userhandler "github.com/labib0x9/ffgif/internal/transport/http/handlers/user"
	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
	"github.com/labib0x9/ffgif/pkg/password"
	minioClientPkg "github.com/minio/minio-go/v7"
)

func getTestConfig() *config.Config {
	getEnv := func(k, defaultVal string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return defaultVal
	}

	return &config.Config{
		Version:    "1.0.0",
		Addr:       "127.0.0.1",
		Port:       8080,
		Service:    "ffgif",
		JwtSecret:  []byte(getEnv("JWT_SECRET", "integration-test-secret-key-32bytes")),
		BcryptCost: 10,
		HashPepper: getEnv("HASH_PEPPER", "test-pepper"),
		PostgreSQL: &config.PostgreSQL{
			User:          getEnv("PG_USER", "ffgif"),
			Pass:          getEnv("PG_PASSWORD", "secret"),
			Port:          getEnv("PG_PORT", "5432"),
			Addr:          getEnv("PG_ADDRESS", "127.0.0.1"),
			DatabaseName:  getEnv("PG_NAME", "ffgif"),
			SslMode:       getEnv("PG_SSLMODE", "disable"),
			SuperUser:     getEnv("PG_SUPERUSER", "postgres"),
			SuperDatabase: getEnv("PG_SUPERDB", "postgres"),
		},
		Redis: &config.Redis{
			Addr: getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		},
		Minio: &config.Minio{
			Endpoint:       getEnv("MINIO_ADDR", "127.0.0.1:9000"),
			PublicEndpoint: getEnv("MINIO_PUBLIC_ENDPOINT", "127.0.0.1:9000"),
			RootUser:       getEnv("MINIO_ROOT_USER", "minioadmin"),
			RootPass:       getEnv("MINIO_ROOT_PASSWORD", "minioadmin"),
			TempBucket:     getEnv("MINIO_TEMP_BUCKET", "uploads"),
			StorageBucket:  getEnv("MINIO_PERSIST_BUCKET", "storage"),
			TTL:            1,
			ExchangeQueue:  getEnv("MINIO_NOTIFY_EXCHANGE", "notify.upload.exchange"),
			Allowed:        []string{"*"},
		},
		RabbitMq: &config.RabbitMq{
			Addr: getEnv("RMQ_ADDR", "127.0.0.1:5672"),
			User: getEnv("RMQ_USER", "guest"),
			Pass: getEnv("RMQ_PASS", "guest"),
		},
	}
}

func probePort(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 1500*time.Millisecond)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}

func probeRealInfrastructure(cfg *config.Config) error {
	pgAddr := net.JoinHostPort(cfg.PostgreSQL.Addr, cfg.PostgreSQL.Port)
	if err := probePort(pgAddr); err != nil {
		return fmt.Errorf("postgres at %s unreachable: %w", pgAddr, err)
	}
	if err := probePort(cfg.Redis.Addr); err != nil {
		return fmt.Errorf("redis at %s unreachable: %w", cfg.Redis.Addr, err)
	}
	if err := probePort(cfg.RabbitMq.Addr); err != nil {
		return fmt.Errorf("rabbitmq at %s unreachable: %w", cfg.RabbitMq.Addr, err)
	}
	if err := probePort(cfg.Minio.Endpoint); err != nil {
		return fmt.Errorf("minio at %s unreachable: %w", cfg.Minio.Endpoint, err)
	}
	return nil
}

func TestRealInfrastructure_EndToEnd(t *testing.T) {
	cfg := getTestConfig()

	// 1. Guard check: verify real infrastructure is reachable
	if err := probeRealInfrastructure(cfg); err != nil {
		t.Skipf("Skipping real infrastructure E2E test: %v. (Start with 'docker compose up -d' to run this test)", err)
		return
	}

	t.Log("Live infrastructure detected! Initializing real DB, Redis, RabbitMQ, and MinIO clients...")

	// 2. Connect to real PostgreSQL
	dbConn := postgresinfra.NewPostgresConn(cfg.PostgreSQL)
	defer dbConn.Close()

	// 3. Connect to real Redis
	redisClient := redisinfra.Client(cfg.Redis)
	defer redisClient.Close()

	// 4. Connect to real MinIO
	minioClient := minioinfra.NewMinio(cfg.Minio)
	minioPublicClient := minioinfra.NewPublicMinio(cfg.Minio)
	_ = minioinfra.Setup(minioClient, cfg.Minio)

	// 5. Connect to real RabbitMQ
	rmq := rabbitmqinfra.NewRabbitMQ(cfg.RabbitMq)
	defer rmq.Close()
	_ = rabbitmqinfra.Setup(rmq, cfg.Minio)

	// 6. Real Repositories
	authRepo := postgresinfra.NewAuthRepository(dbConn)
	userRepo := postgresinfra.NewUserRepository(dbConn)
	verifierRepo := postgresinfra.NewVerifierRepo(dbConn)
	reseterRepo := postgresinfra.NewReseterRepo(dbConn)
	quotaRepo := postgresinfra.NewQuotaRepository(dbConn)
	lastUploadRepo := postgresinfra.NewLastVideoRepository(dbConn)
	gifRepo := postgresinfra.NewGifRepository(dbConn)
	shareRepo := postgresinfra.NewShareRepository(dbConn)
	txManager := postgresinfra.NewTxManager(dbConn)

	storageRepo := minioinfra.NewStorageRepository(minioClient, minioPublicClient, cfg.Minio)
	cacheRepo := cache.NewCache(redisClient)
	limiterRepo := ratelimitter.NewRateLimiter(redisClient)
	ffmpegProcessor := ffmpeg.NewFmeg(storageRepo)

	jwtProvider := jwtpkg.NewJwt(cfg.JwtSecret)
	hasher := password.NewHasher(cfg.HashPepper, cfg.BcryptCost)
	middlewares := middleware.NewMiddlewares(cfg, cacheRepo, *jwtProvider)
	val := validator.New()

	// 7. Real App Services
	authService := authapp.NewService(authRepo, verifierRepo, userRepo, reseterRepo, quotaRepo, cacheRepo, rmq, *jwtProvider, *hasher, txManager)
	mediaService := mediaapp.NewService(authRepo, userRepo, quotaRepo, gifRepo, shareRepo, lastUploadRepo, storageRepo, txManager, rmq, cacheRepo, ffmpegProcessor, cfg)
	shareService := shareapp.NewService(authRepo, gifRepo, shareRepo, rmq)
	userService := userapp.NewService(userRepo, quotaRepo, authRepo, txManager, *jwtProvider, *hasher)

	// 8. Real Handlers & Routing
	authH := authhandler.NewHandler(authService, middlewares, val)
	mediaH := mediahandler.NewHandler(mediaService, middlewares, val)
	shareH := sharehandler.NewHandler(shareService, middlewares, val)
	userH := userhandler.NewHandler(userService, middlewares, val)
	staticH := static.NewHandler()

	manager := middleware.NewManager()
	rateLimiter := middleware.NewRateLimiter(limiterRepo, 100, 200)
	manager.Use(
		middleware.Cors,
		middleware.Preflight,
		rateLimiter.Limit(),
	)

	mux := http.NewServeMux()
	authH.RegisterRoutes(mux, manager)
	mediaH.RegisterRoutes(mux, manager)
	shareH.RegisterRoutes(mux, manager)
	userH.RegisterRoutes(mux, manager)
	staticH.RegisterRoutes(mux, manager)
	wrappedHandler := manager.WrapMux(mux)

	// Unique test user identifiers
	testID := uuid.New().String()[:8]
	aliceEmail := fmt.Sprintf("alice_%s@realinfra.test", testID)
	bobEmail := fmt.Sprintf("bob_%s@realinfra.test", testID)

	defer func() {
		_ = authRepo.DeleteByEmail(context.Background(), aliceEmail)
		_ = authRepo.DeleteByEmail(context.Background(), bobEmail)
	}()

	// ==========================================
	// [AUTH ROUTES]
	// ==========================================

	// Route 1: POST /auth/signup
	signupBody, _ := json.Marshal(map[string]string{
		"username":         "alice" + testID,
		"fullname":         "Alice Real Infra",
		"email":            aliceEmail,
		"password":         "Password123!",
		"confirm_password": "Password123!",
	})
	signupReq := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(signupBody))
	signupRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(signupRec, signupReq)
	if signupRec.Code != http.StatusCreated {
		t.Fatalf("POST /auth/signup failed (%d): %s", signupRec.Code, signupRec.Body.String())
	}
	t.Log("✓ (1/28) POST /auth/signup passed")

	// Route 2: POST /auth/verify/resend
	resendBody, _ := json.Marshal(map[string]string{"email": aliceEmail})
	resendReq := httptest.NewRequest(http.MethodPost, "/auth/verify/resend", bytes.NewReader(resendBody))
	resendRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(resendRec, resendReq)
	if resendRec.Code != http.StatusOK {
		t.Fatalf("POST /auth/verify/resend failed (%d): %s", resendRec.Code, resendRec.Body.String())
	}
	t.Log("✓ (2/28) POST /auth/verify/resend passed")

	// Route 3: GET /auth/verify
	aliceUser, err := authRepo.GetByEmail(context.Background(), aliceEmail)
	if err != nil {
		t.Fatalf("Failed to fetch alice user from PostgreSQL: %v", err)
	}
	var verifierToken string
	err = dbConn.GetContext(context.Background(), &verifierToken, "SELECT token FROM verifiers WHERE user_id = $1 LIMIT 1", aliceUser.Id)
	if err != nil {
		t.Fatalf("Failed to retrieve verification token from PostgreSQL: %v", err)
	}
	verifyReq := httptest.NewRequest(http.MethodGet, "/auth/verify?token="+verifierToken, nil)
	verifyRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(verifyRec, verifyReq)
	if verifyRec.Code != http.StatusOK {
		t.Fatalf("GET /auth/verify failed (%d): %s", verifyRec.Code, verifyRec.Body.String())
	}
	t.Log("✓ (3/28) GET /auth/verify passed")

	// Route 4: POST /auth/login
	loginBody, _ := json.Marshal(map[string]string{
		"email":    aliceEmail,
		"password": "Password123!",
	})
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	loginRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("POST /auth/login failed (%d): %s", loginRec.Code, loginRec.Body.String())
	}
	var loginResp map[string]string
	json.NewDecoder(loginRec.Body).Decode(&loginResp)
	token := loginResp["token"]
	if token == "" {
		t.Fatalf("Expected token in login response")
	}
	t.Log("✓ (4/28) POST /auth/login passed")

	// Route 5: POST /auth/forgot-password
	forgotBody, _ := json.Marshal(map[string]string{"email": aliceEmail})
	forgotReq := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", bytes.NewReader(forgotBody))
	forgotRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(forgotRec, forgotReq)
	if forgotRec.Code != http.StatusOK {
		t.Fatalf("POST /auth/forgot-password failed (%d): %s", forgotRec.Code, forgotRec.Body.String())
	}
	t.Log("✓ (5/28) POST /auth/forgot-password passed")

	// Route 6: GET /auth/reset
	var resetToken string
	err = dbConn.GetContext(context.Background(), &resetToken, "SELECT token FROM reseter WHERE user_id = $1 LIMIT 1", aliceUser.Id)
	if err != nil {
		t.Fatalf("Failed to retrieve reset token from PostgreSQL: %v", err)
	}
	resetGetReq := httptest.NewRequest(http.MethodGet, "/auth/reset?token="+resetToken, nil)
	resetGetRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(resetGetRec, resetGetReq)
	if resetGetRec.Code != http.StatusOK {
		t.Fatalf("GET /auth/reset failed (%d): %s", resetGetRec.Code, resetGetRec.Body.String())
	}
	t.Log("✓ (6/28) GET /auth/reset passed")

	// Route 7: POST /auth/reset
	resetPostBody, _ := json.Marshal(map[string]string{
		"token":            resetToken,
		"password":         "Password123!",
		"confirm_password": "Password123!",
	})
	resetPostReq := httptest.NewRequest(http.MethodPost, "/auth/reset", bytes.NewReader(resetPostBody))
	resetPostRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(resetPostRec, resetPostReq)
	if resetPostRec.Code != http.StatusOK {
		t.Fatalf("POST /auth/reset failed (%d): %s", resetPostRec.Code, resetPostRec.Body.String())
	}
	t.Log("✓ (7/28) POST /auth/reset passed")

	// Route 8: GET /auth/logout
	logoutReq := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer "+token)
	logoutRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("GET /auth/logout failed (%d): %s", logoutRec.Code, logoutRec.Body.String())
	}
	t.Log("✓ (8/28) GET /auth/logout passed")

	// Re-login after logout to get a fresh active JWT token
	loginReq = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	loginRec = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(loginRec, loginReq)
	json.NewDecoder(loginRec.Body).Decode(&loginResp)
	token = loginResp["token"]

	// ==========================================
	// [USER ROUTES]
	// ==========================================

	// Route 9: GET /users/profile/me
	profileReq := httptest.NewRequest(http.MethodGet, "/users/profile/me", nil)
	profileReq.Header.Set("Authorization", "Bearer "+token)
	profileRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(profileRec, profileReq)
	if profileRec.Code != http.StatusOK {
		t.Fatalf("GET /users/profile/me failed (%d): %s", profileRec.Code, profileRec.Body.String())
	}
	t.Log("✓ (9/28) GET /users/profile/me passed")

	var userProf domainuser.ProfileResponse
	if err := json.Unmarshal(profileRec.Body.Bytes(), &userProf); err != nil {
		t.Fatalf("failed to decode profile body: %v", err)
	}
	userETag := userProf.UpdatedAt.Format(time.RFC3339Nano)

	// Route 10: GET /users/me/quota
	quotaReq := httptest.NewRequest(http.MethodGet, "/users/me/quota", nil)
	quotaReq.Header.Set("Authorization", "Bearer "+token)
	quotaRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(quotaRec, quotaReq)
	if quotaRec.Code != http.StatusOK {
		t.Fatalf("GET /users/me/quota failed (%d): %s", quotaRec.Code, quotaRec.Body.String())
	}
	t.Log("✓ (10/28) GET /users/me/quota passed")

	// Route 11: PATCH /users/profile/me
	newUName := "alice" + testID
	newFName := "Alice Updated"
	updateProfBody, _ := json.Marshal(domainuser.ProfileUpdateRequest{
		Username: &newUName,
		Fullname: &newFName,
		Email:    &aliceEmail,
	})
	updateProfReq := httptest.NewRequest(http.MethodPatch, "/users/profile/me", bytes.NewReader(updateProfBody))
	updateProfReq.Header.Set("Authorization", "Bearer "+token)
	updateProfReq.Header.Set("If-Match", userETag)
	updateProfRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(updateProfRec, updateProfReq)
	if updateProfRec.Code != http.StatusOK {
		t.Fatalf("PATCH /users/profile/me failed (%d): %s", updateProfRec.Code, updateProfRec.Body.String())
	}
	t.Log("✓ (11/28) PATCH /users/profile/me passed")

	// ==========================================
	// [MEDIA / UPLOAD / CONVERT ROUTES]
	// ==========================================

	// Route 12: POST /uploads
	uploadBody, _ := json.Marshal(map[string]string{"filename": "test_video.mp4"})
	uploadReq := httptest.NewRequest(http.MethodPost, "/uploads", bytes.NewReader(uploadBody))
	uploadReq.Header.Set("Authorization", "Bearer "+token)
	uploadRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(uploadRec, uploadReq)
	if uploadRec.Code != http.StatusCreated {
		t.Fatalf("POST /uploads failed (%d): %s", uploadRec.Code, uploadRec.Body.String())
	}
	var uploadResp domainmedia.UploadResult
	json.NewDecoder(uploadRec.Body).Decode(&uploadResp)
	t.Log("✓ (12/28) POST /uploads passed")

	// Upload video bytes to MinIO
	putReq, _ := http.NewRequest(http.MethodPut, uploadResp.Url, bytes.NewReader([]byte("video-raw-content-test")))
	putReq.Header.Set("Content-Type", "video/mp4")
	putResp, err := http.DefaultClient.Do(putReq)
	if err == nil {
		_ = putResp.Body.Close()
	}

	// Record last upload in PostgreSQL
	_ = lastUploadRepo.Create(context.Background(), domainmedia.LastUpload{
		UserID:      aliceUser.Id,
		FileKey:     uploadResp.Key,
		Filename:    "test_video.mp4",
		ContentType: "video/mp4",
	})

	// Route 13: GET /uploads/{key}/status
	upStatusReq := httptest.NewRequest(http.MethodGet, "/uploads/"+uploadResp.Key+"/status", nil)
	upStatusReq.Header.Set("Authorization", "Bearer "+token)
	upStatusRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(upStatusRec, upStatusReq)
	if upStatusRec.Code != http.StatusOK {
		t.Fatalf("GET /uploads/{key}/status failed (%d): %s", upStatusRec.Code, upStatusRec.Body.String())
	}
	t.Log("✓ (13/28) GET /uploads/{key}/status passed")

	// Route 14: GET /uploads/{key}/stream
	streamReq := httptest.NewRequest(http.MethodGet, "/uploads/"+uploadResp.Key+"/stream", nil)
	streamReq.Header.Set("Authorization", "Bearer "+token)
	streamRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(streamRec, streamReq)
	if streamRec.Code != http.StatusOK {
		t.Fatalf("GET /uploads/{key}/stream failed (%d): %s", streamRec.Code, streamRec.Body.String())
	}
	t.Log("✓ (14/28) GET /uploads/{key}/stream passed")

	// Route 15: GET /uploads/last
	lastVidReq := httptest.NewRequest(http.MethodGet, "/uploads/last", nil)
	lastVidReq.Header.Set("Authorization", "Bearer "+token)
	lastVidRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(lastVidRec, lastVidReq)
	if lastVidRec.Code != http.StatusOK {
		t.Fatalf("GET /uploads/last failed (%d): %s", lastVidRec.Code, lastVidRec.Body.String())
	}
	t.Log("✓ (15/28) GET /uploads/last passed")

	// Route 16: POST /convert
	convertBody, _ := json.Marshal(map[string]any{
		"upload_key": uploadResp.Key,
		"start_time": 0.0,
		"end_time":   2.0,
		"fps":        15,
		"width":      480,
		"loop":       true,
	})
	convertReq := httptest.NewRequest(http.MethodPost, "/convert", bytes.NewReader(convertBody))
	convertReq.Header.Set("Authorization", "Bearer "+token)
	convertRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(convertRec, convertReq)
	if convertRec.Code != http.StatusOK {
		t.Fatalf("POST /convert failed (%d): %s", convertRec.Code, convertRec.Body.String())
	}
	var convertResult map[string]string
	json.NewDecoder(convertRec.Body).Decode(&convertResult)
	jobId := convertResult["job_id"]
	t.Log("✓ (16/28) POST /convert passed")

	// Simulate worker creating GIF in MinIO & DB
	generatedGifKey := fmt.Sprintf("%s:%s.gif", aliceUser.Id.String(), uuid.New().String())
	tmpFile := filepath.Join(os.TempDir(), "test_real_gif.gif")
	_ = os.WriteFile(tmpFile, []byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\xff\xff\xff\x00\x00\x00!\xf9\x04\x01\x00\x00\x00\x00,\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02D\x01\x00;"), 0644)
	defer os.Remove(tmpFile)
	_ = storageRepo.Upload(context.Background(), generatedGifKey, tmpFile, "image/gif")

	_ = gifRepo.Create(context.Background(), domainmedia.Gif{
		Key:          generatedGifKey,
		Name:         "test_real.gif",
		UserId:       aliceUser.Id.String(),
		Status:       "public",
		Persist:      true,
		Url:          generatedGifKey,
		ThumbnailUrl: "thumb.jpg",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})

	jobStatus, _ := json.Marshal(map[string]string{"status": "completed", "gif_id": generatedGifKey})
	_ = cacheRepo.Set(context.Background(), "job:"+jobId, string(jobStatus), 24*time.Hour)

	// Route 17: GET /convert/{jobId}/status
	convStatusReq := httptest.NewRequest(http.MethodGet, "/convert/"+jobId+"/status", nil)
	convStatusReq.Header.Set("Authorization", "Bearer "+token)
	convStatusRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(convStatusRec, convStatusReq)
	if convStatusRec.Code != http.StatusOK {
		t.Fatalf("GET /convert/{jobId}/status failed (%d): %s", convStatusRec.Code, convStatusRec.Body.String())
	}
	t.Log("✓ (17/28) GET /convert/{jobId}/status passed")

	// ==========================================
	// [GIF MANAGEMENT ROUTES]
	// ==========================================

	// Route 18: GET /gifs/me
	getGifsReq := httptest.NewRequest(http.MethodGet, "/gifs/me", nil)
	getGifsReq.Header.Set("Authorization", "Bearer "+token)
	getGifsRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(getGifsRec, getGifsReq)
	if getGifsRec.Code != http.StatusOK {
		t.Fatalf("GET /gifs/me failed (%d): %s", getGifsRec.Code, getGifsRec.Body.String())
	}
	t.Log("✓ (18/28) GET /gifs/me passed")

	// Route 19: GET /gifs/me/{key}
	getByKeyReq := httptest.NewRequest(http.MethodGet, "/gifs/me/"+generatedGifKey, nil)
	getByKeyReq.Header.Set("Authorization", "Bearer "+token)
	getByKeyRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(getByKeyRec, getByKeyReq)
	if getByKeyRec.Code != http.StatusOK {
		t.Fatalf("GET /gifs/me/{key} failed (%d): %s", getByKeyRec.Code, getByKeyRec.Body.String())
	}
	t.Log("✓ (19/28) GET /gifs/me/{key} passed")

	// Route 20: GET /gifs/me/recents
	getRecentsReq := httptest.NewRequest(http.MethodGet, "/gifs/me/recents", nil)
	getRecentsReq.Header.Set("Authorization", "Bearer "+token)
	getRecentsRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(getRecentsRec, getRecentsReq)
	if getRecentsRec.Code != http.StatusOK {
		t.Fatalf("GET /gifs/me/recents failed (%d): %s", getRecentsRec.Code, getRecentsRec.Body.String())
	}
	t.Log("✓ (20/28) GET /gifs/me/recents passed")

	// Route 21: POST /gifs/me/recents/{key}/save
	saveRecentReq := httptest.NewRequest(http.MethodPost, "/gifs/me/recents/"+generatedGifKey+"/save", nil)
	saveRecentReq.Header.Set("Authorization", "Bearer "+token)
	saveRecentRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(saveRecentRec, saveRecentReq)
	if saveRecentRec.Code != http.StatusNoContent && saveRecentRec.Code != http.StatusOK {
		t.Fatalf("POST /gifs/me/recents/{key}/save failed (%d): %s", saveRecentRec.Code, saveRecentRec.Body.String())
	}
	t.Log("✓ (21/28) POST /gifs/me/recents/{key}/save passed")

	// Route 22: PATCH /gifs/me/{key}
	var fetchedGif domainmedia.GifResponse
	if err := json.Unmarshal(getByKeyRec.Body.Bytes(), &fetchedGif); err != nil {
		t.Fatalf("failed to decode fetched gif for etag: %v", err)
	}
	gifETag := fetchedGif.UpdatedAt.Format(time.RFC3339Nano)
	newName := "renamed.gif"
	updatePayload, _ := json.Marshal(domainmedia.GifUpdateRequest{Name: &newName})
	updateGifReq := httptest.NewRequest(http.MethodPatch, "/gifs/me/"+generatedGifKey, bytes.NewReader(updatePayload))
	updateGifReq.Header.Set("Authorization", "Bearer "+token)
	updateGifReq.Header.Set("If-Match", gifETag)
	updateGifRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(updateGifRec, updateGifReq)
	if updateGifRec.Code != http.StatusOK {
		t.Fatalf("PATCH /gifs/me/{key} failed (%d): %s", updateGifRec.Code, updateGifRec.Body.String())
	}
	t.Log("✓ (22/28) PATCH /gifs/me/{key} passed")

	// Route 23: GET /gifs/me/{key}/download
	downloadReq := httptest.NewRequest(http.MethodGet, "/gifs/me/"+generatedGifKey+"/download", nil)
	downloadReq.Header.Set("Authorization", "Bearer "+token)
	downloadRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(downloadRec, downloadReq)
	if downloadRec.Code != http.StatusOK {
		t.Fatalf("GET /gifs/me/{key}/download failed (%d): %s", downloadRec.Code, downloadRec.Body.String())
	}
	t.Log("✓ (23/28) GET /gifs/me/{key}/download passed")

	// ==========================================
	// [SHARE ROUTES]
	// ==========================================

	// Register Bob in PostgreSQL
	_, _ = authRepo.Create(context.Background(), domainauth.User{
		Username:     "bob" + testID,
		Email:        bobEmail,
		Fullname:     "Bob Real",
		PasswordHash: "$2a$10$fakehashfortestingonly",
		IsVerified:   true,
	})

	// Route 24: POST /gifs/me/{key}/shares
	shareBody, _ := json.Marshal(map[string]any{
		"shared_with": bobEmail,
		"expire_at":   time.Now().Add(24 * time.Hour),
	})
	shareReq := httptest.NewRequest(http.MethodPost, "/gifs/me/"+generatedGifKey+"/shares", bytes.NewReader(shareBody))
	shareReq.Header.Set("Authorization", "Bearer "+token)
	shareRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(shareRec, shareReq)
	if shareRec.Code != http.StatusCreated {
		t.Fatalf("POST /gifs/me/{key}/shares failed (%d): %s", shareRec.Code, shareRec.Body.String())
	}
	t.Log("✓ (24/28) POST /gifs/me/{key}/shares passed")

	// Route 25: GET /gifs/me/shares
	getSharesReq := httptest.NewRequest(http.MethodGet, "/gifs/me/shares", nil)
	getSharesReq.Header.Set("Authorization", "Bearer "+token)
	getSharesRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(getSharesRec, getSharesReq)
	if getSharesRec.Code != http.StatusOK {
		t.Fatalf("GET /gifs/me/shares failed (%d): %s", getSharesRec.Code, getSharesRec.Body.String())
	}
	t.Log("✓ (25/28) GET /gifs/me/shares passed")

	// Route 26: DELETE /gifs/me/{key}/shares/{shareWithId}
	bobUser, _ := authRepo.GetByEmail(context.Background(), bobEmail)
	deleteShareReq := httptest.NewRequest(http.MethodDelete, "/gifs/me/"+generatedGifKey+"/shares/"+bobUser.Id.String(), nil)
	deleteShareReq.Header.Set("Authorization", "Bearer "+token)
	deleteShareRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(deleteShareRec, deleteShareReq)
	if deleteShareRec.Code != http.StatusOK {
		t.Fatalf("DELETE /gifs/me/{key}/shares/{shareWithId} failed (%d): %s", deleteShareRec.Code, deleteShareRec.Body.String())
	}
	t.Log("✓ (26/29) DELETE /gifs/me/{key}/shares/{shareWithId} passed")

	// Route 27: DELETE /gifs/me/{key}
	deleteGifReq := httptest.NewRequest(http.MethodDelete, "/gifs/me/"+generatedGifKey, nil)
	deleteGifReq.Header.Set("Authorization", "Bearer "+token)
	deleteGifRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(deleteGifRec, deleteGifReq)
	if deleteGifRec.Code != http.StatusOK {
		t.Fatalf("DELETE /gifs/me/{key} failed (%d): %s", deleteGifRec.Code, deleteGifRec.Body.String())
	}
	t.Log("✓ (27/29) DELETE /gifs/me/{key} passed")

	// ==========================================
	// [PASSWORD & ACCOUNT DELETION ROUTES]
	// ==========================================

	// Route 28: PATCH /users/change-password
	changePassBody, _ := json.Marshal(map[string]string{
		"current_password": "Password123!",
		"password":         "NewSecretPass456!",
		"confirm_password": "NewSecretPass456!",
	})
	changePassReq := httptest.NewRequest(http.MethodPatch, "/users/change-password", bytes.NewReader(changePassBody))
	changePassReq.Header.Set("Authorization", "Bearer "+token)
	changePassRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(changePassRec, changePassReq)
	if changePassRec.Code != http.StatusOK {
		t.Fatalf("PATCH /users/change-password failed (%d): %s", changePassRec.Code, changePassRec.Body.String())
	}
	t.Log("✓ (28/29) PATCH /users/change-password passed")

	// Route 29: DELETE /users/me
	deleteUserBody, _ := json.Marshal(map[string]string{"password": "NewSecretPass456!"})
	deleteUserReq := httptest.NewRequest(http.MethodDelete, "/users/me", bytes.NewReader(deleteUserBody))
	deleteUserReq.Header.Set("Authorization", "Bearer "+token)
	deleteUserRec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(deleteUserRec, deleteUserReq)
	if deleteUserRec.Code != http.StatusGone && deleteUserRec.Code != http.StatusOK {
		t.Fatalf("DELETE /users/me failed (%d): %s", deleteUserRec.Code, deleteUserRec.Body.String())
	}
	t.Log("✓ (29/29) DELETE /users/me passed")

	// Clean up MinIO objects
	_ = minioClient.RemoveObject(context.Background(), cfg.Minio.TempBucket, uploadResp.Key, minioClientPkg.RemoveObjectOptions{})
	_ = minioClient.RemoveObject(context.Background(), cfg.Minio.StorageBucket, generatedGifKey, minioClientPkg.RemoveObjectOptions{})

	t.Log("🎉 100% OF ALL 29 ROUTES COVERED AND VERIFIED AGAINST REAL INFRASTRUCTURE!")
}

// EXPECTED TO FAIL: Rate limiter burst bug in internal/infra/redis/rate_limiter/rate_limiter.go.
// A brand-new token bucket key initializes with 'rate' tokens instead of 'capacity' tokens.
// When capacity=10 and rate=1, the first request on a fresh key should allow burst requests up to capacity (10),
// but the second immediate request fails because it only started with 1 token.
func TestRealInfrastructure_RateLimiter_BurstCapacity_Adversarial(t *testing.T) {
	cfg := getTestConfig()
	if err := probePort(cfg.Redis.Addr); err != nil {
		t.Skipf("Skipping Redis rate limiter test: %v", err)
		return
	}

	redisClient := redisinfra.Client(cfg.Redis)
	defer redisClient.Close()

	limiter := ratelimitter.NewRateLimiter(redisClient)
	ctx := context.Background()

	testKey := fmt.Sprintf("test_ratelimit_burst_%s", uuid.New().String())
	defer redisClient.Del(ctx, testKey)

	capacity := 5
	rate := 1
	now := time.Now().UnixMilli()

	// 1st request should succeed
	res1, err := limiter.RunScript(ctx, testKey, capacity, rate, now)
	if err != nil {
		t.Fatalf("first rate limit call failed: %v", err)
	}

	// 2nd immediate request (0ms elapsed) should ALSO succeed from burst capacity
	res2, err := limiter.RunScript(ctx, testKey, capacity, rate, now)
	if err != nil {
		t.Fatalf("second rate limit call failed: %v", err)
	}

	slice2, ok := res2.([]interface{})
	if !ok || len(slice2) < 3 {
		t.Fatalf("unexpected lua return format: %+v", res2)
	}

	allowed, ok := slice2[0].(int64)
	if !ok || allowed != 1 {
		t.Errorf("BUG DETECTED: Rate limiter failed on 2nd burst request (allowed=%d, return=%+v). Brand-new key initialized with rate=%d instead of capacity=%d",
			allowed, slice2, rate, capacity)
	}
	_ = res1
}

// EXPECTED TO FAIL: SaveRecent in internal/infra/postgres/gif_repo.go is a no-op.
func TestRealInfrastructure_GifRepo_SaveRecent_Adversarial(t *testing.T) {
	cfg := getTestConfig()
	pgAddr := net.JoinHostPort(cfg.PostgreSQL.Addr, cfg.PostgreSQL.Port)
	if err := probePort(pgAddr); err != nil {
		t.Skipf("Skipping PostgreSQL SaveRecent test: %v", err)
		return
	}

	dbConn := postgresinfra.NewPostgresConn(cfg.PostgreSQL)
	defer dbConn.Close()

	// Ensure table exists for isolated test
	_, _ = dbConn.Exec(`
		CREATE TABLE IF NOT EXISTS gifs (
			key VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			user_id VARCHAR(255) NOT NULL,
			status VARCHAR(50) DEFAULT 'public',
			persist BOOLEAN DEFAULT FALSE,
			download INT DEFAULT 0,
			url TEXT NOT NULL,
			thumbnail_url TEXT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);
	`)

	repo := postgresinfra.NewGifRepository(dbConn)
	ctx := context.Background()

	testUserId := uuid.New().String()
	testGifKey := fmt.Sprintf("test_recent_%s.gif", uuid.New().String())

	// Create gif
	gif := domainmedia.Gif{
		Key:          testGifKey,
		Name:         "Recent Test",
		UserId:       testUserId,
		Status:       "public",
		Persist:      false,
		Url:          "https://storage/" + testGifKey,
		ThumbnailUrl: "https://storage/thumb.jpg",
	}

	if err := repo.Create(ctx, gif); err != nil {
		t.Fatalf("failed to create gif: %v", err)
	}
	defer repo.Delete(ctx, testGifKey)

	// Call SaveRecent
	if err := repo.SaveRecent(ctx, testGifKey); err != nil {
		t.Fatalf("SaveRecent returned error: %v", err)
	}

	// Query recents
	recents, err := repo.GetRecents(ctx, testUserId)
	if err != nil {
		t.Fatalf("GetRecents failed: %v", err)
	}

	found := false
	for _, r := range recents {
		if r.Key == testGifKey {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("BUG DETECTED: SaveRecent is a no-op; gif %s does not appear in recents list (got %d recents)",
			testGifKey, len(recents))
	}
}

