// Package config provides configuration parsing and validation for the application.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"time"
)

const minSecretLength = 32

// Config holds the application configuration values.
type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	JWTSecret       []byte
	JWTIssuer       string
	AccessTokenTTL  time.Duration
	ShutdownTimeout time.Duration
	LogLevel        slog.Level
}

// Load reads and parses application configuration using the provided environment lookup function.
func Load(getenv func(string) string) (Config, error) {
	env := &envReader{getenv: getenv}

	cfg := Config{
		HTTPAddr:        env.optional("HTTP_ADDR", ":8080"),
		DatabaseURL:     env.required("DATABASE_URL"),
		JWTSecret:       env.secret("JWT_SECRET", minSecretLength),
		JWTIssuer:       env.optional("JWT_ISSUER", "dema"),
		AccessTokenTTL:  env.duration("ACCESS_TOKEN_TTL", 15*time.Minute),
		ShutdownTimeout: env.duration("SHUTDOWN_TIMEOUT", 10*time.Second),
		LogLevel:        env.logLevel("LOG_LEVEL", slog.LevelInfo),
	}

	if err := errors.Join(env.errs...); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

type envReader struct {
	getenv func(string) string
	errs   []error
}

func (r *envReader) required(key string) string {
	value := r.getenv(key)
	if value == "" {
		r.errs = append(r.errs, fmt.Errorf("%s is required", key))
	}

	return value
}

func (r *envReader) optional(key, fallback string) string {
	if value := r.getenv(key); value != "" {
		return value
	}

	return fallback
}

func (r *envReader) secret(key string, minLength int) []byte {
	value := r.required(key)
	if value != "" && len(value) < minLength {
		r.errs = append(r.errs, fmt.Errorf("%s must be at least %d bytes", key, minLength))
	}

	return []byte(value)
}

func (r *envReader) duration(key string, fallback time.Duration) time.Duration {
	raw := r.getenv(key)
	if raw == "" {
		return fallback
	}

	value, err := time.ParseDuration(raw)
	switch {
	case err != nil:
		r.errs = append(r.errs, fmt.Errorf("%s: %w", key, err))
	case value <= 0:
		r.errs = append(r.errs, fmt.Errorf("%s must be positive", key))
	}

	return value
}

func (r *envReader) logLevel(key string, fallback slog.Level) slog.Level {
	raw := r.getenv(key)
	if raw == "" {
		return fallback
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(raw)); err != nil {
		r.errs = append(r.errs, fmt.Errorf("%s: %w", key, err))
	}

	return level
}
