package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/carvalhocaio/cost-cap-api/internal/httpx"
	"github.com/carvalhocaio/cost-cap-api/internal/user"
)

type UserService interface {
	Register(ctx context.Context, creds user.Credentials) (user.User, error)
	Authenticate(ctx context.Context, creds user.Credentials) (user.User, error)
}

type TokenIssuer interface {
	Issue(userID uuid.UUID) (AccessToken, error)
}

type Handler struct {
	users  UserService
	tokens TokenIssuer
}

func NewHandler(users UserService, tokens TokenIssuer) *Handler {
	return &Handler{users: users, tokens: tokens}
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (c credentialsRequest) toCredentials() user.Credentials {
	return user.Credentials{Email: c.Email, Password: c.Password}
}

type tokenResponse struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
}

func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) error {
	request, err := httpx.DecodeJSON[credentialsRequest](w, r)
	if err != nil {
		return err
	}

	registered, err := h.users.Register(r.Context(), request.toCredentials())
	if errors.Is(err, user.ErrEmailTaken) {
		return httpx.NewError(http.StatusConflict, "email already registered", err)
	}
	if err != nil {
		return err
	}

	return h.respondWithToken(w, http.StatusCreated, registered.ID)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) error {
	request, err := httpx.DecodeJSON[credentialsRequest](w, r)
	if err != nil {
		return err
	}

	authenticated, err := h.users.Authenticate(r.Context(), request.toCredentials())
	if errors.Is(err, user.ErrInvalidCredentials) {
		return httpx.NewError(http.StatusUnauthorized, "invalid email or password", err)
	}
	if err != nil {
		return err
	}

	return h.respondWithToken(w, http.StatusOK, authenticated.ID)
}

func (h *Handler) respondWithToken(w http.ResponseWriter, status int, userID uuid.UUID) error {
	token, err := h.tokens.Issue(userID)
	if err != nil {
		return fmt.Errorf("issue token: %w", err)
	}

	return httpx.WriteJSON(w, status, tokenResponse{
		AccessToken: token.Value,
		TokenType:   "Bearer",
		ExpiresAt:   token.ExpiresAt,
	})
}
