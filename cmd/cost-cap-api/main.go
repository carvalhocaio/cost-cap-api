package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/carvalhocaio/cost-cap-api/internal/auth"
	"github.com/carvalhocaio/cost-cap-api/internal/config"
	"github.com/carvalhocaio/cost-cap-api/internal/postgres"
	"github.com/carvalhocaio/cost-cap-api/internal/user"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	if err := postgres.Migrate(ctx, pool, logger); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	users, err := user.NewService(postgres.NewUserRepository(pool), auth.NewArgon2idHasher(auth.DefaultArgon2idParams))
	if err != nil {
		return fmt.Errorf("create user service: %w", err)
	}

	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.AccessTokenTTL, time.Now)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           newRouter(logger, pool, auth.NewHandler(users, tokens)),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", cfg.HTTPAddr, "version", version)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done():
	}

	logger.Info("shutting down", "timeout", cfg.ShutdownTimeout)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}
