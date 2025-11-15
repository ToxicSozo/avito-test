package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

var EnvVars = struct {
	Port         string
	DatabaseURL  string
	ReadTimeout  string
	WriteTimeout string
	IdleTimeout  string
}{
	Port:         "PORT",
	DatabaseURL:  "DATABASE_URL",
	ReadTimeout:  "READ_TIMEOUT",
	WriteTimeout: "WRITE_TIMEOUT",
	IdleTimeout:  "IDLE_TIMEOUT",
}

type Config struct {
	Port        string
	DatabaseURL string

	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("", ""),

		ReadTimeout:  parseDuration("", 15*time.Second),
		WriteTimeout: parseDuration("", 15*time.Second),
		IdleTimeout:  parseDuration("", 60*time.Second),
	}
}

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}

func parseDuration(key string, defaultVal time.Duration) time.Duration {
	if valueStr := getEnv(key, ""); valueStr != "" {
		d, err := time.ParseDuration(valueStr)
		if err != nil {
			return defaultVal
		}
		return d
	}
	return defaultVal
}
