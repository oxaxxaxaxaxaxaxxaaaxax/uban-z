package repository

import "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/domain"

type UserRepository interface {
	Create(user *domain.User) error
	GetByLogin(login string) (*domain.User, error)
}
