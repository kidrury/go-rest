package domain

import (
	"errors"
	"time"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
	ErrSessionReuse    = errors.New("refresh token reuse detected")
)

type AuthSession struct {
	ID               string
	FamilyID         string
	UserID           string
	RefreshTokenHash []byte
	CreatedAt        time.Time
	ExpiresAt        time.Time
	LastUsedAt       *time.Time
	RevokedAt        *time.Time
	ReplacedBy       *string
}
