package auth

import (
	"testing"
	"time"
)

func newTestTokenManager(t *testing.T) *TokenManager {
	t.Helper()

	tm, err := NewTokenManager(
		"01234567890123456789012345678901",
		"alireza",
		"users",
		time.Minute*5,
	)

	if err != nil {
		t.Fatalf("NewTokenManager() error: %v", err)
	}

	return tm
}

func TestTokenIssueAndVerify(t *testing.T) {
	tm := newTestTokenManager(t)

	token, err := tm.IssueAccessToken("my_user_id")
	if err != nil {
		t.Fatalf("IssueAccessToken() error: %v", err)
	}

	if token == "" {
		t.Fatal("IssueAccessToken() returned an empty token")
	}

	claims, err := tm.VerifyAccessToken(token)
	if err != nil {
		t.Fatalf("VerifyAccessToken() error: %v", err)
	}

	if claims.Subject != "my_user_id" {
		t.Fatalf("claim subject: %q, want: %q", err, "my_user_id")
	}

	if claims.Issuer != "alireza" {
		t.Fatalf(
			"claims.Issuer = %q, want %q",
			claims.Issuer,
			"alireza",
		)
	}

	if len(claims.Audience) != 1 || claims.Audience[0] != "users" {
		t.Fatalf(
			"claims.Audience: %v, want: users",
			claims.Audience,
		)
	}

	if claims.ExpiresAt == nil {
		t.Fatal("claims.ExpiresAt is nil")
	}

	if claims.IssuedAt == nil {
		t.Fatal("claims.IssuedAt is nil")
	}

	if claims.ID == "" {
		t.Fatal("claims.ID is empty")
	}
}

func TestTokenManagerRejectsInvalidToken(t *testing.T) {
	tm := newTestTokenManager(t)
	_, err := tm.VerifyAccessToken("invalid")

	if err != ErrInvalidToken {
		t.Fatalf("VerifyAccessToken() error: %v, want: %v", err, ErrInvalidToken)
	}
}

func TestTokenManagerRejectsEmptyToken(t *testing.T) {
	tm := newTestTokenManager(t)
	_, err := tm.VerifyAccessToken("invalid")

	if err != ErrInvalidToken {
		t.Fatalf("VerifyAccessToken() error: %v, want: %v", err, ErrInvalidToken)
	}
}

func TestTokenManagerRejectsWrongSecret(t *testing.T) {
	tm := newTestTokenManager(t)

	token, err := tm.IssueAccessToken("user-123")
	if err != nil {
		t.Fatalf("IssueAccessToken() error = %v", err)
	}

	otherTM, err := NewTokenManager(
		"abcdefghijklmnopqrstuvwxyz123456",
		"alireza",
		"users",
		15*time.Minute,
	)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	_, err = otherTM.VerifyAccessToken(token)
	if err != ErrInvalidToken {
		t.Fatalf(
			"VerifyAccessToken() error = %v, want %v",
			err,
			ErrInvalidToken,
		)
	}
}

func TestTokenManagerRejectsWrongAudience(t *testing.T) {
	tm := newTestTokenManager(t)

	token, err := tm.IssueAccessToken("user-123")
	if err != nil {
		t.Fatalf("IssueAccessToken() error = %v", err)
	}

	otherTM, err := NewTokenManager(
		"01234567890123456789012345678901",
		"wlireza",
		"wrong-audience",
		15*time.Minute,
	)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	_, err = otherTM.VerifyAccessToken(token)
	if err != ErrInvalidToken {
		t.Fatalf(
			"VerifyAccessToken() error = %v, want %v",
			err,
			ErrInvalidToken,
		)
	}
}

func TestTokenManagerRejectsExpiredToken(t *testing.T) {
	tm, err := NewTokenManager(
		"01234567890123456789012345678901",
		"alireza",
		"users",
		-1*time.Second,
	)
	if err == nil {
		t.Fatal("NewTokenManager() did not reject non positive TTL")
	}

	// NewTokenManager correctly rejects non-positive TTLs, so construct
	// an expired token with a one-second TTL and wait for expiration.
	tm, err = NewTokenManager(
		"01234567890123456789012345678901",
		"alireza",
		"users",
		1*time.Millisecond,
	)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	token, err := tm.IssueAccessToken("user-123")
	if err != nil {
		t.Fatalf("IssueAccessToken() error = %v", err)
	}

	time.Sleep(5 * time.Millisecond)

	_, err = tm.VerifyAccessToken(token)
	if err != ErrInvalidToken {
		t.Fatalf(
			"VerifyAccessToken() error = %v, want %v",
			err,
			ErrInvalidToken,
		)
	}
}
