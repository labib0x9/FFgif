package password_test

import (
	"testing"

	"github.com/labib0x9/ffgif/pkg/password"
)

func TestHasher_GenerateAndCompare_Success(t *testing.T) {
	pepper := "secret-pepper-string"
	cost := 10
	h := password.NewHasher(pepper, cost)

	rawPass := "MySecurePassword123!"

	hash, err := h.GenerateHash(rawPass)
	if err != nil {
		t.Fatalf("expected hash generation to succeed, got: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash string")
	}

	if !h.CompareHashAndPassword(hash, rawPass) {
		t.Errorf("expected CompareHashAndPassword to return true for matching password")
	}

	if h.CompareHashAndPassword(hash, "WrongPassword") {
		t.Errorf("expected CompareHashAndPassword to return false for wrong password")
	}
}

func TestHasher_PepperSensitivity(t *testing.T) {
	h1 := password.NewHasher("pepper-one", 10)
	h2 := password.NewHasher("pepper-two", 10)

	rawPass := "MyPassword"
	hash, err := h1.GenerateHash(rawPass)
	if err != nil {
		t.Fatalf("failed to generate hash: %v", err)
	}

	if h2.CompareHashAndPassword(hash, rawPass) {
		t.Errorf("expected hash comparison to fail when verified with different pepper")
	}
}
