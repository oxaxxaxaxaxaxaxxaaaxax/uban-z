package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	bookingserver "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/bookingserver"
	bookinghttp "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/http"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/inmemory"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/config"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/service"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/platform/httpx"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/platform/logging"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("config load failed", slog.Any("err", err))
		os.Exit(1)
	}

	logger, err := logging.New(cfg.LogLevel)
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("logger init failed", slog.Any("err", err))
		os.Exit(1)
	}

	store := inmemory.NewStore()
	useCase := service.New(store, store)
	handler := bookinghttp.NewHandler(useCase, logger)

	router := httpx.Chain(
		bookingserver.Handler(handler),
		httpx.RequestID,
		httpx.RecoverPanic(logger),
		httpx.AccessLog(logger),
	)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: cfg.ShutdownTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("server shutdown failed", slog.Any("err", err))
		}
	}()

	logger.Info("booking service starting",
		slog.String("addr", server.Addr),
		slog.String("storage", cfg.Storage),
		slog.String("log_level", cfg.LogLevel),
	)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server failed", slog.Any("err", err))
		os.Exit(1)
	}
}
