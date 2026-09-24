package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/kidrury/rest-pro/internal/auth"
	"github.com/kidrury/rest-pro/internal/domain"
)

type fakeAuthUserRepository struct {
	user          domain.User
	getByEmailErr error
	gotEmail      string
}

func (f *fakeAuthUserRepository) GetUserByEmail(
	ctx context.Context,
	email string,
) (domain.User, error) {
	f.gotEmail = email

	if f.getByEmailErr != nil {
		return domain.User{}, f.getByEmailErr
	}

	return f.user, nil
}

type fakeSessionRepository struct {
	createErr error

	createdSession *domain.AuthSession

	rotateErr         error
	rotateOldHash     []byte
	rotateReplacement domain.AuthSession
	rotateNow         time.Time
	rotateIdleTTL     time.Duration
	rotateSession     domain.AuthSession

	logoutErr     error
	logoutHash    []byte
	logoutSession domain.AuthSession

	revokeFamilyErr error
	revokedFamilyID string
}

func (f *fakeSessionRepository) Create(
	ctx context.Context,
	session domain.AuthSession,
) error {
	if f.createErr != nil {
		return f.createErr
	}

	copySession := session
	f.createdSession = &copySession

	return nil
}

func (f *fakeSessionRepository) Rotate(
	ctx context.Context,
	oldRefreshHash []byte,
	replacement domain.AuthSession,
	now time.Time,
	idleTTL time.Duration,
) (domain.AuthSession, error) {
	f.rotateOldHash = append([]byte(nil), oldRefreshHash...)
	f.rotateReplacement = replacement
	f.rotateNow = now
	f.rotateIdleTTL = idleTTL

	if f.rotateErr != nil {
		return f.rotateSession, f.rotateErr
	}

	return f.rotateSession, nil
}

func (f *fakeSessionRepository) RevokeFamily(
	ctx context.Context,
	familyID string,
) error {
	f.revokedFamilyID = familyID

	return f.revokeFamilyErr
}

func (f *fakeSessionRepository) Logout(
	ctx context.Context,
	hashedRefreshToken []byte,
) (domain.AuthSession, error) {
	f.logoutHash = append([]byte(nil), hashedRefreshToken...)

	if f.logoutErr != nil {
		return domain.AuthSession{}, f.logoutErr
	}

	return f.logoutSession, nil
}

func newTestAuthService(
	t *testing.T,
	users AuthUserRepository,
	sessions SessionRepository,
) *AuthService {
	t.Helper()

	tokenManager, err := auth.NewTokenManager(
		"01234567890123456789012345678901",
		"rest-pro",
		"rest-pro-api",
		15*time.Minute,
	)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	service, err := NewAuthService(
		users,
		sessions,
		tokenManager,
		24*time.Hour,
		30*time.Minute,
	)
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}

	return service
}

func TestAuthServiceLogin(t *testing.T) {
	password := "correct-password"

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	users := &fakeAuthUserRepository{
		user: domain.User{
			ID:           "user-123",
			Email:        "user@example.com",
			PasswordHash: passwordHash,
		},
	}

	sessions := &fakeSessionRepository{}

	service := newTestAuthService(t, users, sessions)

	before := time.Now()

	result, err := service.Login(
		context.Background(),
		"  USER@Example.com  ",
		password,
	)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	after := time.Now()

	if users.gotEmail != "user@example.com" {
		t.Fatalf(
			"repository received email %q, want %q",
			users.gotEmail,
			"user@example.com",
		)
	}

	if result.AccessToken == "" {
		t.Fatal("Login() returned empty access token")
	}

	if result.RefreshToken == "" {
		t.Fatal("Login() returned empty refresh token")
	}

	if sessions.createdSession == nil {
		t.Fatal("Login() did not create a session")
	}

	session := sessions.createdSession

	if session.UserID != "user-123" {
		t.Fatalf(
			"session.UserID = %q, want %q",
			session.UserID,
			"user-123",
		)
	}

	if session.ID == "" {
		t.Fatal("session.ID is empty")
	}

	if session.FamilyID == "" {
		t.Fatal("session.FamilyID is empty")
	}

	if len(session.RefreshTokenHash) == 0 {
		t.Fatal("session.RefreshTokenHash is empty")
	}

	if !session.ExpiresAt.After(session.CreatedAt) {
		t.Fatal("session expiration is not after creation")
	}

	if result.RefreshUntil.Before(before.Add(23*time.Hour)) ||
		result.RefreshUntil.After(after.Add(25*time.Hour)) {
		t.Fatalf(
			"RefreshUntil = %v, outside expected range",
			result.RefreshUntil,
		)
	}
}

