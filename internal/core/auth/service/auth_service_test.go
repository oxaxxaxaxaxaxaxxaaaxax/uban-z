package service_test

import (
	"errors"
	"testing"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/repository/in_memory"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/domain"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/ports"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/service"
)

func TestAuthServiceRegisterValidatesRole(t *testing.T) {
	t.Parallel()

	authService := service.NewAuthService(in_memory.NewInMemoryUserRepo(), fakeTokenManager{})

	user, err := authService.Register("student", "secret", domain.RoleStudentB)
	if err != nil {
		t.Fatalf("Register valid role err = %v", err)
	}
	if user.ID == 0 {
		t.Fatal("Register did not assign user id")
	}

	_, err = authService.Register("bad-role", "secret", "user")
	if !errors.Is(err, domain.ErrInvalidRole) {
		t.Fatalf("Register invalid role err = %v, want ErrInvalidRole", err)
	}
}

func TestAuthServiceLogin(t *testing.T) {
	t.Parallel()

	authService := service.NewAuthService(in_memory.NewInMemoryUserRepo(), fakeTokenManager{token: "jwt-token"})
	if _, err := authService.Register("teacher", "secret", domain.RoleTeacher); err != nil {
		t.Fatalf("Register err = %v", err)
	}

	token, err := authService.Login("teacher", "secret")
	if err != nil {
		t.Fatalf("Login err = %v", err)
	}
	if token != "jwt-token" {
		t.Fatalf("token = %q, want jwt-token", token)
	}

	if _, err := authService.Login("teacher", "wrong"); err == nil {
		t.Fatal("Login err = nil, want invalid credentials")
	}
}

func TestAuthServiceUpdateUser(t *testing.T) {
	t.Parallel()

	authService := service.NewAuthService(in_memory.NewInMemoryUserRepo(), fakeTokenManager{})
	user, err := authService.Register("student", "secret", domain.RoleStudentB)
	if err != nil {
		t.Fatalf("Register err = %v", err)
	}

	updated, err := authService.UpdateUser(user.ID, "student_new", "", domain.RoleStudentM)
	if err != nil {
		t.Fatalf("UpdateUser err = %v", err)
	}
	if updated.Login != "student_new" || updated.Role != domain.RoleStudentM {
		t.Fatalf("updated = %#v, want changed login and role", updated)
	}

	_, err = authService.UpdateUser(user.ID, "", "", "unknown")
	if !errors.Is(err, domain.ErrInvalidRole) {
		t.Fatalf("UpdateUser invalid role err = %v, want ErrInvalidRole", err)
	}
}

type fakeTokenManager struct {
	token string
}

func (m fakeTokenManager) Generate(*domain.User) (string, error) {
	return m.token, nil
}

func (m fakeTokenManager) Verify(string) (*ports.Claims, error) {
	return nil, nil
}
