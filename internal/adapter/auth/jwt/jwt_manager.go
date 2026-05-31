package jwt

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/domain"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/ports"
)

type JWTManager struct {
	secret []byte
}

func NewJWTManager(secret string) *JWTManager {
	return &JWTManager{
		secret: []byte(secret),
	}
}

func (j *JWTManager) Generate(user *domain.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":     strconv.Itoa(user.ID),
		"user_id": user.ID,
		"login":   user.Login,
		"role":    user.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}

func (j *JWTManager) Verify(tokenStr string) (*ports.Claims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		// Проверка метода подписи (важно для безопасности!)
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return j.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Извлекаем claims из токена
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	userID, err := userIDFromClaims(claims)
	if err != nil {
		return nil, err
	}
	login, ok := claims["login"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid login claim")
	}
	role, ok := claims["role"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid role claim")
	}

	return &ports.Claims{
		UserID: userID,
		Login:  login,
		Role:   role,
	}, nil
}

func userIDFromClaims(claims jwt.MapClaims) (int, error) {
	if sub, ok := claims["sub"].(string); ok && sub != "" {
		userID, err := strconv.Atoi(sub)
		if err != nil {
			return 0, fmt.Errorf("invalid sub claim")
		}
		return userID, nil
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid user_id claim")
	}
	return int(userID), nil
}