func TestAuthServiceLoginUnknownUser(t *testing.T) {
	users := &fakeAuthUserRepository{
		getByEmailErr: domain.ErrUserNotFound,
	}

	sessions := &fakeSessionRepository{}

	service := newTestAuthService(t, users, sessions)

	_, err := service.Login(
		context.Background(),
		"user@example.com",
		"wrong-password",
	)
	if err == nil {
		t.Fatal("Login() returned nil error")
	}

	var domainErr domain.Error
	if !errors.As(err, &domainErr) {
		t.Fatalf("error type = %T, want domain.Error", err)
	}

	if domainErr.Code != "UNAUTHORIZED" {
		t.Fatalf(
			"error code = %q, want %q",
			domainErr.Code,
			"UNAUTHORIZED",
		)
	}

	if domainErr.Message != "invalid username or password" {
		t.Fatalf(
			"error message = %q, want %q",
			domainErr.Message,
			"invalid username or password",
		)
	}

	if sessions.createdSession != nil {
		t.Fatal("session was created for unknown user")
	}
}

func TestAuthServiceLoginWrongPassword(t *testing.T) {
	passwordHash, err := auth.HashPassword("correct-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	users := &fakeAuthUserRepository{
		user: domain.User{
			ID:           "user-123",
			Email:        "user@example.com",
			PasswordHash: passwordHash,
		},
	}

	sessions := &fakeSessionRepository{}

	service := newTestAuthService(t, users, sessions)

	_, err = service.Login(
		context.Background(),
		"user@example.com",
		"wrong-password",
	)
	if err == nil {
		t.Fatal("Login() returned nil error")
	}

	var domainErr domain.Error
	if !errors.As(err, &domainErr) {
		t.Fatalf("error type = %T, want domain.Error", err)
	}

	if domainErr.Code != "UNAUTHORIZED" {
		t.Fatalf(
			"error code = %q, want %q",
			domainErr.Code,
			"UNAUTHORIZED",
		)
	}

	if sessions.createdSession != nil {
		t.Fatal("session was created after wrong password")
	}
}

func TestAuthServiceLoginSessionCreationFailure(t *testing.T) {
	passwordHash, err := auth.HashPassword("correct-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	users := &fakeAuthUserRepository{
		user: domain.User{
			ID:           "user-123",
			Email:        "user@example.com",
			PasswordHash: passwordHash,
		},
	}

	sessionErr := errors.New("database unavailable")

	sessions := &fakeSessionRepository{
		createErr: sessionErr,
	}

	service := newTestAuthService(t, users, sessions)

	_, err = service.Login(
		context.Background(),
		"user@example.com",
		"correct-password",
	)
	if err == nil {
		t.Fatal("Login() returned nil error")
	}

	if !errors.Is(err, sessionErr) {
		t.Fatalf(
			"Login() error = %v, does not wrap %v",
			err,
			sessionErr,
		)
	}
}

