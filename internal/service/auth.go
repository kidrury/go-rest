package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kidrury/rest-pro/internal/auth"
	"github.com/kidrury/rest-pro/internal/domain"
)

type AuthUserRepository interface {
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
}

type SessionRepository interface {
	Create(ctx context.Context, session domain.AuthSession) error
	Rotate(ctx context.Context, oldRefreshHash []byte, replacement domain.AuthSession, now time.Time, idleTTL time.Duration) (domain.AuthSession, error)
	RevokeFamily(ctx context.Context, familyID string) error
	Logout(ctx context.Context, hashedRefreshToken []byte) (domain.AuthSession, error)
}

type AuthService struct {
	users           AuthUserRepository
	sessions        SessionRepository
	refreshTokenTTL time.Duration
	refreshIdleTTL  time.Duration
	tokenManager    *auth.TokenManager
	dummyHash       string
}

type LoginResult struct {
	AccessToken  string
	RefreshToken string
	RefreshUntil time.Time
}

func NewAuthService(r AuthUserRepository, s SessionRepository, tm *auth.TokenManager, refreshTTL time.Duration, idleTTL time.Duration) (*AuthService, error) {
	dummyPassword := make([]byte, 32)

	_, err := rand.Read(dummyPassword)
	if err != nil {
		return nil, fmt.Errorf("generate dummy password: %w", err)
	}

	dh, err := auth.HashPassword(string(dummyPassword))
	if err != nil {
		return nil, fmt.Errorf("hash dummy password: %w", err)
	}

	return &AuthService{
		users:           r,
		sessions:        s,
		refreshTokenTTL: refreshTTL,
		refreshIdleTTL:  idleTTL,
		tokenManager:    tm,
		dummyHash:       dh,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (LoginResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.users.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			_ = auth.VerifyPassword(password, s.dummyHash)
			return LoginResult{}, domain.Error{
				Code:    "UNAUTHORIZED",
				Message: "invalid username or password",
			}
		}
		return LoginResult{}, fmt.Errorf("find user for login: %w", err)
	}

	err = auth.VerifyPassword(password, user.PasswordHash)

	if err != nil {
		if errors.Is(err, auth.ErrPasswordsMismatch) {
			return LoginResult{}, domain.Error{
				Code:    "UNAUTHORIZED",
				Message: "invalid username or password",
			}
		}
		return LoginResult{}, fmt.Errorf("verify password: %w", err)
	}

	now := time.Now()

	refreshUntil := now.Add(s.refreshTokenTTL)

	refreshToken, refreshTokenHash, err := auth.GenerateRefreshToken()
	if err != nil {
		return LoginResult{}, fmt.Errorf("generate refresh token: %w\n", err)
	}

	session := domain.AuthSession{
		ID:               uuid.New().String(),
		FamilyID:         uuid.New().String(),
		UserID:           user.ID,
		RefreshTokenHash: refreshTokenHash,
		CreatedAt:        now,
		ExpiresAt:        refreshUntil,
	}

	err = s.sessions.Create(ctx, session)
	if err != nil {
		return LoginResult{}, fmt.Errorf("create auth session: %w\n", err)
	}

	accessToken, err := s.tokenManager.IssueAccessToken(user.ID)

	if err != nil {
		return LoginResult{}, fmt.Errorf("issue access tokrn: %w", err)
	}

	return LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		RefreshUntil: refreshUntil,
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, oldRefresh string) (LoginResult, error) {
	if oldRefresh == "" {
		return LoginResult{}, domain.Error{
			Code:    "UNAUTHORIZED",
			Message: "authentication required",
		}
	}

	oldHashed := sha256.Sum256([]byte(oldRefresh))

	newRefresh, newRefreshHashed, err := auth.GenerateRefreshToken()
	if err != nil {
		return LoginResult{}, fmt.Errorf("generate new refresh token: %w", err)
	}

	now := time.Now()

	replacement := domain.AuthSession{
		ID:               uuid.NewString(),
		RefreshTokenHash: newRefreshHashed,
		CreatedAt:        now,
	}

	oldSession, err := s.sessions.Rotate(ctx, oldHashed[:], replacement, now, s.refreshIdleTTL)

	if err != nil {
		if errors.Is(err, domain.ErrSessionReuse) {
			if err := s.sessions.RevokeFamily(ctx, oldSession.FamilyID); err != nil {
				return LoginResult{}, fmt.Errorf("revoke reused session: %w", err)
			}
			return LoginResult{}, domain.Error{
				Code:    "UNAUTHORIZED",
				Message: "authentication required",
			}
		}

		if errors.Is(err, domain.ErrSessionExpired) ||
			errors.Is(err, domain.ErrSessionNotFound) {
			return LoginResult{}, domain.Error{
				Code:    "UNAUTHORIZED",
				Message: "authentication required",
			}
		}
		return LoginResult{}, fmt.Errorf("rotate session: %w", err)
	}

	accessToken, err := s.tokenManager.IssueAccessToken(oldSession.UserID)
	if err != nil {
		return LoginResult{}, fmt.Errorf("issue replacement access token: %w", err)
	}

	return LoginResult{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
		RefreshUntil: oldSession.ExpiresAt,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshTokenString string) error {
	if refreshTokenString == "" {
		return domain.Error{
			Code:    "UNAUTHORIZED",
			Message: "authenitication required",
		}
	}
	hash := sha256.Sum256([]byte(refreshTokenString))

	_, err := s.sessions.Logout(ctx, hash[:])
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return nil
		}
		return fmt.Errorf("logout: %w", err)
	}

	return nil
}
