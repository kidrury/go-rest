package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/kidrury/rest-pro/internal/app"
	"github.com/kidrury/rest-pro/internal/config"
)

func main() {
	// if err := godotenv.Load(); err != nil {
	// 	slog.Warn("could not load .env file", "error", err)
	// }
	godotenv.Load()
	conf, err := config.Load()
	if err != nil {
		slog.Error("failed loading configurations", "error", err)
		os.Exit(1)
	}

	// tm, err := auth.NewTokenManager(conf.JWTSecret, conf.JWTIssuer, conf.JWTAudience, conf.AccessTokenTTL)

	// if err != nil {
	// 	slog.Error("failed creating token manager", "error", err)
	// 	os.Exit(1)
	// }

	// pool, err := database.NewPostgresPool(context.Background(), conf.DatabaseURL)
	// if err != nil {
	// 	slog.Error("failed creating postgres pool", "error", err)
	// 	os.Exit(1)
	// }

	// userRepo := postgres.NewUserRepository(pool)

	// sessionRepo := postgres.NewSessionRepository(pool)

	// authService, err := service.NewAuthService(userRepo, sessionRepo, tm, conf.RefreshTokenTTL, conf.RefreshIdleTTL)
	// if err != nil {
	// 	slog.Error("failed creating auth service", "error", err)
	// 	os.Exit(1)
	// }

	// authHandler := handler.NewAuthHandler(authService, conf.AccessTokenTTL)

	// userService := service.NewUserService(userRepo)
	// userHandler := handler.NewUserHandler(userService)

	// mux := http.NewServeMux()

	// handler := middleware.Recover(mux)
	// handler = middleware.Log(handler)
	// handler = middleware.RequestID(handler)

	// mux.HandleFunc("GET /health/live", h.Live)
	// mux.HandleFunc("GET /health/ready", h.Ready)
	// mux.HandleFunc("GET /health/notexist", h.NotExistTest)
	// mux.HandleFunc("GET /health/created", h.CreatedTest)
	// mux.HandleFunc("GET /health/panic", h.PanicTest)
	// mux.HandleFunc("GET /health/panic-after-write", h.PanicAfterWriteTest)
	// mux.Handle("POST /user", h.HandlerFunc(h.CreateUser))
	// mux.Handle("POST /validate", h.HandlerFunc(h.TestValidation))
	// mux.Handle("GET /user/{id}", middleware.Auth(tm)(h.HandlerFunc(userHandler.GetUser)))
	// mux.Handle(
	// 	"GET /user",
	// 	middleware.Auth(tm)(middleware.RequirePermission(userService, authz.PermissionReadAny)(h.HandlerFunc(h.ListUsers))),
	// )
	// mux.Handle("POST /auth/login", h.HandlerFunc(authHandler.Login))
	// mux.Handle("POST /auth/refresh", middleware.Auth(tm)(h.HandlerFunc(authHandler.Refresh)))
	// mux.Handle("POST /auth/logout", middleware.Auth(tm)(h.HandlerFunc(authHandler.Logout)))

	application, err := app.New(*conf)
	if err != nil {
		slog.Error("failed wiring up the app", "error", err)
		os.Exit(1)
	}

	defer application.Close()

	server := http.Server{
		Addr:              conf.HTTPAddr,
		Handler:           application.Handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			slog.Error("failed running server", "error", err)
			os.Exit(1)
		}
	}()

	fmt.Printf("server started on port%s\n", conf.HTTPAddr)

	<-sigChan
	fmt.Println("received shutdown signal")

	ctx, cancel := context.WithTimeout(context.Background(), conf.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server failed shutting down gracefully", "error", err)
	}

}
