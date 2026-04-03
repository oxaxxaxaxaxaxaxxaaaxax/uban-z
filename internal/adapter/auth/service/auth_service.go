package service

import (
	"errors"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/authutils"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/domain"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/repository"
)

type AuthService struct {
	repo       repository.UserRepository
	jwtManager *authutils.JWTManager
}

func NewAuthService(r repository.UserRepository, jwt *authutils.JWTManager) *AuthService {
	return &AuthService{
		repo:       r,
		jwtManager: jwt,
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

	return s.jwtManager.Generate(user.ID, user.Login, user.Role)
}
