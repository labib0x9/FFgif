package token

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/google/uuid"
)

func GetTokenHash(token string) string {
	shaHash := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(shaHash[:])
	return hash
}

func GenerateToken() (string, string) {
	token := uuid.NewString()
	hash := GetTokenHash(token)
	return token, hash
}
