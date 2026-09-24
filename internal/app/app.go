package app

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kidrury/rest-pro/internal/auth"
	"github.com/kidrury/rest-pro/internal/authz"
	"github.com/kidrury/rest-pro/internal/config"
	"github.com/kidrury/rest-pro/internal/database"
	"github.com/kidrury/rest-pro/internal/http/handler"
	"github.com/kidrury/rest-pro/internal/http/middleware"
	"github.com/kidrury/rest-pro/internal/repository/postgres"
	"github.com/kidrury/rest-pro/internal/service"
)

type App struct {
	Handler http.Handler
	pool    *pgxpool.Pool
}

func New(config config.Config) (*App, error) {
	tm, err := auth.NewTokenManager(
		config.JWTSecret,
		config.JWTIssuer,
		config.JWTAudience,
		config.AccessTokenTTL,
	)
	if err != nil {
		return nil, err
	}

	pool, err := database.NewPostgresPool(context.Background(), config.DatabaseURL)
	if err != nil {
		return nil, err
	}

	userRepo := postgres.NewUserRepository(pool)

	sessionRepo := postgres.NewSessionRepository(pool)

	userService := service.NewUserService(userRepo)

	authService, err := service.NewAuthService(
		userRepo,
		sessionRepo,
		tm,
		config.RefreshTokenTTL,
		config.RefreshIdleTTL,
	)
	if err != nil {
		pool.Close()
		return nil, err
	}

	userHandler := handler.NewUserHandler(userService)
	authHandler := handler.NewAuthHandler(authService, config.AccessTokenTTL)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health/live", handler.Live)
	mux.HandleFunc("GET /health/ready", handler.Ready)

	mux.Handle(
		"POST /user",
		handler.HandlerFunc(handler.CreateUser),
	)
	mux.Handle(
		"POST /validate",
		handler.HandlerFunc(handler.TestValidation),
	)
	mux.Handle(
		"GET /user/{id}",
		middleware.Auth(tm)(handler.HandlerFunc(userHandler.GetUser)),
	)
	mux.Handle(
		"GET /user",
		middleware.Auth(tm)(middleware.RequirePermission(userService, authz.PermissionReadAny)(handler.HandlerFunc(handler.ListUsers))),
	)
	mux.Handle(
		"POST /auth/login",
		handler.HandlerFunc(authHandler.Login),
	)
	mux.Handle(
		"POST /auth/refresh",
		handler.HandlerFunc(authHandler.Refresh),
	)
	mux.Handle(
		"POST /auth/logout",
		middleware.Auth(tm)(handler.HandlerFunc(authHandler.Logout)),
	)

	root := middleware.Recover(mux)
	root = middleware.Log(root)
	root = middleware.RequestID(root)

	app := &App{
		Handler: root,
		pool:    pool,
	}

	return app, nil
}

func (a *App) Close() {
	if a.pool != nil {
		a.pool.Close()
	}
}
