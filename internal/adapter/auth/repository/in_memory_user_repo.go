package repository

import (
	"errors"
	"sync"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/domain"
)

type InMemoryUserRepo struct {
	mu    sync.RWMutex
	users map[string]*domain.User // ключ = login
	idSeq int
}

func NewInMemoryUserRepo() *InMemoryUserRepo {
	return &InMemoryUserRepo{
		users: make(map[string]*domain.User),
	}
}

func (r *InMemoryUserRepo) Create(user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.Login]; exists {
		return errors.New("user already exists")
	}

	r.idSeq++
	user.ID = r.idSeq
	r.users[user.Login] = user

	return nil
}

func (r *InMemoryUserRepo) GetByLogin(login string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[login]
	if !ok {
		return nil, errors.New("user not found")
	}

	return user, nil
}
