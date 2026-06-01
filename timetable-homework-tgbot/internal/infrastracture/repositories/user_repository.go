package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"timetable-homework-tgbot/internal/infrastracture/database"
)

const (
	RoleGuest   = ""
	RoleStudent = "student"
	RoleTeacher = "teacher"
)

type UsersRepository interface {
	GetRole(ctx context.Context, userID int64) (string, error)
	GetGroup(ctx context.Context, userID int64) (string, error)
	LoginStudent(ctx context.Context, userID int64, login, password string) error
	LoginTeacher(ctx context.Context, userID int64, login, password string) error
	Leave(ctx context.Context, userID int64) error
}

type UserRepo struct {
	db *database.DB
}

func NewUserRepo(db *database.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) GetRole(ctx context.Context, userID int64) (string, error) {
	var exists int
	err := r.db.GetSql().QueryRowContext(ctx, `SELECT 1 FROM students WHERE user_id = $1`, userID).Scan(&exists)
	if err == nil {
		return RoleStudent, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("GetRole student query: %w", err)
	}

	err = r.db.GetSql().QueryRowContext(ctx, `SELECT 1 FROM teachers WHERE user_id = $1`, userID).Scan(&exists)
	if err == nil {
		return RoleTeacher, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("GetRole teacher query: %w", err)
	}

	return RoleGuest, nil
}

func (r *UserRepo) GetGroup(ctx context.Context, userID int64) (string, error) {
	var group string

	err := r.db.GetSql().QueryRowContext(ctx, `
SELECT g.group_name
FROM students s
JOIN groups g ON g.id = s.group_id
WHERE s.user_id = $1
`, userID).Scan(&group)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("GetGroup query: %w", err)
	}

	return group, nil
}

func (r *UserRepo) LoginStudent(ctx context.Context, userID int64, login, password string) error {
	tx, err := r.db.GetSql().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("LoginStudent begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var studentID int64
	if err := tx.QueryRowContext(ctx, `
SELECT id
FROM students
WHERE login = $1
  AND password = $2
`, strings.TrimSpace(login), password).Scan(&studentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("invalid student credentials")
		}
		return fmt.Errorf("LoginStudent account: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO users (tg_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID); err != nil {
		return fmt.Errorf("LoginStudent user: %w", err)
	}
	if err := unbindTeacherRows(ctx, tx, userID); err != nil {
		return fmt.Errorf("LoginStudent unbind teacher: %w", err)
	}
	if err := unbindStudentRows(ctx, tx, userID); err != nil {
		return fmt.Errorf("LoginStudent unbind student: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE students SET user_id = $1 WHERE id = $2`, userID, studentID); err != nil {
		return fmt.Errorf("LoginStudent bind: %w", err)
	}

	return tx.Commit()
}

func (r *UserRepo) LoginTeacher(ctx context.Context, userID int64, login, password string) error {
	tx, err := r.db.GetSql().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("LoginTeacher begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var teacherID int64
	if err := tx.QueryRowContext(ctx, `
SELECT id
FROM teachers
WHERE login = $1
  AND password = $2
`, strings.TrimSpace(login), password).Scan(&teacherID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("invalid teacher credentials")
		}
		return fmt.Errorf("LoginTeacher account: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO users (tg_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID); err != nil {
		return fmt.Errorf("LoginTeacher user: %w", err)
	}
	if err := unbindTeacherRows(ctx, tx, userID); err != nil {
		return fmt.Errorf("LoginTeacher unbind previous teacher: %w", err)
	}
	if err := unbindStudentRows(ctx, tx, userID); err != nil {
		return fmt.Errorf("LoginTeacher unbind student: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE teachers SET user_id = $1 WHERE id = $2`, userID, teacherID); err != nil {
		return fmt.Errorf("LoginTeacher bind: %w", err)
	}

	return tx.Commit()
}

func (r *UserRepo) Leave(ctx context.Context, userID int64) error {
	tx, err := r.db.GetSql().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("Leave begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := unbindStudentRows(ctx, tx, userID); err != nil {
		return fmt.Errorf("Leave unbind student: %w", err)
	}
	if err := unbindTeacherRows(ctx, tx, userID); err != nil {
		return fmt.Errorf("Leave unbind teacher: %w", err)
	}

	return tx.Commit()
}

func unbindStudentRows(ctx context.Context, tx *sql.Tx, userID int64) error {
	if _, err := tx.ExecContext(ctx, `
INSERT INTO users (tg_id)
SELECT -300000000000000 - id
FROM students
WHERE user_id = $1
ON CONFLICT DO NOTHING
`, userID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
UPDATE students
SET user_id = -300000000000000 - id
WHERE user_id = $1
`, userID)
	return err
}

func unbindTeacherRows(ctx context.Context, tx *sql.Tx, userID int64) error {
	if _, err := tx.ExecContext(ctx, `
INSERT INTO users (tg_id)
SELECT -200000000000000 - id
FROM teachers
WHERE user_id = $1
ON CONFLICT DO NOTHING
`, userID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
UPDATE teachers
SET user_id = -200000000000000 - id
WHERE user_id = $1
`, userID)
	return err
}
