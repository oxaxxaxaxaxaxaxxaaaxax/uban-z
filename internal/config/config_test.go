package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/config"
)

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	base := config.Config{
		Port:             "8080",
		DatabaseURL:      "postgres://x",
		LogLevel:         "info",
		ShutdownTimeout:  5 * time.Second,
		RabbitMQExchange: "booking.events",
		JWTSecret:        "test-secret",

		ParserBaseURL:         "https://table.nsu.ru",
		ParserTimeout:         30 * time.Second,
		ParserWeeksAhead:      16,
		ParserDefaultBuilding: "НГУ",
		ParserDefaultCapacity: 30,
		ParserTimezone:        "Asia/Novosibirsk",
	}

	cases := []struct {
		name      string
		mutate    func(*config.Config)
		wantError string
	}{
		{name: "valid config", mutate: func(*config.Config) {}},
		{
			name:      "missing DATABASE_URL fails",
			mutate:    func(c *config.Config) { c.DatabaseURL = "" },
			wantError: "DATABASE_URL",
		},
		{
			name:      "unknown log level fails",
			mutate:    func(c *config.Config) { c.LogLevel = "verbose" },
			wantError: "LOG_LEVEL",
		},
		{
			name:      "zero shutdown timeout fails",
			mutate:    func(c *config.Config) { c.ShutdownTimeout = 0 },
			wantError: "SHUTDOWN_TIMEOUT",
		},
		{
			name:   "events enabled with RABBITMQ_URL is valid",
			mutate: func(c *config.Config) { c.EventsEnabled = true; c.RabbitMQURL = "amqp://localhost" },
		},
		{
			name:      "events enabled without RABBITMQ_URL fails",
			mutate:    func(c *config.Config) { c.EventsEnabled = true },
			wantError: "RABBITMQ_URL",
		},
		{
			name:      "missing JWT_SECRET fails",
			mutate:    func(c *config.Config) { c.JWTSecret = "" },
			wantError: "JWT_SECRET",
		},
		{
			name: "invalid parser timezone fails",
			mutate: func(c *config.Config) {
				c.ParserTimezone = "Mars/Olympus"
			},
			wantError: "PARSER_TIMEZONE",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := base
			tc.mutate(&cfg)
			err := cfg.Validate()
			switch {
			case tc.wantError == "" && err != nil:
				t.Fatalf("unexpected err = %v", err)
			case tc.wantError != "" && err == nil:
				t.Fatalf("err = nil, want substring %q", tc.wantError)
			case tc.wantError != "" && !strings.Contains(err.Error(), tc.wantError):
				t.Fatalf("err = %q, want substring %q", err.Error(), tc.wantError)
			}
		})
	}
}

func TestLoad_readsAndNormalisesEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("LOG_LEVEL", "DEBUG")
	t.Setenv("SHUTDOWN_TIMEOUT", "10s")
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("JWT_SECRET", "test")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load err = %v", err)
	}
	if cfg.Port != "9090" {
		t.Fatalf("Port = %q, want 9090", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("LogLevel = %q, want debug (lowercased)", cfg.LogLevel)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Fatalf("ShutdownTimeout = %v, want 10s", cfg.ShutdownTimeout)
	}
}

func TestLoad_failsWithoutDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "test")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load err = nil, want error about DATABASE_URL")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("err = %q, want substring DATABASE_URL", err.Error())
	}
}

func TestLoad_failsWithoutJWTSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("JWT_SECRET", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load err = nil, want error about JWT_SECRET")
	}
	if !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("err = %q, want substring JWT_SECRET", err.Error())
	}
}
