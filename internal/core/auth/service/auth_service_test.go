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

	user, err := authService.Register("student", "secret", domain.RoleStudentB, "Student Name")
	if err != nil {
		t.Fatalf("Register valid role err = %v", err)
	}
	if user.ID == 0 {
		t.Fatal("Register did not assign user id")
	}

	if user.FullName != "Student Name" {
		t.Fatalf("FullName = %q, want Student Name", user.FullName)
	}

	_, err = authService.Register("bad-role", "secret", "user", "Bad Role")
	if !errors.Is(err, domain.ErrInvalidRole) {
		t.Fatalf("Register invalid role err = %v, want ErrInvalidRole", err)
	}

	_, err = authService.Register("", "secret", domain.RoleStudentB, "Student Name")
	if !errors.Is(err, domain.ErrInvalidUserData) {
		t.Fatalf("Register empty login err = %v, want ErrInvalidUserData", err)
	}

	_, err = authService.Register("student-empty-name", "secret", domain.RoleStudentB, "")
	if !errors.Is(err, domain.ErrInvalidUserData) {
		t.Fatalf("Register empty full name err = %v, want ErrInvalidUserData", err)
	}
}

func TestAuthServiceLogin(t *testing.T) {
	t.Parallel()

	authService := service.NewAuthService(in_memory.NewInMemoryUserRepo(), fakeTokenManager{token: "jwt-token"})
	if _, err := authService.Register("teacher", "secret", domain.RoleTeacher, "Teacher Name"); err != nil {
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

func TestAuthServiceGetUserByID(t *testing.T) {
	t.Parallel()

	authService := service.NewAuthService(in_memory.NewInMemoryUserRepo(), fakeTokenManager{})
	user, err := authService.Register("student", "secret", domain.RoleStudentB, "Student Name")
	if err != nil {
		t.Fatalf("Register err = %v", err)
	}

	got, err := authService.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("GetUserByID err = %v", err)
	}
	if got.Login != "student" || got.Password != "secret" || got.Role != domain.RoleStudentB || got.FullName != "Student Name" {
		t.Fatalf("user = %#v, want registered user", got)
	}

	_, err = authService.GetUserByID(user.ID + 100)
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("GetUserByID missing err = %v, want ErrUserNotFound", err)
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
