package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	httpHandler "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/http/handler"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/http/middleware"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/jwt"
	authpostgres "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/repository/postgres"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/service"
)

const dbConnectTimeout = 10 * time.Second

func main() {
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

	pool, err := openPostgres(databaseURL)
	if err != nil {
		log.Fatalf("postgres connect failed: %v", err)
	}
	defer pool.Close()

	jwtManager := jwt.NewJWTManager(secret)
	repo := authpostgres.NewUserRepository(pool)
	authService := service.NewAuthService(repo, jwtManager)

	authHandler := httpHandler.NewAuthHandler(authService)

	tokenMw := middleware.JWTMiddleware(jwtManager)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/auth/register", authHandler.PostAuthRegister)
	mux.HandleFunc("POST /api/auth/login", authHandler.PostAuthLogin)
	mux.Handle("GET /api/auth/me", tokenMw(http.HandlerFunc(authHandler.GetAuthMe)))

	log.Printf("Server started on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
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
