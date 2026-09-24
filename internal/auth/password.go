package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	PasswordAlg                = "argon2id"
	PasswordVersion            = 19
	PasswordMemory      int32  = 64 * 1024
	PasswordIterations  uint32 = 3
	PasswordParallelism uint8  = 4
	PasswordSaltLength         = 16
	PasswordKeyLength          = 32
)

var (
	ErrInvalidHash       = errors.New("invalid password hash")
	ErrPasswordsMismatch = errors.New("password does not match")
)

type passwordHashParams struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	salt        []byte
	hash        []byte
}

func HashPassword(password string) (string, error) {
	salt := make([]byte, PasswordSaltLength)
	_, err := rand.Read(salt)
	if err != nil {
		return "", fmt.Errorf("generate password salt: %w\n", err)
	}

	hashed := argon2.IDKey([]byte(password), salt, PasswordIterations, uint32(PasswordMemory), PasswordParallelism, PasswordKeyLength)

	encoded := base64.RawStdEncoding

	return fmt.Sprintf(
		"$%s$v=%d$m=%d,t=%d,p=%d$%s$%s",
		PasswordAlg,
		PasswordVersion,
		PasswordMemory,
		PasswordIterations,
		PasswordParallelism,
		encoded.EncodeToString(salt),
		encoded.EncodeToString(hashed),
	), nil
}

func VerifyPassword(password, encodedHash string) error {
	passwordParams, err := parsePasswordHash(encodedHash)
	if err != nil {
		return ErrInvalidHash
	}

	hashed := argon2.IDKey(
		[]byte(password),
		passwordParams.salt,
		passwordParams.iterations,
		passwordParams.memory,
		passwordParams.parallelism,
		uint32(len(passwordParams.hash)),
	)

	if subtle.ConstantTimeCompare(hashed, passwordParams.hash) != 1 {
		return ErrPasswordsMismatch
	}
	return nil
}

func parsePasswordHash(encodedHash string) (passwordHashParams, error) {
	parts := strings.Split(encodedHash, "$")

	if len(parts) != 6 || parts[0] != "" {
		return passwordHashParams{}, ErrInvalidHash
	}

	if parts[1] != PasswordAlg {
		return passwordHashParams{}, ErrInvalidHash
	}

	v, err := parseUintParameter(parts[2], "v")
	if err != nil || v != PasswordVersion {
		return passwordHashParams{}, ErrInvalidHash
	}

	args := strings.Split(parts[3], ",")
	if len(args) != 3 {
		return passwordHashParams{}, ErrInvalidHash

	}

	m, err := parseUintParameter(args[0], "m")
	if err != nil || m == 0 {
		return passwordHashParams{}, ErrInvalidHash
	}

	t, err := parseUintParameter(args[1], "t")
	if err != nil || t == 0 {
		return passwordHashParams{}, ErrInvalidHash
	}

	p, err := parseUintParameter(args[2], "p")
	if err != nil || p == 0 {
		return passwordHashParams{}, ErrInvalidHash
	}

	if m > uint64(^uint32(0)) || t > uint64(^uint32(0)) || p > uint64(^uint8(0)) {
		return passwordHashParams{}, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 8 {
		return passwordHashParams{}, ErrInvalidHash
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(hash) < 16 {
		return passwordHashParams{}, ErrInvalidHash
	}

	var hashParams passwordHashParams = passwordHashParams{
		memory:      uint32(m),
		iterations:  uint32(t),
		parallelism: uint8(p),
		salt:        salt,
		hash:        hash,
	}

	return hashParams, nil
}

func parseUintParameter(value, name string) (uint64, error) {
	prefix := name + "="

	if !strings.HasPrefix(value, prefix) {
		return 0, ErrInvalidHash
	}

	val := strings.TrimPrefix(value, prefix)

	return strconv.ParseUint(val, 10, 64)
}
