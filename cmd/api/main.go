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
	godotenv.Load()
	conf, err := config.Load()
	if err != nil {
		slog.Error("failed loading configurations", "error", err)
		os.Exit(1)
	}

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
