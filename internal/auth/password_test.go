package auth

import (
	"errors"
	"testing"
)

func TestHashPasswordAndVerify(t *testing.T) {

	password := "here you go"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error: %v", err)
	}

	err = VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword() returned error for correct password: %v", err)
	}

	err = VerifyPassword("this is not the correct password", hash)
	if !errors.Is(err, ErrPasswordsMismatch) {
		t.Fatalf("VerifyPassword() returned: %v, want: %v", err, ErrPasswordsMismatch)
	}

}

func TestHashPasswordGeneratesDifferentHashes(t *testing.T) {
	password := "same password"

	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash1 == hash2 {
		t.Fatal("two hashes of the same password are identical")
	}
}
