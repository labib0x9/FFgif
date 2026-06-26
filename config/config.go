package config

import (
	"bytes"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type PostgreSQL struct {
	User         string
	Pass         string
	Port         string
	Addr         string
	DatabaseName string
	SslMode      string

	SuperUser     string
	SuperDatabase string
}

type RedisConfig struct {
	Addr string
	Pass string
	User string
}

type MinioConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
}

type RabbitMq struct {
	Addr string
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

	PostgreSQL   *PostgreSQL
	RedisConfig  *RedisConfig
	MailtrapUser string
	MailtrapPass string
	Email        string
	MinioConfig  *MinioConfig
	RabbitMq     *RabbitMq
}

var configuration *Config

func loadConfig() {
	if err := godotenv.Load(".env"); err != nil {
		log.Panic(err)
	}

	version := os.Getenv("VERSION")
	if version == "" {
		log.Panic("VERSION")
	}

	addr := os.Getenv("ADDR")
	if addr == "" {
		log.Panic("ADDR")
	}

	portS := os.Getenv("PORT")
	if portS == "" {
		log.Panic("PORT")
	}

	port, err := strconv.Atoi(portS)
	if err != nil {
		log.Fatalln(err)
	}

	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	if bytes.Equal(jwtSecret, []byte("")) == true {
		log.Panic("JWT_SECRET")
	}

	pepper := os.Getenv("HASH_PEPPER")
	if pepper == "" {
		log.Panic("HASH_PEPPER")
	}

	bcryptCostStr := os.Getenv("BCRYPT_COST")
	if bcryptCostStr == "" {
		log.Panic("BCRYPT_COST")
	}

	bcryptCost, err := strconv.Atoi(bcryptCostStr)
	if err != nil {
		log.Panic(err)
	}

	serviceName := os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		log.Panic("SERVICE_NAME")
	}

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		log.Panic("DB_USER")
	}

	dbPass := os.Getenv("DB_PASSWORD")
	if dbPass == "" {
		log.Panic("DB_PASSWORD")
	}

	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		log.Panic("DB_PORT")
	}

	dbAddr := os.Getenv("DB_ADDRESS")
	if dbAddr == "" {
		log.Panic("DB_ADDRESS")
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		log.Panic("DB_NAME")
	}

	dbSSlmode := os.Getenv("DB_SSLMODE")
	if dbSSlmode == "" {
		log.Panic("DB_SSLMODE")
	}

	dbSuperUser := os.Getenv("DB_SUPERUSER")
	if dbSSlmode == "" {
		log.Panic("DB_SUPERUSER")
	}

	dbSuperDb := os.Getenv("DB_SUPERDB")
	if dbSSlmode == "" {
		log.Panic("DB_SUPERDB")
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		log.Panic("REDIS_ADDR")
	}

	// redisUser := os.Getenv("REDIS_USER")
	// if redisUser == "" {
	// 	log.Panic("REDIS_USER")
	// }

	// redisPass := os.Getenv("REDIS_PASS")
	// if redisPass == "" {
	// 	log.Panic("REDIS_PASS")
	// }

	email := os.Getenv("EMAIL")
	if email == "" {
		log.Panic("EMAIL")
	}

	mailtrapUser := os.Getenv("MAILTRAP_USERNAME")
	if mailtrapUser == "" {
		log.Panic("MAILTRAP_USERNAME")
	}

	mailtrapPass := os.Getenv("MAILTRAP_PASSWORD")
	if mailtrapPass == "" {
		log.Panic("MAILTRAP_PASSWORD")
	}

	endpoint := os.Getenv("ENDPOINT")
	if endpoint == "" {
		log.Panic("ENDPOINT")
	}

	accessKeyId := os.Getenv("ACCESS_KEY_ID")
	if accessKeyId == "" {
		log.Panic("ACCESS_KEY_ID")
	}

	secretAccessKey := os.Getenv("SECRET_ACCESS_KEY")
	if secretAccessKey == "" {
		log.Panic("SECRET_ACCESS_KEY")
	}

	bucketName := os.Getenv("BUCKET_NAME")
	if bucketName == "" {
		log.Panic("BUCKET_NAME")
	}

	rmqAddr := os.Getenv("RMQ_ADDR")
	if rmqAddr == "" {
		log.Panic("RMQ_ADDR")
	}

	rmqUser := os.Getenv("RMQ_USER")
	if rmqUser == "" {
		log.Panic("RMQ_USER")
	}

	rmqPass := os.Getenv("RMQ_PASS")
	if rmqPass == "" {
		log.Panic("RMQ_PASS")
	}

	configuration = &Config{
		Version:    version,
		Addr:       addr,
		Port:       port,
		Service:    serviceName,
		JwtSecret:  jwtSecret,
		BcryptCost: bcryptCost,
		HashPepper: pepper,
		PostgreSQL: &PostgreSQL{
			User:          dbUser,
			Pass:          dbPass,
			Addr:          dbAddr,
			Port:          dbPort,
			DatabaseName:  dbName,
			SslMode:       dbSSlmode,
			SuperUser:     dbSuperUser,
			SuperDatabase: dbSuperDb,
		},

		RedisConfig: &RedisConfig{
			Addr: redisAddr,
			// User: redisUser,
			// Pass: redisPass,
		},
		Email:        email,
		MailtrapUser: mailtrapUser,
		MailtrapPass: mailtrapPass,
		MinioConfig: &MinioConfig{
			Endpoint:        endpoint,
			AccessKeyID:     accessKeyId,
			SecretAccessKey: secretAccessKey,
			BucketName:      bucketName,
		},
		RabbitMq: &RabbitMq{
			Addr: rmqAddr,
			User: rmqUser,
			Pass: rmqPass,
		},
	}
}

func GetConfig() *Config {
	if configuration == nil {
		loadConfig()
	}
	return configuration
}
