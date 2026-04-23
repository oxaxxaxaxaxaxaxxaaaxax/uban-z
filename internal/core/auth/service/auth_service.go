package service

import (
	"errors"

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

func (s *AuthService) Register(login, password, role string) (*domain.User, error) {
	user := &domain.User{
		Login:    login,
		Password: password, // позже: hash
		Role:     role,
	}

	err := s.repo.Create(user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) Login(login, password string) (string, error) {
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

func (s *AuthService) UpdateUser(id int, login, password, role string) (*domain.User, error) {
	user, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if login != "" {
		user.Login = login
	}
	if password != "" {
		user.Password = password // позже: hash
	}
	if role != "" {
		user.Role = role
	}

	if err := s.repo.Update(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) DeleteUser(id int) error {
	return s.repo.Delete(id)
}

func (s *AuthService) ListUsers() ([]*domain.User, error) {
	return s.repo.List()
}