func TestAuthServiceRefresh(t *testing.T) {
	oldRefresh := "old-refresh-token"

	oldExpiresAt := time.Now().Add(12 * time.Hour)

	sessions := &fakeSessionRepository{
		rotateSession: domain.AuthSession{
			ID:        "old-session",
			FamilyID:  "family-123",
			UserID:    "user-123",
			ExpiresAt: oldExpiresAt,
		},
	}

	users := &fakeAuthUserRepository{}
	service := newTestAuthService(t, users, sessions)

	result, err := service.Refresh(
		context.Background(),
		oldRefresh,
	)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	if result.AccessToken == "" {
		t.Fatal("Refresh() returned empty access token")
	}

	if result.RefreshToken == "" {
		t.Fatal("Refresh() returned empty refresh token")
	}

	if result.RefreshToken == oldRefresh {
		t.Fatal("Refresh() returned the old refresh token")
	}

	if result.RefreshUntil != oldExpiresAt {
		t.Fatalf(
			"RefreshUntil = %v, want %v",
			result.RefreshUntil,
			oldExpiresAt,
		)
	}

	if len(sessions.rotateOldHash) == 0 {
		t.Fatal("Refresh() did not pass a refresh-token hash to Rotate()")
	}

	if sessions.rotateReplacement.ID == "" {
		t.Fatal("replacement session ID is empty")
	}

	if sessions.rotateReplacement.RefreshTokenHash == nil {
		t.Fatal("replacement refresh-token hash is nil")
	}

	if sessions.rotateReplacement.UserID != "" {
		t.Fatalf(
			"replacement.UserID = %q, want empty before repository enrichment",
			sessions.rotateReplacement.UserID,
		)
	}
}

func TestAuthServiceRefreshEmptyToken(t *testing.T) {
	service := newTestAuthService(
		t,
		&fakeAuthUserRepository{},
		&fakeSessionRepository{},
	)

	_, err := service.Refresh(context.Background(), "")
	if err == nil {
		t.Fatal("Refresh() returned nil error")
	}

	var domainErr domain.Error
	if !errors.As(err, &domainErr) {
		t.Fatalf("error type = %T, want domain.Error", err)
	}

	if domainErr.Code != "UNAUTHORIZED" {
		t.Fatalf(
			"error code = %q, want %q",
			domainErr.Code,
			"UNAUTHORIZED",
		)
	}
}

func TestAuthServiceRefreshSessionNotFound(t *testing.T) {
	sessions := &fakeSessionRepository{
		rotateErr: domain.ErrSessionNotFound,
	}

	service := newTestAuthService(
		t,
		&fakeAuthUserRepository{},
		sessions,
	)

	_, err := service.Refresh(
		context.Background(),
		"invalid-refresh-token",
	)
	if err == nil {
		t.Fatal("Refresh() returned nil error")
	}

	var domainErr domain.Error
	if !errors.As(err, &domainErr) {
		t.Fatalf("error type = %T, want domain.Error", err)
	}

	if domainErr.Code != "UNAUTHORIZED" {
		t.Fatalf(
			"error code = %q, want %q",
			domainErr.Code,
			"UNAUTHORIZED",
		)
	}
}

func TestAuthServiceRefreshExpiredSession(t *testing.T) {
	sessions := &fakeSessionRepository{
		rotateErr: domain.ErrSessionExpired,
	}

	service := newTestAuthService(
		t,
		&fakeAuthUserRepository{},
		sessions,
	)

	_, err := service.Refresh(
		context.Background(),
		"expired-refresh-token",
	)
	if err == nil {
		t.Fatal("Refresh() returned nil error")
	}

	var domainErr domain.Error
	if !errors.As(err, &domainErr) {
		t.Fatalf("error type = %T, want domain.Error", err)
	}

	if domainErr.Code != "UNAUTHORIZED" {
		t.Fatalf(
			"error code = %q, want %q",
			domainErr.Code,
			"UNAUTHORIZED",
		)
	}
}

