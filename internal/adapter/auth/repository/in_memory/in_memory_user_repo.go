package in_memory

import (
	"sync"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/domain"
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
		return domain.ErrUserAlreadyExists
	}

	r.idSeq++
	user.ID = r.idSeq
	userCopy := *user
	r.users[user.Login] = &userCopy

	return nil
}

func (r *InMemoryUserRepo) GetByLogin(login string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[login]
	if !ok {
		return nil, domain.ErrUserNotFound
	}

	userCopy := *user
	return &userCopy, nil
}

func (r *InMemoryUserRepo) GetByID(id int) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if user.ID == id {
			userCopy := *user
			return &userCopy, nil
		}
	}

	return nil, domain.ErrUserNotFound
}
