package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestGenerateRefreshToken(t *testing.T) {
	token, hash, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error: %v", err)
	}

	if token == "" {
		t.Fatal("GenerateRefreshToken() returned empty token")
	}

	if len(hash) != sha256.Size {
		t.Fatalf("hash length = %d, want %d", len(hash), sha256.Size)
	}

	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("refresh token is not valid Base64URL: %v", err)
	}

	if len(raw) != tokenLength {
		t.Fatalf("decoded token length = %d, want %d", len(raw), tokenLength)
	}

	expectedHash := sha256.Sum256([]byte(token))

	if string(hash) != string(expectedHash[:]) {
		t.Fatal("stored hashed refresh token does not match token")
	}

}
