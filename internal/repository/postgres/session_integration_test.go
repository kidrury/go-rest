//go:build integration

package postgres_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kidrury/rest-pro/internal/domain"
	"github.com/kidrury/rest-pro/internal/repository/postgres"
)

var integrationPool *pgxpool.Pool

func TestMain(m *testing.M) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")

	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/rest_pro_test?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"create test database pool: %v\n",
			err,
		)
		os.Exit(1)
	}

	if err := pool.Ping(ctx); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"ping test database: %v\n",
			err,
		)
		pool.Close()
		os.Exit(1)
	}

	integrationPool = pool

	if err := runMigrations(databaseURL); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"run test database migrations: %v\n",
			err,
		)
		pool.Close()
		os.Exit(1)
	}

	exitCode := m.Run()

	pool.Close()

	os.Exit(exitCode)
}

func runMigrations(databaseURL string) error {
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	migrationsDir, err := filepath.Abs(
		filepath.Join(workingDir, "../../../migrations"),
	)
	if err != nil {
		return fmt.Errorf("resolve migrations directory: %w", err)
	}

	sourceURL := (&url.URL{
		Scheme: "file",
		Path:   migrationsDir,
	}).String()

	migrator, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	defer func() {
		_, _ = migrator.Close()
	}()

	if err := migrator.Up(); err != nil &&
		!errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

func createTestUser(t *testing.T) string {
	t.Helper()

	ctx := context.Background()

	userID := uuid.NewString()
	email := fmt.Sprintf("%s@example.com", userID)

	const query = `
		INSERT INTO users (
			id,
			email,
			password_hash
		)
		VALUES ($1, $2, $3)
	`

	_, err := integrationPool.Exec(
		ctx,
		query,
		userID,
		email,
		"test-password-hash",
	)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	t.Cleanup(func() {
		_, _ = integrationPool.Exec(
			context.Background(),
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	})

	return userID
}

func refreshHash(token string) []byte {
	hash := sha256.Sum256([]byte(token))

	return hash[:]
}

func insertTestSession(
	t *testing.T,
	session domain.AuthSession,
) {
	t.Helper()

	const query = `
		INSERT INTO sessions (
			id,
			family_id,
			user_id,
			refresh_token_hash,
			created_at,
			expires_at,
			last_used_at,
			revoked_at,
			replaced_by
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9
		)
	`

	_, err := integrationPool.Exec(
		context.Background(),
		query,
		session.ID,
		session.FamilyID,
		session.UserID,
		session.RefreshTokenHash,
		session.CreatedAt,
		session.ExpiresAt,
		session.LastUsedAt,
		session.RevokedAt,
		session.ReplacedBy,
	)
	if err != nil {
		t.Fatalf("insert test session: %v", err)
	}
}

func TestSessionRepositoryRotate(t *testing.T) {
	userID := createTestUser(t)

	now := time.Now().UTC().Truncate(time.Microsecond)

	oldSession := domain.AuthSession{
		ID:               uuid.NewString(),
		FamilyID:         uuid.NewString(),
		UserID:           userID,
		RefreshTokenHash: refreshHash("old-refresh-token"),
		CreatedAt:        now.Add(-5 * time.Minute),
		ExpiresAt:        now.Add(24 * time.Hour),
	}

	insertTestSession(t, oldSession)

	replacement := domain.AuthSession{
		ID:               uuid.NewString(),
		RefreshTokenHash: refreshHash("new-refresh-token"),
		CreatedAt:        now,
	}

	repo := postgres.NewSessionRepository(integrationPool)

	returnedOld, err := repo.Rotate(
		context.Background(),
		oldSession.RefreshTokenHash,
		replacement,
		now,
		30*time.Minute,
	)
	if err != nil {
		t.Fatalf("Rotate() error = %v", err)
	}

	if returnedOld.ID != oldSession.ID {
		t.Fatalf(
			"returned old session ID = %q, want %q",
			returnedOld.ID,
			oldSession.ID,
		)
	}

	var (
		oldLastUsed   *time.Time
		oldReplacedBy *string
	)

	err = integrationPool.QueryRow(
		context.Background(),
		`
			SELECT last_used_at, replaced_by
			FROM sessions
			WHERE id = $1
		`,
		oldSession.ID,
	).Scan(
		&oldLastUsed,
		&oldReplacedBy,
	)
	if err != nil {
		t.Fatalf("read old session: %v", err)
	}

	if oldLastUsed == nil {
		t.Fatal("old session last_used_at is NULL")
	}

	if !oldLastUsed.Equal(now) {
		t.Fatalf(
			"old last_used_at = %v, want %v",
			*oldLastUsed,
			now,
		)
	}

	if oldReplacedBy == nil {
		t.Fatal("old session replaced_by is NULL")
	}

	if *oldReplacedBy != replacement.ID {
		t.Fatalf(
			"old replaced_by = %q, want %q",
			*oldReplacedBy,
			replacement.ID,
		)
	}

	var (
		newFamilyID string
		newUserID   string
		newHash     []byte
		newCreated  time.Time
		newExpires  time.Time
	)

	err = integrationPool.QueryRow(
		context.Background(),
		`
			SELECT
				family_id,
				user_id,
				refresh_token_hash,
				created_at,
				expires_at
			FROM sessions
			WHERE id = $1
		`,
		replacement.ID,
	).Scan(
		&newFamilyID,
		&newUserID,
		&newHash,
		&newCreated,
		&newExpires,
	)
	if err != nil {
		t.Fatalf("read replacement session: %v", err)
	}

	if newFamilyID != oldSession.FamilyID {
		t.Fatalf(
			"replacement family ID = %q, want %q",
			newFamilyID,
			oldSession.FamilyID,
		)
	}

	if newUserID != oldSession.UserID {
		t.Fatalf(
			"replacement user ID = %q, want %q",
			newUserID,
			oldSession.UserID,
		)
	}

	if string(newHash) != string(replacement.RefreshTokenHash) {
		t.Fatal("replacement refresh-token hash is incorrect")
	}

	if !newCreated.Equal(replacement.CreatedAt) {
		t.Fatalf(
			"replacement created_at = %v, want %v",
			newCreated,
			replacement.CreatedAt,
		)
	}

	if !newExpires.Equal(oldSession.ExpiresAt) {
		t.Fatalf(
			"replacement expires_at = %v, want %v",
			newExpires,
			oldSession.ExpiresAt,
		)
	}
}

func TestSessionRepositoryRotateRejectsReuse(t *testing.T) {
	userID := createTestUser(t)

	now := time.Now().UTC().Truncate(time.Microsecond)

	oldSession := domain.AuthSession{
		ID:               uuid.NewString(),
		FamilyID:         uuid.NewString(),
		UserID:           userID,
		RefreshTokenHash: refreshHash("reuse-token"),
		CreatedAt:        now.Add(-5 * time.Minute),
		ExpiresAt:        now.Add(24 * time.Hour),
	}

	insertTestSession(t, oldSession)

	repo := postgres.NewSessionRepository(integrationPool)

	firstReplacement := domain.AuthSession{
		ID:               uuid.NewString(),
		RefreshTokenHash: refreshHash("replacement-one"),
		CreatedAt:        now,
	}

	_, err := repo.Rotate(
		context.Background(),
		oldSession.RefreshTokenHash,
		firstReplacement,
		now,
		30*time.Minute,
	)
	if err != nil {
		t.Fatalf("first Rotate() error = %v", err)
	}

	secondReplacement := domain.AuthSession{
		ID:               uuid.NewString(),
		RefreshTokenHash: refreshHash("replacement-two"),
		CreatedAt:        now.Add(time.Second),
	}

	_, err = repo.Rotate(
		context.Background(),
		oldSession.RefreshTokenHash,
		secondReplacement,
		now.Add(time.Second),
		30*time.Minute,
	)

	if !errors.Is(err, domain.ErrSessionReuse) {
		t.Fatalf(
			"second Rotate() error = %v, want %v",
			err,
			domain.ErrSessionReuse,
		)
	}

	var count int

	err = integrationPool.QueryRow(
		context.Background(),
		`SELECT COUNT(*) FROM sessions WHERE user_id = $1`,
		userID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("count sessions: %v", err)
	}

	if count != 2 {
		t.Fatalf(
			"session count = %d, want 2",
			count,
		)
	}
}

func TestSessionRepositoryRotateRejectsExpiredSession(t *testing.T) {
	userID := createTestUser(t)

	now := time.Now().UTC().Truncate(time.Microsecond)

	oldSession := domain.AuthSession{
		ID:               uuid.NewString(),
		FamilyID:         uuid.NewString(),
		UserID:           userID,
		RefreshTokenHash: refreshHash("expired-token"),
		CreatedAt:        now.Add(-5 * time.Minute),
		ExpiresAt:        now.Add(-time.Minute),
	}

	insertTestSession(t, oldSession)

	replacement := domain.AuthSession{
		ID:               uuid.NewString(),
		RefreshTokenHash: refreshHash("replacement"),
		CreatedAt:        now,
	}

	repo := postgres.NewSessionRepository(integrationPool)

	_, err := repo.Rotate(
		context.Background(),
		oldSession.RefreshTokenHash,
		replacement,
		now,
		30*time.Minute,
	)

	if !errors.Is(err, domain.ErrSessionExpired) {
		t.Fatalf(
			"Rotate() error = %v, want %v",
			err,
			domain.ErrSessionExpired,
		)
	}
}

func TestSessionRepositoryRotateNotFound(t *testing.T) {
	repo := postgres.NewSessionRepository(integrationPool)

	_, err := repo.Rotate(
		context.Background(),
		refreshHash("does-not-exist"),
		domain.AuthSession{
			ID:               uuid.NewString(),
			RefreshTokenHash: refreshHash("replacement"),
			CreatedAt:        time.Now().UTC(),
		},
		time.Now().UTC(),
		30*time.Minute,
	)

	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf(
			"Rotate() error = %v, want %v",
			err,
			domain.ErrSessionNotFound,
		)
	}
}

func TestSessionRepositoryRotateRollsBackOnReplacementInsertFailure(t *testing.T) {
	userID := createTestUser(t)

	now := time.Now().UTC().Truncate(time.Microsecond)

	oldSession := domain.AuthSession{
		ID:               uuid.NewString(),
		FamilyID:         uuid.NewString(),
		UserID:           userID,
		RefreshTokenHash: refreshHash("old-token"),
		CreatedAt:        now.Add(-5 * time.Minute),
		ExpiresAt:        now.Add(24 * time.Hour),
	}

	insertTestSession(t, oldSession)

	duplicateReplacementID := uuid.NewString()

	existingReplacement := domain.AuthSession{
		ID:               duplicateReplacementID,
		FamilyID:         oldSession.FamilyID,
		UserID:           userID,
		RefreshTokenHash: refreshHash("already-exists"),
		CreatedAt:        now,
		ExpiresAt:        now.Add(24 * time.Hour),
	}

	insertTestSession(t, existingReplacement)

	replacement := domain.AuthSession{
		ID:               duplicateReplacementID,
		RefreshTokenHash: refreshHash("new-token"),
		CreatedAt:        now,
	}

	repo := postgres.NewSessionRepository(integrationPool)

	_, err := repo.Rotate(
		context.Background(),
		oldSession.RefreshTokenHash,
		replacement,
		now,
		30*time.Minute,
	)

	if err == nil {
		t.Fatal("Rotate() returned nil error, want insertion failure")
	}

	var (
		lastUsed   *time.Time
		replacedBy *string
	)

	err = integrationPool.QueryRow(
		context.Background(),
		`
			SELECT last_used_at, replaced_by
			FROM sessions
			WHERE id = $1
		`,
		oldSession.ID,
	).Scan(
		&lastUsed,
		&replacedBy,
	)
	if err != nil {
		t.Fatalf("read old session after rollback: %v", err)
	}

	if lastUsed != nil {
		t.Fatalf(
			"old last_used_at = %v, want NULL",
			*lastUsed,
		)
	}

	if replacedBy != nil {
		t.Fatalf(
			"old replaced_by = %q, want NULL",
			*replacedBy,
		)
	}
}

func TestSessionRepositoryRotateConcurrentReuse(t *testing.T) {
	userID := createTestUser(t)

	now := time.Now().UTC().Truncate(time.Microsecond)

	oldSession := domain.AuthSession{
		ID:               uuid.NewString(),
		FamilyID:         uuid.NewString(),
		UserID:           userID,
		RefreshTokenHash: refreshHash("concurrent-token"),
		CreatedAt:        now.Add(-5 * time.Minute),
		ExpiresAt:        now.Add(24 * time.Hour),
	}

	insertTestSession(t, oldSession)

	repo := postgres.NewSessionRepository(integrationPool)

	type result struct {
		err error
	}

	results := make(chan result, 2)

	for i := 0; i < 2; i++ {
		go func(i int) {
			replacement := domain.AuthSession{
				ID: uuid.NewString(),
				RefreshTokenHash: refreshHash(
					fmt.Sprintf("replacement-%d", i),
				),
				CreatedAt: now,
			}

			_, err := repo.Rotate(
				context.Background(),
				oldSession.RefreshTokenHash,
				replacement,
				now,
				30*time.Minute,
			)

			results <- result{err: err}
		}(i)
	}

	var successCount int
	var reuseCount int

	for i := 0; i < 2; i++ {
		result := <-results

		switch {
		case result.err == nil:
			successCount++

		case errors.Is(result.err, domain.ErrSessionReuse):
			reuseCount++

		default:
			t.Fatalf(
				"unexpected rotation error = %v",
				result.err,
			)
		}
	}

	if successCount != 1 {
		t.Fatalf(
			"successful rotations = %d, want 1",
			successCount,
		)
	}

	if reuseCount != 1 {
		t.Fatalf(
			"reuse errors = %d, want 1",
			reuseCount,
		)
	}

	var count int

	err := integrationPool.QueryRow(
		context.Background(),
		`SELECT COUNT(*) FROM sessions WHERE user_id = $1`,
		userID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("count sessions: %v", err)
	}

	if count != 2 {
		t.Fatalf(
			"session count = %d, want 2",
			count,
		)
	}
}
