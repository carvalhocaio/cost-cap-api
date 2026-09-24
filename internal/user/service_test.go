package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/carvalhocaio/cost-cap-api/internal/user"
	"github.com/carvalhocaio/cost-cap-api/internal/validation"
)

type fakeRepository struct {
	byEmail map[string]user.User
}

func (r *fakeRepository) Create(_ context.Context, u user.User) (user.User, error) {
	if _, exists := r.byEmail[u.Email]; exists {
		return user.User{}, user.ErrEmailTaken
	}
	r.byEmail[u.Email] = u

	return u, nil
}

func (r *fakeRepository) GetByEmail(_ context.Context, email string) (user.User, error) {
	found, ok := r.byEmail[email]
	if !ok {
		return user.User{}, user.ErrNotFound
	}

	return found, nil
}

type fakeHasher struct {
	verifications int
}

func (h *fakeHasher) Hash(password string) (string, error) {
	return "hashed:" + password, nil
}

func (h *fakeHasher) Verify(password, encoded string) (bool, error) {
	h.verifications++
	return encoded == "hashed:"+password, nil
}

func newService(t *testing.T) (*user.Service, *fakeHasher) {
	t.Helper()

	hasher := &fakeHasher{}
	service, err := user.NewService(&fakeRepository{byEmail: map[string]user.User{}}, hasher)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	return service, hasher
}

func TestRegisterNormalizesAndHashes(t *testing.T) {
	service, _ := newService(t)

	registered, err := service.Register(t.Context(), user.Credentials{
		Email:    "  Tyler@Dema.io ",
		Password: "stressed-out",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if registered.Email != "tyler@dema.io" {
		t.Errorf("Email = %q, want %q", registered.Email, "tyler@dema.io")
	}
	if registered.PasswordHash != "hashed:stressed-out" {
		t.Errorf("PasswordHash = %q, want it hashed", registered.PasswordHash)
	}
	if registered.ID == uuid.Nil {
		t.Error("ID was not generated")
	}
}

func TestRegisterRejectsInvalidCredentials(t *testing.T) {
	tests := []struct {
		name      string
		creds     user.Credentials
		wantField string
	}{
		{"malformed email", user.Credentials{Email: "blurryface", Password: "stressed-out"}, "email"},
		{"display name email", user.Credentials{Email: "Tyler <tyler@dema.io>", Password: "stressed-out"}, "email"},
		{"short password", user.Credentials{Email: "tyler@dema.io", Password: "short"}, "password"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, _ := newService(t)

			_, err := service.Register(t.Context(), tt.creds)

			var validationErrs validation.Errors
			if !errors.As(err, &validationErrs) {
				t.Fatalf("error = %v, want validation.Errors", err)
			}
			if validationErrs[0].Field != tt.wantField {
				t.Errorf("Field = %q, want %q", validationErrs[0].Field, tt.wantField)
			}
		})
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	service, _ := newService(t)
	creds := user.Credentials{Email: "josh@dema.io", Password: "heathens-2016"}

	if _, err := service.Register(t.Context(), creds); err != nil {
		t.Fatalf("first register: %v", err)
	}

	if _, err := service.Register(t.Context(), creds); !errors.Is(err, user.ErrEmailTaken) {
		t.Fatalf("error = %v, want %v", err, user.ErrEmailTaken)
	}
}

func TestAuthenticate(t *testing.T) {
	registered := user.Credentials{Email: "tyler@dema.io", Password: "stressed-out"}

	tests := []struct {
		name    string
		creds   user.Credentials
		wantErr error
	}{
		{"valid credentials", user.Credentials{Email: " TYLER@dema.io", Password: "stressed-out"}, nil},
		{"wrong password", user.Credentials{Email: "tyler@dema.io", Password: "ride-slowly"}, user.ErrInvalidCredentials},
		{"unknown email", user.Credentials{Email: "nico@dema.io", Password: "stressed-out"}, user.ErrInvalidCredentials},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, hasher := newService(t)
			if _, err := service.Register(t.Context(), registered); err != nil {
				t.Fatalf("register: %v", err)
			}

			_, err := service.Authenticate(t.Context(), tt.creds)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if hasher.verifications != 1 {
				t.Errorf("verifications = %d, want 1 to keep timing uniform", hasher.verifications)
			}
		})
	}
}
