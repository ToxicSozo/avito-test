package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Logger   LoggerConfig
}

type ServerConfig struct {
	Port            string        `env:"PORT" envDefault:"8080"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
}

type DatabaseConfig struct {
	Host     string `env:"DB_HOST" envDefault:"localhost"`
	Port     string `env:"DB_PORT" envDefault:"5432"`
	User     string `env:"DB_USER" envDefault:"postgres"`
	Password string `env:"DB_PASSWORD" envDefault:"postgres"`
	Database string `env:"DB_NAME" envDefault:"reviewers"`
}

type LoggerConfig struct {
	Level string `env:"LOG_LEVEL" envDefault:"info"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	var cfg Config

	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}

	return &cfg, nil
}
