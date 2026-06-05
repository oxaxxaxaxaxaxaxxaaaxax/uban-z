package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
)

func New(level string) (*slog.Logger, error) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		return nil, fmt.Errorf("unknown log level %q", level)
	}

	writer := io.Writer(os.Stdout)
	if addr := os.Getenv("LOGSTASH_ADDR"); addr != "" {
		writer = io.MultiWriter(writer, newAsyncTCPWriter(addr))
	}

	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: lvl})
	logger := slog.New(handler)
	if service := os.Getenv("SERVICE_NAME"); service != "" {
		logger = logger.With(slog.String("service", service))
	}
	return logger, nil
}
