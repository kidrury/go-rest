package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kidrury/rest-pro/internal/auth"
	"github.com/kidrury/rest-pro/internal/authz"
	"github.com/kidrury/rest-pro/internal/domain"
)

type fakeUserRepository struct {
	users       map[string]domain.User
	getUserErr  error
	requestedID string
}

func (f *fakeUserRepository) GetUserByID(
	ctx context.Context,
	userID string,
) (domain.User, error) {
	f.requestedID = userID

	if f.getUserErr != nil {
		return domain.User{}, f.getUserErr
	}

	user, ok := f.users[userID]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}

	return user, nil
}

func newFakeUserRepository(users ...domain.User) *fakeUserRepository {
	repo := &fakeUserRepository{
		users: make(map[string]domain.User),
	}

	for _, user := range users {
		repo.users[user.ID] = user
	}

	return repo
}

func TestUserServiceGetUserByIDSelf(t *testing.T) {
	actor := domain.User{
		ID:        "user-123",
		Email:     "user@example.com",
		Role:      "user",
		CreatedAt: time.Now(),
	}

	repo := newFakeUserRepository(actor)
	service := NewUserService(repo)

	identity := auth.Identity{
		UserID: actor.ID,
	}

	result, err := service.GetUserByID(
		context.Background(),
		identity,
		actor.ID,
	)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}

	if result.ID != actor.ID {
		t.Fatalf(
			"result.ID = %q, want %q",
			result.ID,
			actor.ID,
		)
	}

	if result.Email != actor.Email {
		t.Fatalf(
			"result.Email = %q, want %q",
			result.Email,
			actor.Email,
		)
	}

	if repo.requestedID != actor.ID {
		t.Fatalf(
			"repository requested ID = %q, want %q",
			repo.requestedID,
			actor.ID,
		)
	}
}

func TestUserServiceGetUserByIDUserCannotReadOtherUser(t *testing.T) {
	actor := domain.User{
		ID:    "user-123",
		Email: "user@example.com",
		Role:  "user",
	}

	target := domain.User{
		ID:    "user-456",
		Email: "other@example.com",
		Role:  "user",
	}

	repo := newFakeUserRepository(actor, target)
	service := NewUserService(repo)

	identity := auth.Identity{
		UserID: actor.ID,
	}

	_, err := service.GetUserByID(
		context.Background(),
		identity,
		target.ID,
	)
	if err == nil {
		t.Fatal("GetUserByID() returned nil error")
	}

	var domainErr domain.Error
	if !errors.As(err, &domainErr) {
		t.Fatalf("error type = %T, want domain.Error", err)
	}

	if domainErr.Code != "FORBIDDEN" {
		t.Fatalf(
			"error code = %q, want %q",
			domainErr.Code,
			"FORBIDDEN",
		)
	}
}

func TestUserServiceAdminCanReadOtherUser(t *testing.T) {
	actor := domain.User{
		ID:    "admin-123",
		Email: "admin@example.com",
		Role:  "admin",
	}

	target := domain.User{
		ID:    "user-456",
		Email: "user@example.com",
		Role:  "user",
	}

	repo := newFakeUserRepository(actor, target)
	service := NewUserService(repo)

	identity := auth.Identity{
		UserID: actor.ID,
	}

	result, err := service.GetUserByID(
		context.Background(),
		identity,
		target.ID,
	)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}

	if result.ID != target.ID {
		t.Fatalf(
			"result.ID = %q, want %q",
			result.ID,
			target.ID,
		)
	}
}

func TestUserServiceActorNotFound(t *testing.T) {
	repo := newFakeUserRepository()
	service := NewUserService(repo)

	identity := auth.Identity{
		UserID: "missing-user",
	}

	_, err := service.GetUserByID(
		context.Background(),
		identity,
		"missing-user",
	)
	if err == nil {
		t.Fatal("GetUserByID() returned nil error")
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

func TestUserServiceRepositoryFailure(t *testing.T) {
	repositoryErr := errors.New("database unavailable")

	repo := &fakeUserRepository{
		getUserErr: repositoryErr,
	}

	service := NewUserService(repo)

	identity := auth.Identity{
		UserID: "user-123",
	}

	_, err := service.GetUserByID(
		context.Background(),
		identity,
		"user-123",
	)
	if err == nil {
		t.Fatal("GetUserByID() returned nil error")
	}

	if !errors.Is(err, repositoryErr) {
		t.Fatalf(
			"GetUserByID() error = %v, does not wrap %v",
			err,
			repositoryErr,
		)
	}
}

func TestUserServiceRequirePermission(t *testing.T) {
	tests := []struct {
		name       string
		role       string
		permission authz.Permission
		wantCode   string
	}{
		{
			name:       "user can read self",
			role:       "user",
			permission: authz.PermissionReadSelf,
			wantCode:   "",
		},
		{
			name:       "user cannot read any",
			role:       "user",
			permission: authz.PermissionReadAny,
			wantCode:   "FORBIDDEN",
		},
		{
			name:       "admin can read any",
			role:       "admin",
			permission: authz.PermissionReadAny,
			wantCode:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := domain.User{
				ID:   "user-123",
				Role: tt.role,
			}

			repo := newFakeUserRepository(user)
			service := NewUserService(repo)

			err := service.RequirePermission(
				context.Background(),
				auth.Identity{UserID: user.ID},
				tt.permission,
			)

			if tt.wantCode == "" {
				if err != nil {
					t.Fatalf(
						"RequirePermission() error = %v, want nil",
						err,
					)
				}
				return
			}

			if err == nil {
				t.Fatal("RequirePermission() returned nil error")
			}

			var domainErr domain.Error
			if !errors.As(err, &domainErr) {
				t.Fatalf(
					"error type = %T, want domain.Error",
					err,
				)
			}

			if domainErr.Code != tt.wantCode {
				t.Fatalf(
					"error code = %q, want %q",
					domainErr.Code,
					tt.wantCode,
				)
			}
		})
	}
}

func permissionFromString(value string) authz.Permission {
	return authz.Permission(value)
}
