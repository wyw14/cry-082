package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

type Config struct {
	Environment     string        `validate:"required,oneof=development test production"`
	HTTPAddress     string        `validate:"required"`
	DatabaseURL     string        `validate:"omitempty,url"`
	Timezone        string        `validate:"required"`
	AccessTokenTTL  time.Duration `validate:"gt=0"`
	AccessTokenKey  string        `validate:"required,min=32"`
	RefreshTokenTTL time.Duration `validate:"gt=0"`
	RequestTimeout  time.Duration `validate:"gt=0"`
	ShutdownTimeout time.Duration `validate:"gt=0"`
	FileRoot        string        `validate:"required"`
	MaximumFileSize int64         `validate:"gt=0"`
	SeedDemo        bool
	LogLevel        string `validate:"required,oneof=debug info warn error"`
}

func Load() (Config, error) {
	config := Config{
		Environment:     env("APP_ENV", "development"),
		HTTPAddress:     env("HTTP_ADDR", ":8080"),
		DatabaseURL:     strings.TrimSpace(os.Getenv("DATABASE_URL")),
		Timezone:        env("APP_TIMEZONE", "Asia/Shanghai"),
		AccessTokenTTL:  duration("ACCESS_TOKEN_TTL", 15*time.Minute),
		AccessTokenKey:  env("ACCESS_TOKEN_KEY", "development-only-access-token-key-082"),
		RefreshTokenTTL: duration("REFRESH_TOKEN_TTL", 7*24*time.Hour),
		RequestTimeout:  duration("REQUEST_TIMEOUT", 5*time.Second),
		ShutdownTimeout: duration("SHUTDOWN_TIMEOUT", 15*time.Second),
		FileRoot:        env("FILE_ROOT", "./uploads"),
		MaximumFileSize: int64Value("MAX_FILE_SIZE", 10<<20),
		SeedDemo:        boolean("SEED_DEMO", true),
		LogLevel:        env("LOG_LEVEL", "info"),
	}
	if _, err := time.LoadLocation(config.Timezone); err != nil {
		return Config{}, fmt.Errorf("APP_TIMEZONE: %w", err)
	}
	if config.Environment == "production" && config.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required in production")
	}
	if config.Environment == "production" && strings.TrimSpace(os.Getenv("ACCESS_TOKEN_KEY")) == "" {
		return Config{}, errors.New("ACCESS_TOKEN_KEY is required in production")
	}
	if err := validator.New().Struct(config); err != nil {
		return Config{}, fmt.Errorf("configuration validation: %w", err)
	}
	return config, nil
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
func duration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
func boolean(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
func int64Value(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}
