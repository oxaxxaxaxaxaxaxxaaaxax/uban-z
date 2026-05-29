package httpx

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/domain"
)

type identityKey int

const identityCtxKey identityKey = iota

// Identity is the authenticated caller extracted from a verified JWT.
type Identity struct {
	UserID int
	Login  string
	Role   domain.Role
}

// IdentityFrom retrieves the Identity placed in ctx by the JWT middleware.
// The second return is false when the request was anonymous.
func IdentityFrom(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityCtxKey).(Identity)
	return id, ok
}

// WriteUnauthorized writes a uniform 401 JSON error.
func WriteUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"` + message + `"}`))
}

// ParseToken returns middleware that tries to parse and verify an Authorization
// Bearer token using the supplied HMAC secret. If a token is present and valid,
// the resulting Identity is stored in the request context. Anonymous requests
// (no header) pass through untouched. Malformed or invalid tokens are rejected
// with 401 — we do not silently accept bad tokens as anonymous.
func ParseToken(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				next.ServeHTTP(w, r)
				return
			}

			raw, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || raw == "" {
				WriteUnauthorized(w, "invalid authorization header")
				return
			}

			identity, err := verify(raw, secret)
			if err != nil {
				WriteUnauthorized(w, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), identityCtxKey, identity)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireIdentity returns the authenticated identity or writes a 401 and
// reports false. Handlers serving auth-required endpoints should call this
// before doing any work.
func RequireIdentity(w http.ResponseWriter, r *http.Request) (Identity, bool) {
	id, ok := IdentityFrom(r.Context())
	if !ok {
		WriteUnauthorized(w, "authentication required")
		return Identity{}, false
	}
	return id, true
}

func verify(raw string, secret []byte) (Identity, error) {
	token, err := jwt.Parse(
		raw,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return secret, nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
	)
	if err != nil || !token.Valid {
		return Identity{}, errors.New("token rejected")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return Identity{}, errors.New("unexpected claims type")
	}

	sub, err := stringClaim(claims, "sub")
	if err != nil {
		return Identity{}, err
	}
	userID, err := strconv.Atoi(sub)
	if err != nil {
		return Identity{}, errors.New("sub is not an integer")
	}

	login, err := stringClaim(claims, "login")
	if err != nil {
		return Identity{}, err
	}

	roleStr, err := stringClaim(claims, "role")
	if err != nil {
		return Identity{}, err
	}
	role, err := domain.ParseRole(roleStr)
	if err != nil {
		return Identity{}, err
	}

	return Identity{UserID: userID, Login: login, Role: role}, nil
}

func stringClaim(claims jwt.MapClaims, key string) (string, error) {
	v, ok := claims[key]
	if !ok {
		return "", errors.New("missing claim: " + key)
	}
	s, ok := v.(string)
	if !ok {
		return "", errors.New("claim is not a string: " + key)
	}
	return s, nil
}
