package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/joho/godotenv"
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
	TempBucket     string
	StorageBucket  string
	TTL            int
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
	if err := godotenv.Load(".env"); err != nil {
		if !os.IsNotExist(err) {
			log.Panic(err)
		}
	}

	fn := func(name string) string {
		value := os.Getenv(name)
		if value == "" {
			log.Panic(name)
		}
		return value
	}

	port, err := strconv.Atoi(fn("PORT"))
	if err != nil {
		log.Fatalln(err)
	}

	bcryptCost, err := strconv.Atoi(fn("BCRYPT_COST"))
	if err != nil {
		log.Panic(err)
	}

	minioTTL, err := strconv.Atoi(fn("MINIO_TEMP_BUCKET_TTL_DAYS"))
	if err != nil {
		log.Panic(err)
	}

	origins := strings.Split(fn("MINIO_API_CORS_ALLOW_ORIGIN"), ",")

	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	smtpPort, err := strconv.Atoi(fn("SMTP_PORT"))
	if err != nil {
		log.Fatalln(err)
	}

	configuration = &Config{
		Version:    fn("VERSION"),
		Addr:       fn("ADDR"),
		Port:       port,
		Service:    fn("SERVICE_NAME"),
		JwtSecret:  []byte(fn("JWT_SECRET")),
		BcryptCost: bcryptCost,
		HashPepper: fn("HASH_PEPPER"),
		PostgreSQL: &PostgreSQL{
			User:          fn("PG_USER"),
			Pass:          fn("PG_PASSWORD"),
			Port:          fn("PG_PORT"),
			Addr:          fn("PG_ADDRESS"),
			DatabaseName:  fn("PG_NAME"),
			SslMode:       fn("PG_SSLMODE"),
			SuperUser:     fn("PG_SUPERUSER"),
			SuperDatabase: fn("PG_SUPERDB"),
		},
		Redis: &Redis{
			Addr: fn("REDIS_ADDR"),
		},
		Email: fn("EMAIL"),
		Mailtrap: &Mailtrap{
			User: fn("MAILTRAP_USERNAME"),
			Pass: fn("MAILTRAP_PASSWORD"),
		},
		Minio: &Minio{
			Endpoint:       fn("MINIO_ADDR"),
			PublicEndpoint: fn("MINIO_PUBLIC_ENDPOINT"),
			RootUser:       fn("MINIO_ROOT_USER"),
			RootPass:       fn("MINIO_ROOT_PASSWORD"),
			StorageBucket:  fn("MINIO_PERSIST_BUCKET"),
			TempBucket:     fn("MINIO_TEMP_BUCKET"),
			TTL:            minioTTL,
			ExchangeQueue:  fn("MINIO_NOTIFY_EXCHANGE"),
			Allowed:        origins,
		},
		RabbitMq: &RabbitMq{
			Addr: fn("RMQ_ADDR"),
			User: fn("RMQ_USER"),
			Pass: fn("RMQ_PASS"),
		},
		SMTP: &SMTP{
			Host: fn("SMTP_HOST"),
			Port: smtpPort,
			User: fn("SMTP_USER"),
			Pass: fn("SMTP_PASS"),
		},
	}
}

func GetConfig() *Config {
	once.Do(func() {
		loadConfig()
	})
	return configuration
}
