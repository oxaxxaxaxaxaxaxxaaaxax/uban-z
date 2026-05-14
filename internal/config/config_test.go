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
		Port:            "8080",
		Storage:         "memory",
		LogLevel:        "info",
		ShutdownTimeout: 5 * time.Second,
	}

	cases := []struct {
		name      string
		mutate    func(*config.Config)
		wantError string
	}{
		{name: "memory defaults are valid", mutate: func(*config.Config) {}},
		{
			name:   "postgres with DATABASE_URL is valid",
			mutate: func(c *config.Config) { c.Storage = "postgres"; c.DatabaseURL = "postgres://x" },
		},
		{
			name:      "postgres without DATABASE_URL fails",
			mutate:    func(c *config.Config) { c.Storage = "postgres" },
			wantError: "DATABASE_URL",
		},
		{
			name:      "unknown storage fails",
			mutate:    func(c *config.Config) { c.Storage = "mongo" },
			wantError: "unknown STORAGE",
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
	t.Setenv("STORAGE", "MEMORY")
	t.Setenv("LOG_LEVEL", "DEBUG")
	t.Setenv("SHUTDOWN_TIMEOUT", "10s")
	t.Setenv("DATABASE_URL", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load err = %v", err)
	}
	if cfg.Port != "9090" {
		t.Fatalf("Port = %q, want 9090", cfg.Port)
	}
	if cfg.Storage != "memory" {
		t.Fatalf("Storage = %q, want memory (lowercased)", cfg.Storage)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("LogLevel = %q, want debug (lowercased)", cfg.LogLevel)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Fatalf("ShutdownTimeout = %v, want 10s", cfg.ShutdownTimeout)
	}
}

func TestLoad_failsOnInvalidEnv(t *testing.T) {
	t.Setenv("STORAGE", "postgres")
	t.Setenv("DATABASE_URL", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load err = nil, want error about DATABASE_URL")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("err = %q, want substring DATABASE_URL", err.Error())
	}
}
