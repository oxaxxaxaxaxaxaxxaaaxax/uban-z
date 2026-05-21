package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpHandler "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/http/handler"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/http/middleware"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/jwt"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/repository/in_memory"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/domain"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/service"
)

func TestUserRoutesAccessControl(t *testing.T) {
	t.Parallel()

	mux, studentToken, adminToken := newUserTestServer(t)

	t.Run("student can get own profile without password leak", func(t *testing.T) {
		t.Parallel()

		rec := performRequest(mux, http.MethodGet, "/api/users/me", "", studentToken)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		body := rec.Body.String()
		if !strings.Contains(body, `"login":"student"`) {
			t.Fatalf("body = %s, want student login", body)
		}
		if strings.Contains(strings.ToLower(body), "password") {
			t.Fatalf("body leaks password: %s", body)
		}
	})

	t.Run("student cannot get user by id", func(t *testing.T) {
		t.Parallel()

		rec := performRequest(mux, http.MethodGet, "/api/users/1", "", studentToken)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
		}
	})

	t.Run("admin can get user by id", func(t *testing.T) {
		t.Parallel()

		rec := performRequest(mux, http.MethodGet, "/api/users/2", "", adminToken)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})

	t.Run("self update cannot change role", func(t *testing.T) {
		t.Parallel()

		rec := performRequest(mux, http.MethodPut, "/api/users/me", `{"role":"admin"}`, studentToken)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
		}
	})
}

func newUserTestServer(t *testing.T) (*http.ServeMux, string, string) {
	t.Helper()

	repo := in_memory.NewInMemoryUserRepo()
	tokenManager := jwt.NewJWTManager("test-secret")
	authService := service.NewAuthService(repo, tokenManager)

	admin, err := authService.Register("admin", "secret", domain.RoleAdmin)
	if err != nil {
		t.Fatalf("register admin: %v", err)
	}
	student, err := authService.Register("student", "secret", domain.RoleStudentB)
	if err != nil {
		t.Fatalf("register student: %v", err)
	}

	adminToken, err := tokenManager.Generate(admin)
	if err != nil {
		t.Fatalf("generate admin token: %v", err)
	}
	studentToken, err := tokenManager.Generate(student)
	if err != nil {
		t.Fatalf("generate student token: %v", err)
	}

	userHandler := httpHandler.NewUserHandler(authService)
	tokenMw := middleware.JWTMiddleware(tokenManager)

	mux := http.NewServeMux()
	mux.Handle("GET /api/users/me", tokenMw(http.HandlerFunc(userHandler.GetMe)))
	mux.Handle("PUT /api/users/me", tokenMw(http.HandlerFunc(userHandler.UpdateMe)))
	mux.Handle("GET /api/users/{id}", tokenMw(http.HandlerFunc(userHandler.GetUserByID)))

	return mux, studentToken, adminToken
}

func performRequest(handler http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}
