package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	auth "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/authserver"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/http/middleware"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/domain"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/ports"
)

type errorResponse struct {
	Error string `json:"error"`
}

func requireAuthenticated(w http.ResponseWriter, r *http.Request) (*ports.Claims, bool) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, "missing token claims")
		return nil, false
	}
	return claims, true
}

func toUserResponse(user *domain.User) auth.UserResponse {
	id := user.ID
	login := user.Login
	role := user.Role
	fullName := user.FullName

	return auth.UserResponse{
		Id:       &id,
		Login:    &login,
		Role:     &role,
		FullName: &fullName,
	}
}

func writeUserError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError

	switch {
	case errors.Is(err, domain.ErrInvalidRole), errors.Is(err, domain.ErrInvalidUserData), errors.Is(err, domain.ErrUserNotFound):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrUserAlreadyExists):
		status = http.StatusConflict
	}

	writeStatusError(w, status, err.Error())
}

func writeStatusError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
