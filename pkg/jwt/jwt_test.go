package jwt_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	jwtpkg "github.com/labib0x9/ffgif/pkg/jwt"
)

func TestJwt_CreateAndVerify_Success(t *testing.T) {
	secret := []byte("my-secret-key-12345")
	j := jwtpkg.NewJwt(secret)

	fullname := "Test User"
	userId := "123e4567-e89b-12d3-a456-426614174000"
	email := "test@example.com"
	role := "user"

	tokenStr, err := j.Create(fullname, userId, email, role)
	if err != nil {
		t.Fatalf("expected no error creating jwt, got: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("expected token string, got empty")
	}

	payload, err := j.Verify(tokenStr)
	if err != nil {
		t.Fatalf("expected successful verification, got: %v", err)
	}

	if payload.Subject != userId {
		t.Errorf("expected subject %s, got %s", userId, payload.Subject)
	}
	if payload.Email != email {
		t.Errorf("expected email %s, got %s", email, payload.Email)
	}
	if payload.Fullname != fullname {
		t.Errorf("expected fullname %s, got %s", fullname, payload.Fullname)
	}
	if payload.Role != role {
		t.Errorf("expected role %s, got %s", role, payload.Role)
	}
}

func TestJwt_Create_EmptyFields(t *testing.T) {
	secret := []byte("my-secret-key-12345")
	j := jwtpkg.NewJwt(secret)

	_, err := j.Create("", "id", "test@example.com", "user")
	if err == nil {
		t.Fatal("expected error on empty fullname, got nil")
	}

	_, err = j.Create("User", "id", "test@example.com", "")
	if err == nil {
		t.Fatal("expected error on empty role, got nil")
	}
}

func TestJwt_Verify_InvalidToken(t *testing.T) {
	secret := []byte("my-secret-key-12345")
	j := jwtpkg.NewJwt(secret)

	_, err := j.Verify("invalid.token.string")
	if err == nil {
		t.Fatal("expected error for malformed token, got nil")
	}
}

func TestJwt_Verify_WrongSecret(t *testing.T) {
	j1 := jwtpkg.NewJwt([]byte("secret-one"))
	j2 := jwtpkg.NewJwt([]byte("secret-two"))

	tokenStr, err := j1.Create("User", "id-1", "user@example.com", "user")
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	_, err = j2.Verify(tokenStr)
	if err == nil {
		t.Fatal("expected error verifying with wrong secret, got nil")
	}
}

func TestJwt_Verify_ExpiredToken(t *testing.T) {
	secret := []byte("my-secret-key-12345")
	j := jwtpkg.NewJwt(secret)

	expiredClaims := jwtpkg.Payload{
		Fullname: "Expired User",
		Email:    "expired@example.com",
		Role:     "user",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "id-expired",
			Issuer:    "ffgif",
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	tokenStr, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("failed to sign expired token: %v", err)
	}

	_, err = j.Verify(tokenStr)
	if err == nil {
		t.Fatal("expected verification error for expired token, got nil")
	}
}
