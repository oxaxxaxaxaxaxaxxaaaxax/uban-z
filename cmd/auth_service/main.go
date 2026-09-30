package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	httpHandler "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/http/handler"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/http/middleware"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/jwt"
	authpostgres "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/repository/postgres"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/service"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/platform/httpx"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/platform/logging"
)

const dbConnectTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load()

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET is not set")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	logger, err := logging.New(logLevel)
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("logger init failed", slog.Any("err", err))
		os.Exit(1)
	}
	shutdownTimeout := 5 * time.Second
	if value := os.Getenv("SHUTDOWN_TIMEOUT"); value != "" {
		shutdownTimeout, err = time.ParseDuration(value)
		if err != nil || shutdownTimeout <= 0 {
			logger.Error("SHUTDOWN_TIMEOUT must be a positive duration")
			os.Exit(1)
		}
	}

	pool, err := openPostgres(databaseURL)
	if err != nil {
		logger.Error("postgres connect failed", slog.Any("err", err))
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("connected to postgres")

	jwtManager := jwt.NewJWTManager(secret)
	repo := authpostgres.NewUserRepository(pool)
	authService := service.NewAuthService(repo, jwtManager)

	authHandler := httpHandler.NewAuthHandler(authService, logger)

	tokenMw := middleware.JWTMiddleware(jwtManager)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/auth/register", authHandler.PostAuthRegister)
	mux.HandleFunc("POST /api/auth/login", authHandler.PostAuthLogin)
	mux.Handle("GET /api/auth/me", tokenMw(http.HandlerFunc(authHandler.GetAuthMe)))

	router := httpx.Chain(
		mux,
		httpx.RequestID,
		httpx.RecoverPanic(logger),
		httpx.AccessLog(logger),
	)

	logger.Info("auth service starting",
		slog.String("addr", ":"+port),
		slog.String("log_level", logLevel),
	)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: shutdownTimeout,
	}
	if err := httpx.ListenAndServe(ctx, server, shutdownTimeout); err != nil {
		logger.Error("auth server failed", slog.Any("err", err))
		return err
	}
	return nil
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
