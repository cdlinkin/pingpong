package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
	LogLevel    string
	LogFormat   string
	UploadDir   string
	EnvFile     bool
	HTTPAddr    string
	Storage     string
	S3Endpoint  string
	S3AccessKey string
	S3SecretKey string
	S3Bucket    string
	S3UseSSL    bool
}

func Load() (Config, error) {
	envErr := godotenv.Load()

	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		HTTPAddr:    os.Getenv("HTTP_ADDR"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		LogLevel:    os.Getenv("LOG_LEVEL"),
		LogFormat:   os.Getenv("LOG_FORMAT"),
		UploadDir:   os.Getenv("UPLOAD_DIR"),
		Storage:     os.Getenv("STORAGE"),
		S3Endpoint:  os.Getenv("S3_ENDPOINT"),
		S3AccessKey: os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey: os.Getenv("S3_SECRET_KEY"),
		S3Bucket:    os.Getenv("S3_BUCKET"),
		EnvFile:     envErr == nil,
	}

	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	}

	if cfg.UploadDir == "" {
		cfg.UploadDir = "uploads"
	}

	if cfg.Storage == "" {
		cfg.Storage = "local"
	}

	if cfg.S3Bucket == "" {
		cfg.S3Bucket = "pingpong"
	}

	if v := os.Getenv("S3_USE_SSL"); v != "" {
		useSSL, err := strconv.ParseBool(v)
		if err != nil {
			return cfg, fmt.Errorf(
				"S3_USE_SSL must be true or false, got %q",
				v,
			)
		}
		cfg.S3UseSSL = useSSL
	}

	if cfg.JWTSecret == "" {
		return cfg, errors.New("JWT_SECRET is not set")
	}

	switch cfg.Storage {
	case "local":
	case "s3":
		if cfg.S3Endpoint == "" || cfg.S3AccessKey == "" || cfg.S3SecretKey == "" {
			return cfg, errors.New("STORAGE=s3 needs S3_ENDPOINT, S3_ACCESS_KEY and S3_SECRET_KEY")
		}
	default:
		return cfg, fmt.Errorf(
			"STORAGE must be local or s3, got %q",
			cfg.Storage,
		)
	}

	return cfg, nil
}
