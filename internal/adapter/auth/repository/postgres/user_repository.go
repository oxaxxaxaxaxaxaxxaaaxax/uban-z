package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/domain"
)

const (
	queryTimeout          = 5 * time.Second
	pgCodeUniqueViolation = "23505"
	pgCodeInvalidText     = "22P02"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(user *domain.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (login, password_hash, role, full_name)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, user.Login, user.Password, user.Role, user.FullName).Scan(&user.ID)
	if err != nil {
		return translateWriteError(err, domain.ErrUserAlreadyExists)
	}

	return nil
}

func (r *UserRepository) GetByLogin(login string) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	user, err := scanUser(r.pool.QueryRow(ctx, `
		SELECT id, login, password_hash, role, full_name
		FROM users
		WHERE login = $1
	`, login))
	if err != nil {
		return nil, translateReadError(err, "get user by login")
	}

	return user, nil
}

func (r *UserRepository) GetByID(id int) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	user, err := scanUser(r.pool.QueryRow(ctx, `
		SELECT id, login, password_hash, role, full_name
		FROM users
		WHERE id = $1
	`, id))
	if err != nil {
		return nil, translateReadError(err, "get user by id")
	}

	return user, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (*domain.User, error) {
	var user domain.User
	if err := row.Scan(&user.ID, &user.Login, &user.Password, &user.Role, &user.FullName); err != nil {
		return nil, err
	}
	return &user, nil
}

func translateReadError(err error, operation string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrUserNotFound
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func translateWriteError(err error, duplicateErr error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgCodeUniqueViolation:
			return duplicateErr
		case pgCodeInvalidText:
			return domain.ErrInvalidRole
		}
	}
	return fmt.Errorf("write user: %w", err)
}
