package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	auth "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/authserver"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/authutils"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/handler"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/repository"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/service"
)

func main() {
	_ = godotenv.Load()

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET is not set, please set it via environment variable JWT_SECRET (.env file)")
	}

	jwtManager := authutils.NewJWTManager(secret)

	repo := repository.NewInMemoryUserRepo()
	authService := service.NewAuthService(repo, jwtManager)
	handler := handler.NewAuthHandler(authService)

	h := auth.HandlerWithOptions(handler, auth.StdHTTPServerOptions{
		Middlewares: []auth.MiddlewareFunc{
			authutils.JWTMiddleware(jwtManager),
		},
	})

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", h))
}
