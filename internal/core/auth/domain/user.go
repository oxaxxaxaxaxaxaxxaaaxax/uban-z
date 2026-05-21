package domain

import "errors"

const (
	RoleStudentB = "student_b"
	RoleStudentM = "student_m"
	RoleStudentA = "student_a"
	RoleTeacher  = "teacher"
	RoleAdmin    = "admin"
)

var (
	ErrInvalidRole       = errors.New("invalid role")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserNotFound      = errors.New("user not found")
	ErrLoginAlreadyTaken = errors.New("login already taken")
)

type User struct {
	ID       int
	Login    string
	Password string // позже захешируем
	Role     string
}

func IsValidRole(role string) bool {
	switch role {
	case RoleStudentB, RoleStudentM, RoleStudentA, RoleTeacher, RoleAdmin:
		return true
	default:
		return false
	}
}
