package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	auth "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/authserver"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/service"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(s *service.AuthService) *AuthHandler {
	return &AuthHandler{service: s}
}

var _ auth.ServerInterface = (*AuthHandler)(nil)

func (h *AuthHandler) PostAuthRegister(w http.ResponseWriter, r *http.Request) {
	var req auth.RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeStatusError(w, http.StatusBadRequest, "bad request")
		return
	}
	if isBlank(req.Login) || isBlank(req.Password) || isBlank(req.Role) || isBlank(req.FullName) {
		writeStatusError(w, http.StatusBadRequest, "bad request")
		return
	}

	user, err := h.service.Register(req.Login, req.Password, req.Role, req.FullName)
	if err != nil {
		writeUserError(w, err)
		return
	}

	resp := auth.UserResponse{
		Id:       &user.ID,
		Login:    &user.Login,
		Role:     &user.Role,
		FullName: &user.FullName,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) PostAuthLogin(w http.ResponseWriter, r *http.Request) {
	var req auth.LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeStatusError(w, http.StatusBadRequest, "bad request")
		return
	}
	if isBlank(req.Login) || isBlank(req.Password) {
		writeStatusError(w, http.StatusBadRequest, "bad request")
		return
	}

	token, err := h.service.Login(req.Login, req.Password)
	if err != nil {
		writeStatusError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	resp := auth.LoginResponse{
		Token: &token,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) GetAuthMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireAuthenticated(w, r)
	if !ok {
		return
	}

	user, err := h.service.GetUserByID(claims.UserID)
	if err != nil {
		writeUserError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(user))
}

func isBlank(value string) bool {
	return strings.TrimSpace(value) == ""
}
