package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Port             string        `env:"PORT" env-default:"8080"`
	DatabaseURL      string        `env:"DATABASE_URL"`
	LogLevel         string        `env:"LOG_LEVEL" env-default:"info"`
	ShutdownTimeout  time.Duration `env:"SHUTDOWN_TIMEOUT" env-default:"5s"`
	EventsEnabled    bool          `env:"EVENTS_ENABLED" env-default:"false"`
	RabbitMQURL      string        `env:"RABBITMQ_URL"`
	RabbitMQExchange string        `env:"RABBITMQ_EXCHANGE" env-default:"booking.events"`
	JWTSecret        string        `env:"JWT_SECRET"`

	ParserBaseURL         string        `env:"PARSER_BASE_URL" env-default:"https://table.nsu.ru"`
	ParserTimeout         time.Duration `env:"PARSER_TIMEOUT" env-default:"30s"`
	ParserWeeksAhead      int           `env:"PARSER_WEEKS_AHEAD" env-default:"16"`
	ParserDefaultBuilding string        `env:"PARSER_DEFAULT_BUILDING" env-default:"НГУ"`
	ParserDefaultCapacity int           `env:"PARSER_DEFAULT_CAPACITY" env-default:"30"`
	ParserTimezone        string        `env:"PARSER_TIMEZONE" env-default:"Asia/Novosibirsk"`
}

func Load() (Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return Config{}, fmt.Errorf("read env: %w", err)
	}

	cfg.LogLevel = strings.ToLower(cfg.LogLevel)

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if c.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("unknown LOG_LEVEL %q (expected debug|info|warn|error)", c.LogLevel)
	}

	if c.ShutdownTimeout <= 0 {
		return errors.New("SHUTDOWN_TIMEOUT must be positive")
	}

	if c.EventsEnabled && c.RabbitMQURL == "" {
		return errors.New("EVENTS_ENABLED=true requires RABBITMQ_URL")
	}

	if c.JWTSecret == "" {
		return errors.New("JWT_SECRET is required")
	}

	if c.ParserBaseURL == "" {
		return errors.New("PARSER_BASE_URL is required")
	}
	if c.ParserTimeout <= 0 {
		return errors.New("PARSER_TIMEOUT must be positive")
	}
	if c.ParserWeeksAhead <= 0 {
		return errors.New("PARSER_WEEKS_AHEAD must be positive")
	}
	if c.ParserDefaultCapacity <= 0 {
		return errors.New("PARSER_DEFAULT_CAPACITY must be positive")
	}
	if strings.TrimSpace(c.ParserDefaultBuilding) == "" {
		return errors.New("PARSER_DEFAULT_BUILDING is required")
	}
	if _, err := time.LoadLocation(c.ParserTimezone); err != nil {
		return fmt.Errorf("load PARSER_TIMEZONE: %w", err)
	}

	return nil
}
