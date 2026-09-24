package config_test

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/carvalhocaio/cost-cap-api/internal/config"
)

var validSecret = strings.Repeat("s", 32)

func fakeEnv(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadAppliesDefaults(t *testing.T) {
	cfg, err := config.Load(fakeEnv(map[string]string{
		"DATABASE_URL": "postgres://localhost/costcap",
		"JWT_SECRET":   validSecret,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":8080")
	}
	if cfg.JWTIssuer != "dema" {
		t.Errorf("JWTIssuer = %q, want %q", cfg.JWTIssuer, "dema")
	}
	if cfg.AccessTokenTTL != 15*time.Minute {
		t.Errorf("AccessTokenTTL = %v, want %v", cfg.AccessTokenTTL, 15*time.Minute)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("ShutdownTimeout = %v, want %v", cfg.ShutdownTimeout, 10*time.Second)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelInfo)
	}
}

func TestLoadRejectsInvalidEnvironment(t *testing.T) {
	base := map[string]string{
		"DATABASE_URL": "postgres://localhost/costcap",
		"JWT_SECRET":   validSecret,
	}

	tests := []struct {
		name      string
		overrides map[string]string
		wantErr   string
	}{
		{"missing database url", map[string]string{"DATABASE_URL": ""}, "DATABASE_URL is required"},
		{"missing secret", map[string]string{"JWT_SECRET": ""}, "JWT_SECRET is required"},
		{"short secret", map[string]string{"JWT_SECRET": "short"}, "JWT_SECRET must be at least"},
		{"malformed ttl", map[string]string{"ACCESS_TOKEN_TTL": "soon"}, "ACCESS_TOKEN_TTL"},
		{"non-positive ttl", map[string]string{"ACCESS_TOKEN_TTL": "-1m"}, "ACCESS_TOKEN_TTL must be positive"},
		{"unknown log level", map[string]string{"LOG_LEVEL": "LOUD"}, "LOG_LEVEL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := make(map[string]string, len(base))
			for key, value := range base {
				env[key] = value
			}
			for key, value := range tt.overrides {
				env[key] = value
			}

			_, err := config.Load(fakeEnv(env))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}
