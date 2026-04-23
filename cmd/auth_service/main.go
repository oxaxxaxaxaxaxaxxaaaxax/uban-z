package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	auth "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/authserver"
	httpHandler "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/http/handler"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/http/middleware"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/jwt"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/repository/in_memory"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/service"
)

func main() {
	_ = godotenv.Load()

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET is not set")
	}

	jwtManager := jwt.NewJWTManager(secret)
	repo := in_memory.NewInMemoryUserRepo()
	authService := service.NewAuthService(repo, jwtManager)

	authHandler := httpHandler.NewAuthHandler(authService)
	userHandler := httpHandler.NewUserHandler(authService)

	authRouter := auth.HandlerWithOptions(authHandler, auth.StdHTTPServerOptions{
		BaseURL: "/api",
	})

	tokenMw := middleware.JWTMiddleware(jwtManager)

	mux := http.NewServeMux()

	//добавить auth_middleware для регистрации и login
	mux.Handle("/api/auth/", authRouter)

	mux.Handle("GET /api/users", tokenMw(http.HandlerFunc(userHandler.GetUsers)))
	mux.Handle("GET /api/users/{id}", tokenMw(http.HandlerFunc(userHandler.GetUserByID)))
	mux.Handle("PUT /api/users/{id}", tokenMw(http.HandlerFunc(userHandler.UpdateUser)))
	mux.Handle("DELETE /api/users/{id}", tokenMw(http.HandlerFunc(userHandler.DeleteUser)))

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
