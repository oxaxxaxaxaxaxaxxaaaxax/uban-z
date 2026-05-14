package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

const (
	StorageMemory   = "memory"
	StoragePostgres = "postgres"
)

type Config struct {
	Port            string        `env:"PORT" env-default:"8080"`
	Storage         string        `env:"STORAGE" env-default:"memory"`
	DatabaseURL     string        `env:"DATABASE_URL"`
	LogLevel        string        `env:"LOG_LEVEL" env-default:"info"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" env-default:"5s"`
}

func Load() (Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return Config{}, fmt.Errorf("read env: %w", err)
	}

	cfg.Storage = strings.ToLower(cfg.Storage)
	cfg.LogLevel = strings.ToLower(cfg.LogLevel)

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	switch c.Storage {
	case StorageMemory:
	case StoragePostgres:
		if c.DatabaseURL == "" {
			return errors.New("STORAGE=postgres requires DATABASE_URL")
		}
	default:
		return fmt.Errorf("unknown STORAGE %q (expected memory|postgres)", c.Storage)
	}

	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("unknown LOG_LEVEL %q (expected debug|info|warn|error)", c.LogLevel)
	}

	if c.ShutdownTimeout <= 0 {
		return errors.New("SHUTDOWN_TIMEOUT must be positive")
	}

	return nil
}
