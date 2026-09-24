package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/kidrury/rest-pro/internal/auth"
	"github.com/kidrury/rest-pro/internal/authz"
	"github.com/kidrury/rest-pro/internal/domain"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user domain.User) error
	GetUserByID(ctx context.Context, userID string) (domain.User, error)
}

type UserService struct {
	user    UserRepository
	session SessionRepository
}

func NewUserService(userRepo UserRepository, sessionRepo SessionRepository) *UserService {
	return &UserService{
		user:    userRepo,
		session: sessionRepo,
	}
}

type CreateResult struct {
	AccessToken  string
	RefreshToken string
	RefreshUntil time.Time
}

func (s *UserService) CreateUser(ctx context.Context, email, password string) error {
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}

	auth.GenerateRefreshToken()

	newUser := domain.User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: passwordHash,
		Role:         "user",
	}

	err = s.user.CreateUser(ctx, newUser)
	if err != nil {
		return err
	}
	return nil
}

func (s *UserService) GetUserByID(ctx context.Context, actor auth.Identity, targetUserID string) (domain.User, error) {
	//first we get the actor user because we need their roles
	actorUser, err := s.user.GetUserByID(ctx, actor.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.User{}, domain.Error{
				Code:    "UNAUTHORIZED",
				Message: "authentication required",
			}
		}
		return domain.User{}, err
	}

	//now let's say they are who they wanna see. do they have the rights to see themselves?
	if actorUser.ID == targetUserID {
		if !authz.HasPermission(authz.Role(actorUser.Role), authz.PermissionReadSelf) {
			return domain.User{}, domain.Error{
				Code:    "FORBIDDEN",
				Message: "you are not allowed to access this user",
			}
		}
		return actorUser, nil
	}

	//they are looking for someone else. but can they?
	if !authz.HasPermission(authz.Role(actorUser.Role), authz.PermissionReadAny) {
		return domain.User{}, domain.Error{
			Code:    "FORBIDDEN",
			Message: "you are not allowed to access this user",
		}

	}

	targetUser, err := s.user.GetUserByID(ctx, targetUserID)
	if err != nil {
		return domain.User{}, err
	}
	return targetUser, nil
}

func (s *UserService) RequirePermission(
	ctx context.Context,
	actor auth.Identity,
	permission authz.Permission,
) error {
	actorUser, err := s.user.GetUserByID(ctx, actor.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.Error{
				Code:    "UNAUTHORIZED",
				Message: "authentication required",
			}
		}
		return err
	}

	if !authz.HasPermission(authz.Role(actorUser.Role), permission) {
		return domain.Error{
			Code:    "FORBIDDEN",
			Message: "you are not allowed to perform this operation",
		}
	}

	return nil
}
