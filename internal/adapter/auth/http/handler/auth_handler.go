package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	auth "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/authserver"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/service"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/platform/httpx"
)

type AuthHandler struct {
	service *service.AuthService
	logger  *slog.Logger
}

func NewAuthHandler(s *service.AuthService, loggers ...*slog.Logger) *AuthHandler {
	logger := slog.Default()
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	}
	return &AuthHandler{service: s, logger: logger}
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
		h.logger.WarnContext(r.Context(), "auth.register.failed",
			slog.String("event", "auth.register.failed"),
			slog.String("action", "User registration failed"),
			slog.String("actor", req.Login),
			slog.String("details", "User "+req.Login+" could not register as "+req.Role+": "+err.Error()),
			slog.String("request_id", httpx.RequestIDFrom(r.Context())),
			slog.String("login", req.Login),
			slog.String("role", req.Role),
			slog.String("err", err.Error()),
		)
		writeUserError(w, err)
		return
	}

	h.logger.InfoContext(r.Context(), "auth.register.succeeded",
		slog.String("event", "auth.register.succeeded"),
		slog.String("action", "User registered"),
		slog.String("actor", user.Login),
		slog.String("details", "User "+user.Login+" registered as "+user.Role),
		slog.String("request_id", httpx.RequestIDFrom(r.Context())),
		slog.Int("user_id", user.ID),
		slog.String("login", user.Login),
		slog.String("role", user.Role),
	)

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
		h.logger.WarnContext(r.Context(), "auth.login.failed",
			slog.String("event", "auth.login.failed"),
			slog.String("action", "Login failed"),
			slog.String("actor", req.Login),
			slog.String("details", "User "+req.Login+" failed to log in"),
			slog.String("request_id", httpx.RequestIDFrom(r.Context())),
			slog.String("login", req.Login),
			slog.String("err", err.Error()),
		)
		writeStatusError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	h.logger.InfoContext(r.Context(), "auth.login.succeeded",
		slog.String("event", "auth.login.succeeded"),
		slog.String("action", "User logged in"),
		slog.String("actor", req.Login),
		slog.String("details", "User "+req.Login+" logged in successfully"),
		slog.String("request_id", httpx.RequestIDFrom(r.Context())),
		slog.String("login", req.Login),
	)

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
