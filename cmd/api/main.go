package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/shseooo/go-architecture/internal/bootstrap"
	"github.com/shseooo/go-architecture/internal/platform/database"
)

const (
	defaultTimeout  = 30 * time.Second
	defaultAddress  = ":9090"
	shutdownTimeout = 10 * time.Second
)

func init() {
	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file found, using environment variables")
	}
}

// @title           Shop API
// @version         1.0
// @description     회원/상품/주문 API — modular monolith, stdlib net/http, sqlc.
// @host            localhost:9090
// @BasePath        /
func main() {
	if err := run(); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	db, err := database.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	srv := &http.Server{
		Addr:    serverAddress(),
		Handler: bootstrap.Handler(db, requestTimeout()),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("starting server", "address", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func serverAddress() string {
	if addr := os.Getenv("SERVER_ADDRESS"); addr != "" {
		return addr
	}
	return defaultAddress
}

func requestTimeout() time.Duration {
	if v := os.Getenv("CONTEXT_TIMEOUT"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			return time.Duration(secs) * time.Second
		}
	}
	return defaultTimeout
}
