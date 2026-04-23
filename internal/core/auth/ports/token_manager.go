package ports

import (
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/domain"
)

type Claims struct {
	UserID int
	Login  string
	Role   string
}

type TokenManager interface {
	Generate(user *domain.User) (string, error)
	Verify(tokenStr string) (*Claims, error) // возвращаем свой тип
}
