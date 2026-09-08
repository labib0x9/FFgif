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

func GenerateToken() (token string, tokenHash string) {
	token = uuid.NewString()
	tokenHash = GetTokenHash(token)
	return
}
