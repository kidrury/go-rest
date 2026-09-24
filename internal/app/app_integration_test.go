//go:build integration

package app_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
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
	"github.com/kidrury/rest-pro/internal/app"
	"github.com/kidrury/rest-pro/internal/auth"
	"github.com/kidrury/rest-pro/internal/config"
	"github.com/kidrury/rest-pro/internal/http/response"
)

func setupTestApp(t *testing.T) (*app.App, *pgxpool.Pool, *httptest.Server) {
	t.Helper()

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
		t.Fatalf("create test database pool: %v", err)
	}

	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping test database: %v", err)
	}

	if err := runMigrations(databaseURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	conf := config.Config{
		HTTPAddr:        ":0",
		DatabaseURL:     databaseURL,
		JWTSecret:       "01234567890123456789012345678901",
		JWTIssuer:       "rest-pro-test",
		JWTAudience:     "rest-pro-test-api",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
		RefreshIdleTTL:  30 * time.Minute,
		ShutdownTimeout: 5 * time.Second,
	}

	application, err := app.New(conf)
	if err != nil {
		t.Fatalf("create application: %v", err)
	}

	t.Cleanup(application.Close)

	server := httptest.NewTLSServer(application.Handler)

	t.Cleanup(server.Close)

	return application, pool, server
}

func runMigrations(databaseURL string) error {
	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}

	migrationsDir, err := filepath.Abs(
		filepath.Join(workingDir, "../../migrations"),
	)
	if err != nil {
		return err
	}

	sourceURL := (&url.URL{
		Scheme: "file",
		Path:   migrationsDir,
	}).String()

	migrator, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		return err
	}

	defer func() {
		_, _ = migrator.Close()
	}()

	err = migrator.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}

	return err
}

func createTestUser(
	t *testing.T,
	pool *pgxpool.Pool,
) (string, string, string) {
	t.Helper()

	ctx := context.Background()

	userID := uuid.NewString()
	email := userID + "@example.com"
	password := "correct-password"

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash test password: %v", err)
	}

	_, err = pool.Exec(
		ctx,
		`
			INSERT INTO users (
				id,
				email,
				password_hash
			)
			VALUES ($1, $2, $3)
		`,
		userID,
		email,
		passwordHash,
	)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(
			context.Background(),
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	})

	return userID, email, password
}

func TestAuthHTTPFlow(t *testing.T) {
	_, pool, server := setupTestApp(t)

	userID, email, password := createTestUser(t, pool)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}

	client := server.Client()
	client.Jar = jar

	loginBody, err := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	if err != nil {
		t.Fatalf("marshal login request: %v", err)
	}

	loginResponse, err := client.Post(
		server.URL+"/auth/login",
		"application/json",
		bytes.NewReader(loginBody),
	)
	if err != nil {
		t.Fatalf("login request: %v", err)
	}
	defer loginResponse.Body.Close()

	if loginResponse.StatusCode != http.StatusNoContent {
		t.Fatalf(
			"login status = %d, want %d",
			loginResponse.StatusCode,
			http.StatusNoContent,
		)
	}

	accessCookie := findCookie(
		loginResponse.Cookies(),
		response.AcessCookieName,
	)

	refreshCookie := findCookie(
		loginResponse.Cookies(),
		response.RefreshCookieName,
	)

	if accessCookie == nil {
		t.Fatal("login did not return access cookie")
	}

	if refreshCookie == nil {
		t.Fatal("login did not return refresh cookie")
	}

	assertAccessCookie(t, accessCookie)
	assertRefreshCookie(t, refreshCookie)

	oldRefreshToken := cookieValue(
		t,
		client,
		server.URL,
		"/auth/refresh",
		response.RefreshCookieName,
	)

	if oldRefreshToken == "" {
		t.Fatal("cookie jar does not contain refresh token after login")
	}

	// ----- Refresh -----

	refreshResponse, err := client.Post(
		server.URL+"/auth/refresh",
		"",
		nil,
	)
	if err != nil {
		t.Fatalf("refresh request: %v", err)
	}
	defer refreshResponse.Body.Close()

	if refreshResponse.StatusCode != http.StatusNoContent {
		t.Fatalf(
			"refresh status = %d, want %d",
			refreshResponse.StatusCode,
			http.StatusNoContent,
		)
	}

	newRefreshToken := cookieValue(
		t,
		client,
		server.URL,
		"/auth/refresh",
		response.RefreshCookieName,
	)

	if newRefreshToken == "" {
		t.Fatal("cookie jar does not contain replacement refresh token")
	}

	if newRefreshToken == oldRefreshToken {
		t.Fatal("refresh returned the same refresh token")
	}

	// ----- Verify database rotation -----

	oldHash := sha256.Sum256([]byte(oldRefreshToken))
	newHash := sha256.Sum256([]byte(newRefreshToken))

	var (
		oldSessionID  string
		oldLastUsed   *time.Time
		oldReplacedBy *string
	)

	err = pool.QueryRow(
		context.Background(),
		`
			SELECT
				id,
				last_used_at,
				replaced_by
			FROM sessions
			WHERE refresh_token_hash = $1
		`,
		oldHash[:],
	).Scan(
		&oldSessionID,
		&oldLastUsed,
		&oldReplacedBy,
	)
	if err != nil {
		t.Fatalf("query old session: %v", err)
	}

	if oldLastUsed == nil {
		t.Fatal("old session was not consumed")
	}

	if oldReplacedBy == nil {
		t.Fatal("old session has no replacement")
	}

	var newSessionID string

	err = pool.QueryRow(
		context.Background(),
		`
			SELECT id
			FROM sessions
			WHERE refresh_token_hash = $1
		`,
		newHash[:],
	).Scan(&newSessionID)
	if err != nil {
		t.Fatalf("query replacement session: %v", err)
	}

	if *oldReplacedBy != newSessionID {
		t.Fatalf(
			"old replaced_by = %q, want %q",
			*oldReplacedBy,
			newSessionID,
		)
	}

	if oldSessionID == newSessionID {
		t.Fatal("replacement session has same ID as old session")
	}

	// ----- Logout -----

	logoutRequest, err := http.NewRequest(
		http.MethodPost,
		server.URL+"/auth/logout",
		nil,
	)
	if err != nil {
		t.Fatalf("create logout request: %v", err)
	}

	logoutResponse, err := client.Do(logoutRequest)
	if err != nil {
		t.Fatalf("logout request: %v", err)
	}
	defer logoutResponse.Body.Close()

	if logoutResponse.StatusCode != http.StatusNoContent {
		t.Fatalf(
			"logout status = %d, want %d",
			logoutResponse.StatusCode,
			http.StatusNoContent,
		)
	}

	assertCookieCleared(
		t,
		logoutResponse.Cookies(),
		response.AcessCookieName,
	)

	assertCookieCleared(
		t,
		logoutResponse.Cookies(),
		response.RefreshCookieName,
	)

	var revokedAt *time.Time

	err = pool.QueryRow(
		context.Background(),
		`
			SELECT revoked_at
			FROM sessions
			WHERE id = $1
		`,
		newSessionID,
	).Scan(&revokedAt)
	if err != nil {
		t.Fatalf("query logged-out session: %v", err)
	}

	if revokedAt == nil {
		t.Fatal("logout did not revoke refresh session")
	}

	// Make sure the session belongs to the user we created.
	var sessionUserID string

	err = pool.QueryRow(
		context.Background(),
		`
			SELECT user_id
			FROM sessions
			WHERE id = $1
		`,
		newSessionID,
	).Scan(&sessionUserID)
	if err != nil {
		t.Fatalf("query session user ID: %v", err)
	}

	if sessionUserID != userID {
		t.Fatalf(
			"session user ID = %q, want %q",
			sessionUserID,
			userID,
		)
	}
}

