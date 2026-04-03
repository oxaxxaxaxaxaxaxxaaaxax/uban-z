package authutils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("super-secret") // вынести в env потом

func GenerateJWT(userID int, login string, role string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"login":   login,
		"role":    role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}
