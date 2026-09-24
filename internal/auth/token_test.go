package auth_test

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/carvalhocaio/cost-cap-api/internal/auth"
)

var (
	secret      = []byte("a-secret-long-enough-for-hs256-signing")
	trenchEpoch = time.Date(2018, time.October, 5, 0, 0, 0, 0, time.UTC)
)

func fixedClock(at time.Time) func() time.Time {
	return func() time.Time { return at }
}

func TestTokenManagerRoundTrip(t *testing.T) {
	manager := auth.NewTokenManager(secret, "dema", 15*time.Minute, fixedClock(trenchEpoch))
	userID := uuid.Must(uuid.NewV7())

	token, err := manager.Issue(userID)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	if want := trenchEpoch.Add(15 * time.Minute); !token.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", token.ExpiresAt, want)
	}

	got, err := manager.Verify(token.Value)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got != userID {
		t.Errorf("subject = %v, want %v", got, userID)
	}
}

func TestTokenManagerRejectsInvalidTokens(t *testing.T) {
	issuer := auth.NewTokenManager(secret, "dema", 15*time.Minute, fixedClock(trenchEpoch))
	token, err := issuer.Issue(uuid.Must(uuid.NewV7()))
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{
		Issuer:    "dema",
		Subject:   uuid.Must(uuid.NewV7()).String(),
		ExpiresAt: jwt.NewNumericDate(trenchEpoch.Add(time.Hour)),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign unsigned token: %v", err)
	}

	tests := []struct {
		name     string
		verifier *auth.TokenManager
		raw      string
	}{
		{"expired", auth.NewTokenManager(secret, "dema", 15*time.Minute, fixedClock(trenchEpoch.Add(time.Hour))), token.Value},
		{"foreign issuer", auth.NewTokenManager(secret, "trench", 15*time.Minute, fixedClock(trenchEpoch)), token.Value},
		{"wrong secret", auth.NewTokenManager([]byte("another-secret-long-enough-for-hs256"), "dema", 15*time.Minute, fixedClock(trenchEpoch)), token.Value},
		{"alg none", issuer, unsigned},
		{"garbage", issuer, "not.a.jwt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.verifier.Verify(tt.raw); !errors.Is(err, auth.ErrInvalidToken) {
				t.Errorf("error = %v, want %v", err, auth.ErrInvalidToken)
			}
		})
	}
}
