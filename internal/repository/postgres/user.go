package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kidrury/rest-pro/internal/domain"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		pool: pool,
	}
}

func (r *UserRepository) CreateUser(ctx context.Context, user domain.User) error {
	// query := `
	// 	INSERT INTO users (id, email, password_hash, role, created_at)
	// 	VALUES ($1, $2, $3, $4, $5)
	// `
	// _, err := r.pool.Exec(ctx, query, user.ID, user.Email, user.PasswordHash, user.Role, user.CreatedAt)
	return nil
}
func (r *UserRepository) GetUserByID(ctx context.Context, userID string) (domain.User, error) {
	query := `
		SELECT
			id,
			email,
			password_hash,
			role,
			created_at
		FROM users
		WHERE id = $1
	`
	var user domain.User

	row := r.pool.QueryRow(ctx, query, userID)
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, fmt.Errorf("get user by id: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	const query = `
		SELECT id, email, password_hash, role, created_at
		FROM users
		WHERE email = $1
	`

	var user domain.User

	rows := r.pool.QueryRow(ctx, query, email)

	err := rows.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}

		return domain.User{}, err
	}

	return user, nil
}
