package config

import (
	"errors"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

type PostgreSQL struct {
	User          string
	Pass          string
	Port          string
	Addr          string
	DatabaseName  string
	SslMode       string
	SuperUser     string
	SuperDatabase string
}

type Redis struct {
	Addr string
	Pass string
	User string
}

type Minio struct {
	Endpoint       string
	PublicEndpoint string
	RootUser       string
	RootPass       string
	PresignedUser  string
	PresignedPass  string
	TempBucket     string
	StorageBucket  string
	TTL            int
	MaxUploadBytes int64
	Secure         bool
	ExchangeQueue  string
	Allowed        []string
}

type RabbitMq struct {
	Addr string
	User string
	Pass string
}

type Mailtrap struct {
	User string
	Pass string
}

type SMTP struct {
	Host string
	Port int
	User string
	Pass string
}

type Config struct {
	Version    string
	Addr       string
	Port       int
	Service    string
	JwtSecret  []byte
	BcryptCost int
	HashPepper string
	PostgreSQL *PostgreSQL
	Redis      *Redis
	Email      string
	Mailtrap   *Mailtrap
	SMTP       *SMTP
	Minio      *Minio
	RabbitMq   *RabbitMq
}

var (
	configuration *Config
	once          sync.Once
)

func loadConfig() {
	viper.SetConfigFile(".env")
	if err := viper.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) && !errors.As(err, &viper.ConfigFileNotFoundError{}) {
			var pathErr *os.PathError
			if !errors.As(err, &pathErr) {
				log.Panic(err)
			}
		}
	}

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// defaults for optional values
	viper.SetDefault("BCRYPT_COST", 12)
	viper.SetDefault("PG_SSLMODE", "disable")
	viper.SetDefault("MINIO_TEMP_BUCKET_TTL_DAYS", 1)
	viper.SetDefault("ADDR", "0.0.0.0")
	viper.SetDefault("MINIO_MAX_UPLOAD_BYTES", int64(104857600)) // 100MB
	viper.SetDefault("MINIO_USE_TLS", false)

	required := func(key string) string {
		val := viper.GetString(key)
		if val == "" {
			log.Panic(key)
		}
		return val
	}

	origins := strings.Split(required("MINIO_API_CORS_ALLOW_ORIGIN"), ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	presignedUser := viper.GetString("MINIO_PRESIGNED_USER")
	if presignedUser == "" {
		presignedUser = required("MINIO_ROOT_USER")
	}
	presignedPass := viper.GetString("MINIO_PRESIGNED_PASSWORD")
	if presignedPass == "" {
		presignedPass = required("MINIO_ROOT_PASSWORD")
	}

	configuration = &Config{
		Version:    required("VERSION"),
		Addr:       viper.GetString("ADDR"),
		Port:       viper.GetInt("PORT"),
		Service:    required("SERVICE_NAME"),
		JwtSecret:  []byte(required("JWT_SECRET")),
		BcryptCost: viper.GetInt("BCRYPT_COST"),
		HashPepper: required("HASH_PEPPER"),
		PostgreSQL: &PostgreSQL{
			User:          required("PG_USER"),
			Pass:          required("PG_PASSWORD"),
			Port:          required("PG_PORT"),
			Addr:          required("PG_ADDRESS"),
			DatabaseName:  required("PG_NAME"),
			SslMode:       viper.GetString("PG_SSLMODE"),
			SuperUser:     required("PG_SUPERUSER"),
			SuperDatabase: required("PG_SUPERDB"),
		},
		Redis: &Redis{
			Addr: required("REDIS_ADDR"),
			User: viper.GetString("REDIS_USER"),
			Pass: viper.GetString("REDIS_PASSWORD"),
		},
		Email: required("EMAIL"),
		Mailtrap: &Mailtrap{
			User: required("MAILTRAP_USERNAME"),
			Pass: required("MAILTRAP_PASSWORD"),
		},
		Minio: &Minio{
			Endpoint:       required("MINIO_ADDR"),
			PublicEndpoint: required("MINIO_PUBLIC_ENDPOINT"),
			RootUser:       required("MINIO_ROOT_USER"),
			RootPass:       required("MINIO_ROOT_PASSWORD"),
			PresignedUser:  presignedUser,
			PresignedPass:  presignedPass,
			StorageBucket:  required("MINIO_PERSIST_BUCKET"),
			TempBucket:     required("MINIO_TEMP_BUCKET"),
			TTL:            viper.GetInt("MINIO_TEMP_BUCKET_TTL_DAYS"),
			MaxUploadBytes: viper.GetInt64("MINIO_MAX_UPLOAD_BYTES"),
			Secure:         viper.GetBool("MINIO_USE_TLS"),
			ExchangeQueue:  required("MINIO_NOTIFY_EXCHANGE"),
			Allowed:        origins,
		},
		RabbitMq: &RabbitMq{
			Addr: required("RMQ_ADDR"),
			User: required("RMQ_USER"),
			Pass: required("RMQ_PASS"),
		},
		SMTP: &SMTP{
			Host: required("SMTP_HOST"),
			Port: viper.GetInt("SMTP_PORT"),
			User: required("SMTP_USER"),
			Pass: required("SMTP_PASS"),
		},
	}
}

func GetConfig() *Config {
	once.Do(func() {
		loadConfig()
	})
	return configuration
}
