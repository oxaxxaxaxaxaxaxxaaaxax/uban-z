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

	"github.com/jackc/pgx/v5/pgxpool"

	bookingserver "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/bookingserver"
	bookinghttp "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/http"
	bookingpostgres "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/postgres"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/config"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/service"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/platform/httpx"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/platform/logging"
)

const dbConnectTimeout = 10 * time.Second

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

	pool, err := openPostgres(cfg.DatabaseURL)
	if err != nil {
		logger.Error("postgres connect failed", slog.Any("err", err))
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("connected to postgres")

	store := bookingpostgres.NewStoreFromPool(pool)
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
		slog.String("log_level", cfg.LogLevel),
	)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server failed", slog.Any("err", err))
		os.Exit(1)
	}
}

func openPostgres(url string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbConnectTimeout)
	defer cancel()

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
