package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/domain"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/platform/httpx"
)

var testSecret = []byte("test-secret-key")

func signToken(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(testSecret)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signed
}

func makeProtected(t *testing.T) http.Handler {
	t.Helper()
	chain := httpx.Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := httpx.RequireIdentity(w, r)
			if !ok {
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(id.Login + ":" + string(id.Role) + ":" + strconv.Itoa(id.UserID)))
		}),
		httpx.ParseToken(testSecret),
	)
	return chain
}

func TestParseToken_anonymousRequestPassesThrough(t *testing.T) {
	t.Parallel()

	chain := httpx.Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := httpx.IdentityFrom(r.Context()); ok {
				t.Error("expected no identity for anonymous request")
			}
			w.WriteHeader(http.StatusOK)
		}),
		httpx.ParseToken(testSecret),
	)

	srv := httptest.NewServer(chain)
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestRequireIdentity_missingTokenReturns401(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(makeProtected(t))
	defer srv.Close()

	resp, _ := http.Get(srv.URL)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestParseToken_validToken(t *testing.T) {
	t.Parallel()

	tok := signToken(t, jwt.MapClaims{
		"sub":   "42",
		"login": "alice",
		"role":  string(domain.RoleStudentB),
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(time.Hour).Unix(),
	})

	srv := httptest.NewServer(makeProtected(t))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestParseToken_invalidPaths(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		header string
		token  func(t *testing.T) string
	}{
		{
			name:   "missing Bearer prefix",
			header: "Token abc",
		},
		{
			name:   "malformed token body",
			header: "Bearer not-a-jwt",
		},
		{
			name: "expired token",
			token: func(t *testing.T) string {
				return signToken(t, jwt.MapClaims{
					"sub":   "1",
					"login": "x",
					"role":  string(domain.RoleStudentB),
					"iat":   time.Now().Add(-2 * time.Hour).Unix(),
					"exp":   time.Now().Add(-time.Hour).Unix(),
				})
			},
		},
		{
			name: "wrong signature",
			token: func(t *testing.T) string {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"sub": "1", "login": "x", "role": string(domain.RoleAdmin),
					"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
				})
				signed, _ := token.SignedString([]byte("other-secret"))
				return signed
			},
		},
		{
			name: "unknown role",
			token: func(t *testing.T) string {
				return signToken(t, jwt.MapClaims{
					"sub": "1", "login": "x", "role": "ROOT",
					"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
				})
			},
		},
		{
			name: "non-integer sub",
			token: func(t *testing.T) string {
				return signToken(t, jwt.MapClaims{
					"sub": "abc", "login": "x", "role": string(domain.RoleStudentB),
					"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
				})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(makeProtected(t))
			defer srv.Close()

			req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			} else {
				req.Header.Set("Authorization", "Bearer "+tc.token(t))
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("do: %v", err)
			}
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", resp.StatusCode)
			}
		})
	}
}
