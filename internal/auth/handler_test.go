package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/carvalhocaio/cost-cap-api/internal/auth"
	"github.com/carvalhocaio/cost-cap-api/internal/httpx"
	"github.com/carvalhocaio/cost-cap-api/internal/user"
	"github.com/carvalhocaio/cost-cap-api/internal/validation"
)

type fakeUsers struct {
	registerErr     error
	authenticateErr error
}

func (f fakeUsers) Register(context.Context, user.Credentials) (user.User, error) {
	return user.User{ID: uuid.Must(uuid.NewV7())}, f.registerErr
}

func (f fakeUsers) Authenticate(context.Context, user.Credentials) (user.User, error) {
	return user.User{ID: uuid.Must(uuid.NewV7())}, f.authenticateErr
}

type fakeTokens struct{}

func (fakeTokens) Issue(uuid.UUID) (auth.AccessToken, error) {
	return auth.AccessToken{Value: "signed.jwt.token", ExpiresAt: trenchEpoch}, nil
}

type endpoint func(*auth.Handler, http.ResponseWriter, *http.Request) error

func TestHandlerEndpoints(t *testing.T) {
	const validBody = `{"email":"tyler@dema.io","password":"stressed-out"}`

	signup := (*auth.Handler).Signup
	login := (*auth.Handler).Login

	tests := []struct {
		name       string
		endpoint   endpoint
		body       string
		users      fakeUsers
		wantStatus int
	}{
		{"signup succeeds", signup, validBody, fakeUsers{}, http.StatusCreated},
		{"signup with taken email", signup, validBody, fakeUsers{registerErr: user.ErrEmailTaken}, http.StatusConflict},
		{"signup with invalid fields", signup, validBody, fakeUsers{registerErr: validation.Errors{{Field: "email", Message: "must be a valid email address"}}}, http.StatusUnprocessableEntity},
		{"signup with malformed body", signup, `{"email":`, fakeUsers{}, http.StatusBadRequest},
		{"login succeeds", login, validBody, fakeUsers{}, http.StatusOK},
		{"login with invalid credentials", login, validBody, fakeUsers{authenticateErr: user.ErrInvalidCredentials}, http.StatusUnauthorized},
		{"login with unexpected failure", login, validBody, fakeUsers{authenticateErr: errors.New("connection reset")}, http.StatusInternalServerError},
	}

	adapt := httpx.NewAdapter(slog.New(slog.DiscardHandler))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := auth.NewHandler(tt.users, fakeTokens{})
			target := adapt(func(w http.ResponseWriter, r *http.Request) error {
				return tt.endpoint(handler, w, r)
			})

			recorder := httptest.NewRecorder()
			target.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body)))

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}

			if tt.wantStatus < http.StatusBadRequest {
				assertTokenIssued(t, recorder.Body)
				return
			}

			if contentType := recorder.Header().Get("Content-Type"); contentType != "application/problem+json" {
				t.Errorf("Content-Type = %q, want application/problem+json", contentType)
			}
		})
	}
}

func assertTokenIssued(t *testing.T, body io.Reader) {
	t.Helper()

	var response struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.AccessToken != "signed.jwt.token" || response.TokenType != "Bearer" {
		t.Errorf("response = %+v, want a bearer token", response)
	}
}
