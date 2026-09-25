package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/carvalhocaio/cost-cap-api/internal/auth"
	"github.com/carvalhocaio/cost-cap-api/internal/httpx"
)

type pinger interface {
	Ping(ctx context.Context) error
}

func newRouter(logger *slog.Logger, db pinger, authHandler *auth.Handler) http.Handler {
	adapt := httpx.NewAdapter(logger)

	mux := http.NewServeMux()
	mux.Handle("GET /healthz", adapt(healthCheck(db)))
	mux.Handle("POST /auth/signup", adapt(authHandler.Signup))
	mux.Handle("POST /auth/login", adapt(authHandler.Login))

	return httpx.Chain(mux, httpx.RequestLogger(logger), httpx.Recoverer(logger))
}

func healthCheck(db pinger) httpx.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if err := db.Ping(r.Context()); err != nil {
			return httpx.NewError(http.StatusServiceUnavailable, "database unavailable", err)
		}

		w.WriteHeader(http.StatusNoContent)
		return nil
	}
}
