package in_memory

import (
	"errors"
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

func (r *InMemoryUserRepo) GetByID(id int) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if user.ID == id {
			return user, nil
		}
	}

	return nil, errors.New("user not found")
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
		return errors.New("user not found")
	}

	// если login изменился — проверяем, что новый не занят
	if user.Login != oldLogin {
		if _, taken := r.users[user.Login]; taken {
			return errors.New("login already taken")
		}
		delete(r.users, oldLogin)
	}

	r.users[user.Login] = user
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

	return errors.New("user not found")
}

func (r *InMemoryUserRepo) List() ([]*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*domain.User, 0, len(r.users))
	for _, user := range r.users {
		result = append(result, user)
	}

	return result, nil
}
