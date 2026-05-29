package ports

import (
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/domain"
)

type UserRepository interface {
	Create(user *domain.User) error
	GetByLogin(login string) (*domain.User, error)
	GetByID(id int) (*domain.User, error)
}
