package domain

type User struct {
	ID       int
	Login    string
	Password string // позже захешируем
	Role     string
}
