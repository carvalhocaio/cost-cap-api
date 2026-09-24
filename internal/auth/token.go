package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidToken = errors.New("invalid token")

type AccessToken struct {
	Value     string
	ExpiresAt time.Time
}

type TokenManager struct {
	secret []byte
	issuer string
	ttl    time.Duration
	now    func() time.Time
}

func NewTokenManager(secret []byte, issuer string, ttl time.Duration, now func() time.Time) *TokenManager {
	return &TokenManager{secret: secret, issuer: issuer, ttl: ttl, now: now}
}

func (m *TokenManager) Issue(userID uuid.UUID) (AccessToken, error) {
	issuedAt := m.now()
	expiresAt := issuedAt.Add(m.ttl)

	claims := jwt.RegisteredClaims{
		Issuer:    m.issuer,
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(issuedAt),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return AccessToken{}, fmt.Errorf("sign token: %w", err)
	}

	return AccessToken{Value: signed, ExpiresAt: expiresAt}, nil
}

func (m *TokenManager) Verify(raw string) (uuid.UUID, error) {
	var claims jwt.RegisteredClaims

	_, err := jwt.ParseWithClaims(raw, &claims,
		func(*jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(m.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(m.now),
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: malformed subject", ErrInvalidToken)
	}

	return userID, nil
}
