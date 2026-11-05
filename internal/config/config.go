package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
	LogLevel    string
	LogFormat   string
	UploadDir   string
	EnvFile     bool
}

func Load() (Config, error) {
	envErr := godotenv.Load()

	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		LogLevel:    os.Getenv("LOG_LEVEL"),
		LogFormat:   os.Getenv("LOG_FORMAT"),
		UploadDir:   os.Getenv("UPLOAD_DIR"),
		EnvFile:     envErr == nil,
	}

	if cfg.UploadDir == "" {
		cfg.UploadDir = "uploads"
	}

	if cfg.JWTSecret == "" {
		return cfg, errors.New("JWT_SECRET is not set")
	}

	return cfg, nil
}
