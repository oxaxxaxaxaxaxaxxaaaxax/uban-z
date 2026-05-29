package jwt_test

import (
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"

	authjwt "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/auth/jwt"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/auth/domain"
)

func TestJWTManagerGenerateUsesBookingCompatibleSubject(t *testing.T) {
	t.Parallel()

	manager := authjwt.NewJWTManager("test-secret")
	token, err := manager.Generate(&domain.User{
		ID:    42,
		Login: "alice",
		Role:  domain.RoleStudentB,
	})
	if err != nil {
		t.Fatalf("Generate err = %v", err)
	}

	claims, err := manager.Verify(token)
	if err != nil {
		t.Fatalf("Verify err = %v", err)
	}
	if claims.UserID != 42 || claims.Login != "alice" || claims.Role != domain.RoleStudentB {
		t.Fatalf("claims = %#v, want generated user identity", claims)
	}

	parsed, _, err := gojwt.NewParser().ParseUnverified(token, gojwt.MapClaims{})
	if err != nil {
		t.Fatalf("parse unverified: %v", err)
	}
	rawClaims, ok := parsed.Claims.(gojwt.MapClaims)
	if !ok {
		t.Fatalf("claims type = %T, want jwt.MapClaims", parsed.Claims)
	}
	if rawClaims["sub"] != "42" {
		t.Fatalf("sub = %#v, want %q", rawClaims["sub"], "42")
	}
	if rawClaims["user_id"] == nil {
		t.Fatal("user_id claim is missing")
	}
}

func TestJWTManagerVerifyAcceptsLegacyUserIDClaim(t *testing.T) {
	t.Parallel()

	secret := []byte("test-secret")
	token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, gojwt.MapClaims{
		"user_id": 7,
		"login":   "teacher",
		"role":    domain.RoleTeacher,
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	claims, err := authjwt.NewJWTManager(string(secret)).Verify(signed)
	if err != nil {
		t.Fatalf("Verify err = %v", err)
	}
	if claims.UserID != 7 || claims.Login != "teacher" || claims.Role != domain.RoleTeacher {
		t.Fatalf("claims = %#v, want legacy identity", claims)
	}
}