func TestAuthServiceRefreshReuseRevokesFamily(t *testing.T) {
	sessions := &fakeSessionRepository{
		rotateErr: domain.ErrSessionReuse,
		rotateSession: domain.AuthSession{
			FamilyID: "family-123",
			UserID:   "user-123",
		},
	}

	service := newTestAuthService(
		t,
		&fakeAuthUserRepository{},
		sessions,
	)

	_, err := service.Refresh(
		context.Background(),
		"reused-refresh-token",
	)
	if err == nil {
		t.Fatal("Refresh() returned nil error")
	}

	if sessions.revokedFamilyID != "family-123" {
		t.Fatalf(
			"revoked family ID = %q, want %q",
			sessions.revokedFamilyID,
			"family-123",
		)
	}

	var domainErr domain.Error
	if !errors.As(err, &domainErr) {
		t.Fatalf("error type = %T, want domain.Error", err)
	}

	if domainErr.Code != "UNAUTHORIZED" {
		t.Fatalf(
			"error code = %q, want %q",
			domainErr.Code,
			"UNAUTHORIZED",
		)
	}
}

func TestAuthServiceRefreshRotationFailure(t *testing.T) {
	rotationErr := errors.New("database unavailable")

	sessions := &fakeSessionRepository{
		rotateErr: rotationErr,
	}

	service := newTestAuthService(
		t,
		&fakeAuthUserRepository{},
		sessions,
	)

	_, err := service.Refresh(
		context.Background(),
		"refresh-token",
	)
	if err == nil {
		t.Fatal("Refresh() returned nil error")
	}

	if !errors.Is(err, rotationErr) {
		t.Fatalf(
			"Refresh() error = %v, does not wrap %v",
			err,
			rotationErr,
		)
	}
}

func TestAuthServiceLogoutEmptyToken(t *testing.T) {
	service := newTestAuthService(
		t,
		&fakeAuthUserRepository{},
		&fakeSessionRepository{},
	)

	err := service.Logout(context.Background(), "")
	if err == nil {
		t.Fatal("Logout() returned nil error")
	}

	var domainErr domain.Error
	if !errors.As(err, &domainErr) {
		t.Fatalf("error type = %T, want domain.Error", err)
	}

	if domainErr.Code != "UNAUTHORIZED" {
		t.Fatalf(
			"error code = %q, want %q",
			domainErr.Code,
			"UNAUTHORIZED",
		)
	}
}

func TestAuthServiceLogout(t *testing.T) {
	const refreshToken = "refresh-token-to-logout"

	sessions := &fakeSessionRepository{}

	service := newTestAuthService(
		t,
		&fakeAuthUserRepository{},
		sessions,
	)

	err := service.Logout(
		context.Background(),
		refreshToken,
	)
	if err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	expectedHash := sha256.Sum256([]byte(refreshToken))

	if string(sessions.logoutHash) != string(expectedHash[:]) {
		t.Fatal("Logout() did not hash the refresh token correctly")
	}
}

func TestAuthServiceLogoutNotFoundIsIdempotent(t *testing.T) {
	sessions := &fakeSessionRepository{
		logoutErr: domain.ErrSessionNotFound,
	}

	service := newTestAuthService(
		t,
		&fakeAuthUserRepository{},
		sessions,
	)

	err := service.Logout(
		context.Background(),
		"refresh-token",
	)
	if err != nil {
		t.Fatalf(
			"Logout() error = %v, want nil",
			err,
		)
	}
}

func TestAuthServiceLogoutRepositoryFailure(t *testing.T) {
	logoutErr := errors.New("database unavailable")

	sessions := &fakeSessionRepository{
		logoutErr: logoutErr,
	}

	service := newTestAuthService(
		t,
		&fakeAuthUserRepository{},
		sessions,
	)

	err := service.Logout(
		context.Background(),
		"refresh-token",
	)
	if err == nil {
		t.Fatal("Logout() returned nil error")
	}

	if !errors.Is(err, logoutErr) {
		t.Fatalf(
			"Logout() error = %v, does not wrap %v",
			err,
			logoutErr,
		)
	}
}
