package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/ports"
)

type contextKey string

const UserContextKey contextKey = "user"

func ClaimsFromContext(ctx context.Context) (*ports.Claims, bool) {
	claims, ok := ctx.Value(UserContextKey).(*ports.Claims)
	return claims, ok
}

func JWTMiddleware(manager ports.TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, http.StatusUnauthorized, "missing token")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeError(w, http.StatusUnauthorized, "invalid token format")
				return
			}

			claims, err := manager.Verify(parts[1])
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{Error: message})
}
