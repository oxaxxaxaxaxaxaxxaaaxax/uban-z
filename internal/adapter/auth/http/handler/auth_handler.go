package handler

import (
	"encoding/json"
	"net/http"

	auth "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/authserver"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/service"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(s *service.AuthService) *AuthHandler {
	return &AuthHandler{service: s}
}

func (h *AuthHandler) PostAuthRegister(w http.ResponseWriter, r *http.Request) {
	var req auth.RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	user, err := h.service.Register(req.Login, req.Password, req.Role)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp := auth.UserResponse{
		Id:    &user.ID,
		Login: &user.Login,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) PostAuthLogin(w http.ResponseWriter, r *http.Request) {
	var req auth.LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	token, err := h.service.Login(req.Login, req.Password)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	resp := auth.LoginResponse{
		Token: &token,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
