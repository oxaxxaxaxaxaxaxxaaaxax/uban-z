package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpHandler "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/http/handler"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/http/middleware"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/jwt"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/repository/in_memory"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/service"
)

func TestAuthRoutesRegisterLoginAndMe(t *testing.T) {
	t.Parallel()

	mux := newAuthTestServer(t)

	registerBody := `{"login":"student","password":"secret","role":"student_b","full_name":"Student Name"}`
	registerRec := performRequest(mux, http.MethodPost, "/api/auth/register", registerBody, "")
	if registerRec.Code != http.StatusOK {
		t.Fatalf("register status = %d, want %d; body = %s", registerRec.Code, http.StatusOK, registerRec.Body.String())
	}

	registerResp := registerRec.Body.String()
	for _, want := range []string{`"login":"student"`, `"role":"student_b"`, `"full_name":"Student Name"`, `"id":`} {
		if !strings.Contains(registerResp, want) {
			t.Fatalf("register body = %s, want %s", registerResp, want)
		}
	}
	if strings.Contains(strings.ToLower(registerResp), "password") {
		t.Fatalf("register body leaks password: %s", registerResp)
	}

	loginRec := performRequest(mux, http.MethodPost, "/api/auth/login", `{"login":"student","password":"secret"}`, "")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d; body = %s", loginRec.Code, http.StatusOK, loginRec.Body.String())
	}

	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(loginRec.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if loginResp.Token == "" {
		t.Fatal("login token is empty")
	}

	meRec := performRequest(mux, http.MethodGet, "/api/auth/me", "", loginResp.Token)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me status = %d, want %d; body = %s", meRec.Code, http.StatusOK, meRec.Body.String())
	}
	for _, want := range []string{`"login":"student"`, `"role":"student_b"`, `"full_name":"Student Name"`} {
		if !strings.Contains(meRec.Body.String(), want) {
			t.Fatalf("me body = %s, want %s", meRec.Body.String(), want)
		}
	}
}

func TestAuthRoutesRejectInvalidRequests(t *testing.T) {
	t.Parallel()

	mux := newAuthTestServer(t)

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{
			name:   "register missing full name",
			method: http.MethodPost,
			path:   "/api/auth/register",
			body:   `{"login":"student","password":"secret","role":"student_b"}`,
			want:   http.StatusBadRequest,
		},
		{
			name:   "register invalid role",
			method: http.MethodPost,
			path:   "/api/auth/register",
			body:   `{"login":"student","password":"secret","role":"user","full_name":"Student Name"}`,
			want:   http.StatusBadRequest,
		},
		{
			name:   "login missing password",
			method: http.MethodPost,
			path:   "/api/auth/login",
			body:   `{"login":"student"}`,
			want:   http.StatusBadRequest,
		},
		{
			name:   "removed user update route",
			method: http.MethodPut,
			path:   "/api/users/me",
			body:   `{"role":"admin"}`,
			want:   http.StatusNotFound,
		},
		{
			name:   "removed admin user route",
			method: http.MethodGet,
			path:   "/api/users/1",
			want:   http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rec := performRequest(mux, tc.method, tc.path, tc.body, "")
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

func TestAuthRoutesDuplicateAndWrongPassword(t *testing.T) {
	t.Parallel()

	mux := newAuthTestServer(t)
	body := `{"login":"student","password":"secret","role":"student_b","full_name":"Student Name"}`
	if rec := performRequest(mux, http.MethodPost, "/api/auth/register", body, ""); rec.Code != http.StatusOK {
		t.Fatalf("first register status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if rec := performRequest(mux, http.MethodPost, "/api/auth/register", body, ""); rec.Code != http.StatusConflict {
		t.Fatalf("duplicate register status = %d, want %d; body = %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
	if rec := performRequest(mux, http.MethodPost, "/api/auth/login", `{"login":"student","password":"wrong"}`, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password status = %d, want %d; body = %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestAuthMeRequiresValidToken(t *testing.T) {
	t.Parallel()

	mux := newAuthTestServer(t)

	if rec := performRequest(mux, http.MethodGet, "/api/auth/me", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing token status = %d, want %d; body = %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
	if rec := performRequest(mux, http.MethodGet, "/api/auth/me", "", "not-a-jwt"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("invalid token status = %d, want %d; body = %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func newAuthTestServer(t *testing.T) *http.ServeMux {
	t.Helper()

	repo := in_memory.NewInMemoryUserRepo()
	tokenManager := jwt.NewJWTManager("test-secret")
	authService := service.NewAuthService(repo, tokenManager)
	authHandler := httpHandler.NewAuthHandler(authService)
	tokenMw := middleware.JWTMiddleware(tokenManager)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/register", authHandler.PostAuthRegister)
	mux.HandleFunc("POST /api/auth/login", authHandler.PostAuthLogin)
	mux.Handle("GET /api/auth/me", tokenMw(http.HandlerFunc(authHandler.GetAuthMe)))

	return mux
}

func performRequest(handler http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}
