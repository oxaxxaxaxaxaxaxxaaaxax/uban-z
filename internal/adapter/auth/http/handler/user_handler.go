package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	auth "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/authserver"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/http/middleware"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/domain"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/ports"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/service"
)

type UserHandler struct {
	service *service.AuthService
}

func NewUserHandler(s *service.AuthService) *UserHandler {
	return &UserHandler{service: s}
}

// GET /users
func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	users, err := h.service.ListUsers()
	if err != nil {
		writeUserError(w, err)
		return
	}

	resp := make([]auth.UserResponse, 0, len(users))
	for _, user := range users {
		resp = append(resp, toUserResponse(user))
	}

	writeJSON(w, http.StatusOK, resp)
}

// GET /users/me
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
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

// PUT /users/me
func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireAuthenticated(w, r)
	if !ok {
		return
	}

	req, err := decodeUpdateUserRequest(r)
	if err != nil {
		writeStatusError(w, http.StatusBadRequest, "bad request")
		return
	}
	if req.Role != nil {
		writeStatusError(w, http.StatusForbidden, "role can be changed only by admin")
		return
	}

	user, err := h.service.UpdateUser(claims.UserID, stringValue(req.Login), stringValue(req.Password), "")
	if err != nil {
		writeUserError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(user))
}

// DELETE /users/me
func (h *UserHandler) DeleteMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireAuthenticated(w, r)
	if !ok {
		return
	}

	if err := h.service.DeleteUser(claims.UserID); err != nil {
		writeUserError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /users/{id}
func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeStatusError(w, http.StatusBadRequest, "invalid id")
		return
	}

	user, err := h.service.GetUserByID(id)
	if err != nil {
		writeUserError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(user))
}

// PUT /users/{id}
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeStatusError(w, http.StatusBadRequest, "invalid id")
		return
	}

	req, err := decodeUpdateUserRequest(r)
	if err != nil {
		writeStatusError(w, http.StatusBadRequest, "bad request")
		return
	}

	role := stringValue(req.Role)
	if req.Role != nil && role == "" {
		writeUserError(w, domain.ErrInvalidRole)
		return
	}

	user, err := h.service.UpdateUser(id, stringValue(req.Login), stringValue(req.Password), role)
	if err != nil {
		writeUserError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(user))
}

// DELETE /users/{id}
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeStatusError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.DeleteUser(id); err != nil {
		writeUserError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type updateUserRequest struct {
	Login    *string `json:"login,omitempty"`
	Password *string `json:"password,omitempty"`
	Role     *string `json:"role,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func decodeUpdateUserRequest(r *http.Request) (updateUserRequest, error) {
	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return updateUserRequest{}, err
	}
	return req, nil
}

func requireAuthenticated(w http.ResponseWriter, r *http.Request) (*ports.Claims, bool) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, "missing token claims")
		return nil, false
	}
	return claims, true
}

func requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	claims, ok := requireAuthenticated(w, r)
	if !ok {
		return false
	}
	if claims.Role != domain.RoleAdmin {
		writeStatusError(w, http.StatusForbidden, "admin role required")
		return false
	}
	return true
}

func toUserResponse(user *domain.User) auth.UserResponse {
	id := user.ID
	login := user.Login
	role := user.Role

	return auth.UserResponse{
		Id:    &id,
		Login: &login,
		Role:  &role,
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func writeUserError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError

	switch {
	case errors.Is(err, domain.ErrInvalidRole):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrUserAlreadyExists), errors.Is(err, domain.ErrLoginAlreadyTaken):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrUserNotFound):
		status = http.StatusNotFound
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
