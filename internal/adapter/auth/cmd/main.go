package main

import (
	"log"
	"net/http"

	auth "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/authserver"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/handler"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/repository"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/service"
)

func main() {
	repo := repository.NewInMemoryUserRepo()
	authService := service.NewAuthService(repo)
	handler := handler.NewAuthHandler(authService)

	h := auth.Handler(handler)

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", h))
}
