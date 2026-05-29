package service

import (
	"errors"
	"strings"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/domain"
	ports2 "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/ports"
)

type AuthService struct {
	repo         ports2.UserRepository
	tokenManager ports2.TokenManager
}

func NewAuthService(r ports2.UserRepository, jwt ports2.TokenManager) *AuthService {
	return &AuthService{
		repo:         r,
		tokenManager: jwt,
	}
}

func (s *AuthService) Register(login, password, role, fullName string) (*domain.User, error) {
	login = strings.TrimSpace(login)
	role = strings.TrimSpace(role)
	fullName = strings.TrimSpace(fullName)

	if login == "" || strings.TrimSpace(password) == "" || fullName == "" {
		return nil, domain.ErrInvalidUserData
	}
	if !domain.IsValidRole(role) {
		return nil, domain.ErrInvalidRole
	}

	user := &domain.User{
		Login:    login,
		Password: password, // позже: hash
		Role:     role,
		FullName: fullName,
	}

	err := s.repo.Create(user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) Login(login, password string) (string, error) {
	login = strings.TrimSpace(login)
	if login == "" || strings.TrimSpace(password) == "" {
		return "", errors.New("invalid credentials")
	}

	user, err := s.repo.GetByLogin(login)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	if user.Password != password {
		return "", errors.New("invalid credentials")
	}

	return s.tokenManager.Generate(user)
}

func (s *AuthService) GetUserByID(id int) (*domain.User, error) {
	return s.repo.GetByID(id)
}
