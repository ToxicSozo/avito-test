package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string

	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration

	LogLevel string

	AdminToken string
	UserToken  string
}

func Load() (*Config, error) {
	// Для локальной разработки: грузим .env, но не падаем, если его нет.
	_ = godotenv.Load()

	cfg := &Config{
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", ""),
		ReadTimeout:     parseDuration("READ_TIMEOUT", 5*time.Second),
		WriteTimeout:    parseDuration("WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:     parseDuration("IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout: parseDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		AdminToken:      getEnv("ADMIN_TOKEN", ""),
		UserToken:       getEnv("USER_TOKEN", ""),
	}

	// Минимальная валидация — это уже бизнес-правила конфигурации
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL must be set")
	}
	if cfg.AdminToken == "" || cfg.UserToken == "" {
		return nil, fmt.Errorf("ADMIN_TOKEN and USER_TOKEN must be set")
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return defaultVal
}

func parseDuration(key string, defaultVal time.Duration) time.Duration {
	val := getEnv(key, "")
	if val == "" {
		return defaultVal
	}

	d, err := time.ParseDuration(val)
	if err != nil {
		return defaultVal
	}

	return d
}
