package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	gatewayhttp "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/gateway/http"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/platform/httpx"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/platform/logging"
)

const defaultShutdownTimeout = 5 * time.Second

func main() {
	port := env("PORT", "8080")
	logLevel := env("LOG_LEVEL", "info")
	shutdownTimeout := durationEnv("SHUTDOWN_TIMEOUT", defaultShutdownTimeout)

	logger, err := logging.New(logLevel)
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("logger init failed", slog.Any("err", err))
		os.Exit(1)
	}

	gateway, err := gatewayhttp.NewHandler(gatewayhttp.Config{
		AuthServiceURL:    env("AUTH_SERVICE_URL", "http://localhost:8081"),
		BookingServiceURL: env("BOOKING_SERVICE_URL", "http://localhost:8082"),
		Timeout:           durationEnv("UPSTREAM_TIMEOUT", 30*time.Second),
	})
	if err != nil {
		logger.Error("gateway init failed", slog.Any("err", err))
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}` + "\n"))
	})
	mux.Handle("/", gateway)

	router := httpx.Chain(
		mux,
		httpx.RequestID,
		httpx.RecoverPanic(logger),
		httpx.AccessLog(logger),
	)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: shutdownTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("gateway shutdown failed", slog.Any("err", err))
		}
	}()

	logger.Info("api gateway starting",
		slog.String("addr", server.Addr),
		slog.String("auth_service_url", env("AUTH_SERVICE_URL", "http://localhost:8081")),
		slog.String("booking_service_url", env("BOOKING_SERVICE_URL", "http://localhost:8082")),
	)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("gateway failed", slog.Any("err", err))
		os.Exit(1)
	}
}

func env(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