func findCookie(
	cookies []*http.Cookie,
	name string,
) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}

	return nil
}

func assertAccessCookie(t *testing.T, cookie *http.Cookie) {
	t.Helper()

	if !cookie.HttpOnly {
		t.Error("access cookie is not HttpOnly")
	}

	if !cookie.Secure {
		t.Error("access cookie is not Secure")
	}

	if cookie.Path != "/" {
		t.Fatalf(
			"access cookie Path = %q, want %q",
			cookie.Path,
			"/",
		)
	}

	if cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf(
			"access cookie SameSite = %v, want Strict",
			cookie.SameSite,
		)
	}

	if cookie.MaxAge <= 0 {
		t.Fatalf(
			"access cookie MaxAge = %d, want positive",
			cookie.MaxAge,
		)
	}
}

func assertRefreshCookie(t *testing.T, cookie *http.Cookie) {
	t.Helper()

	if !cookie.HttpOnly {
		t.Error("refresh cookie is not HttpOnly")
	}

	if !cookie.Secure {
		t.Error("refresh cookie is not Secure")
	}

	if cookie.Path != "/auth" {
		t.Fatalf(
			"refresh cookie Path = %q, want %q",
			cookie.Path,
			"/auth",
		)
	}

	if cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf(
			"refresh cookie SameSite = %v, want Strict",
			cookie.SameSite,
		)
	}

	if cookie.MaxAge <= 0 {
		t.Fatalf(
			"refresh cookie MaxAge = %d, want positive",
			cookie.MaxAge,
		)
	}
}

func assertCookieCleared(
	t *testing.T,
	cookies []*http.Cookie,
	name string,
) {
	t.Helper()

	cookie := findCookie(cookies, name)
	if cookie == nil {
		t.Fatalf(
			"response did not clear cookie %q",
			name,
		)
	}

	if cookie.MaxAge != -1 {
		t.Fatalf(
			"cleared cookie %q MaxAge = %d, want -1",
			name,
			cookie.MaxAge,
		)
	}

	if cookie.Value != "" {
		t.Fatalf(
			"cleared cookie %q value = %q, want empty",
			name,
			cookie.Value,
		)
	}
}

func cookieValue(
	t *testing.T,
	client *http.Client,
	serverURL string,
	path string,
	name string,
) string {
	t.Helper()

	u, err := url.Parse(serverURL + path)
	if err != nil {
		t.Fatalf("parse test URL: %v", err)
	}

	for _, cookie := range client.Jar.Cookies(u) {
		if cookie.Name == name {
			return cookie.Value
		}
	}

	return ""
}

func TestAuthHTTPRefreshInvalidToken(t *testing.T) {
	_, _, server := setupTestApp(t)

	client := server.Client()

	request, err := http.NewRequest(
		http.MethodPost,
		server.URL+"/auth/refresh",
		nil,
	)
	if err != nil {
		t.Fatalf("create refresh request: %v", err)
	}

	request.AddCookie(&http.Cookie{
		Name:  response.RefreshCookieName,
		Value: "definitely-not-a-valid-refresh-token",
	})

	result, err := client.Do(request)
	if err != nil {
		t.Fatalf("refresh request: %v", err)
	}
	defer result.Body.Close()

	if result.StatusCode != http.StatusUnauthorized {
		t.Fatalf(
			"refresh status = %d, want %d",
			result.StatusCode,
			http.StatusUnauthorized,
		)
	}
}
