package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kidrury/rest-pro/internal/domain"
)

type SessionRepository struct {
	pool *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{
		pool: pool,
	}
}

func (r *SessionRepository) Create(ctx context.Context, session domain.AuthSession) error {
	query := `
		INSERT INTO sessions(
		id, 
		family_id, 
		user_id, 
		refresh_token_hash, 
		created_at,
		expires_at
		)
		VALUES($1, $2, $3, $4, $5, $6)
	`

	_, err := r.pool.Exec(
		ctx,
		query,
		session.ID,
		session.FamilyID,
		session.UserID,
		session.RefreshTokenHash,
		session.CreatedAt,
		session.ExpiresAt,
	)

	if err != nil {
		return fmt.Errorf("create new session: %w", err)
	}

	return nil
}

// func (r *SessionRepository) FindByRefreshTokenHash(ctx context.Context, hash []byte) (domain.AuthSession, error) {
// 	var session domain.AuthSession

// 	query := `
// 		SELECT
// 			id,
// 			family_id,
// 			user_id,
// 			refresh_token_hash,
// 			created_at,
// 			expires_at,
// 			last_used_at,
// 			revoked_at,
// 			replaced_by
// 		FROM sessions
// 		WHERE refresh_token_hash = $1
// 	`

// 	row := r.pool.QueryRow(ctx, query, hash)

// 	err := row.Scan(
// 		&session.ID,
// 		&session.FamilyID,
// 		&session.UserID,
// 		&session.RefreshTokenHash,
// 		&session.CreatedAt,
// 		&session.ExpiresAt,
// 		&session.LastUsedAt,
// 		&session.RevokedAt,
// 		&session.ReplacedBy,
// 	)

// 	if err != nil {
// 		if errors.Is(err, pgx.ErrNoRows) {
// 			return domain.AuthSession{}, domain.ErrSessionNotFound
// 		}
// 		return domain.AuthSession{}, fmt.Errorf("get session by refresh token hash: %w\n", err)
// 	}

// 	return session, nil
// }

func (r *SessionRepository) Rotate(
	ctx context.Context,
	oldRefreshHash []byte,
	replacement domain.AuthSession,
	now time.Time,
	idleTTL time.Duration,
) (domain.AuthSession, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.AuthSession{}, fmt.Errorf("initialize replacement transaction: %w\n", err)
	}
	defer tx.Rollback(ctx)

	var oldSession domain.AuthSession

	fetchQuery := `
		SELECT
			id,
			family_id,
			user_id,
			refresh_token_hash,
			created_at,
			expires_at,
			last_used_at,
			revoked_at,
			replaced_by
		FROM sessions
		WHERE refresh_token_hash = $1
		FOR UPDATE
	`
	row := tx.QueryRow(ctx, fetchQuery, oldRefreshHash)

	err = row.Scan(
		&oldSession.ID,
		&oldSession.FamilyID,
		&oldSession.UserID,
		&oldSession.RefreshTokenHash,
		&oldSession.CreatedAt,
		&oldSession.ExpiresAt,
		&oldSession.LastUsedAt,
		&oldSession.RevokedAt,
		&oldSession.ReplacedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.AuthSession{}, domain.ErrSessionNotFound
		}
		return domain.AuthSession{}, fmt.Errorf("lock refresh token: %w\n", err)
	}

	if oldSession.RevokedAt != nil || oldSession.LastUsedAt != nil {
		return oldSession, domain.ErrSessionReuse
	}

	if !now.Before(oldSession.ExpiresAt) {
		return oldSession, domain.ErrSessionExpired
	}

	if !now.Before(oldSession.CreatedAt.Add(idleTTL)) {
		return oldSession, domain.ErrSessionExpired
	}

	insertQuery := `
		INSERT INTO sessions(
			id,
			family_id,
			user_id,
			refresh_token_hash,
			created_at,
			expires_at
		)
		VALUES($1, $2, $3, $4, $5, $6)
	`

	_, err = tx.Exec(
		ctx,
		insertQuery,
		replacement.ID,
		oldSession.FamilyID,
		oldSession.UserID,
		replacement.RefreshTokenHash,
		replacement.CreatedAt,
		oldSession.ExpiresAt,
	)

	if err != nil {
		return domain.AuthSession{}, fmt.Errorf(
			"create replacement refresh session: %w",
			err,
		)
	}

	updateQuery := `
		UPDATE sessions
		SET
			last_used_at = $1,
			replaced_by = $2
		WHERE id = $3
	`
	_, err = tx.Exec(ctx, updateQuery, now, replacement.ID, oldSession.ID)

	if err != nil {
		return domain.AuthSession{}, fmt.Errorf(
			"consume refresh session: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.AuthSession{}, fmt.Errorf("commit refresh rotation: %w\n", err)
	}

	return oldSession, nil
}

func (r *SessionRepository) RevokeFamily(ctx context.Context, familyID string) error {
	query := `
		UPDATE sessions
		SET revoked_at = NOW()
		WHERE family_id = $1 AND revoked_at IS NULL
	`

	_, err := r.pool.Exec(ctx, query, familyID)
	if err != nil {
		return fmt.Errorf("revoke auth session family: %w\n", err)
	}
	return nil
}

func (r *SessionRepository) Logout(ctx context.Context, hashedRefreshToken []byte) (domain.AuthSession, error) {
	query := `
		UPDATE sessions
		SET revoked_at = NOW()
		WHERE revoked_at IS NULL AND refresh_token_hash = $1
		RETURNING 
			id,
			family_id,
			user_id,
			refresh_token_hash,
			created_at,
			expires_at,
			last_used_at,
			revoked_at,
			replaced_by
	`

	row := r.pool.QueryRow(ctx, query, hashedRefreshToken)

	var session domain.AuthSession

	err := row.Scan(
		&session.ID,
		&session.FamilyID,
		&session.UserID,
		&session.RefreshTokenHash,
		&session.CreatedAt,
		&session.ExpiresAt,
		&session.LastUsedAt,
		&session.RevokedAt,
		&session.ReplacedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.AuthSession{}, domain.ErrSessionNotFound
		}
		return domain.AuthSession{}, fmt.Errorf("revoke auth session: %w\n", err)
	}

	return session, nil
}
