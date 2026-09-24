package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

const tokenLength = 32

func GenerateRefreshToken() (string, []byte, error) {
	raw := make([]byte, tokenLength)

	_, err := rand.Read(raw)
	if err != nil {
		return "", nil, fmt.Errorf("generate refresh token: %v\n", err)
	}

	// hash := sha256.Sum256(raw)
	// return base64.RawURLEncoding.EncodeToString(raw), hash[:], nil
	token := base64.RawURLEncoding.EncodeToString(raw)

	hash := sha256.Sum256([]byte(token))

	return token, hash[:], nil
}
