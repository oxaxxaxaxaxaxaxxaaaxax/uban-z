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

func (r *InMemoryUserRepo) Update(user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// ищем существующего пользователя по ID
	var existing *domain.User
	var oldLogin string
	for login, u := range r.users {
		if u.ID == user.ID {
			existing = u
			oldLogin = login
			break
		}
	}

	if existing == nil {
		return domain.ErrUserNotFound
	}

	// если login изменился — проверяем, что новый не занят
	if user.Login != oldLogin {
		if _, taken := r.users[user.Login]; taken {
			return domain.ErrLoginAlreadyTaken
		}
		delete(r.users, oldLogin)
	}

	userCopy := *user
	r.users[user.Login] = &userCopy
	return nil
}

func (r *InMemoryUserRepo) Delete(id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for login, user := range r.users {
		if user.ID == id {
			delete(r.users, login)
			return nil
		}
	}

	return domain.ErrUserNotFound
}

func (r *InMemoryUserRepo) List() ([]*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*domain.User, 0, len(r.users))
	for _, user := range r.users {
		userCopy := *user
		result = append(result, &userCopy)
	}

	return result, nil
}
