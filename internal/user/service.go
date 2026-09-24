package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/carvalhocaio/cost-cap-api/internal/validation"
)

const (
	minPasswordLength = 8
	maxPasswordLength = 128
	timingEqualizer   = "timing-equalizer"
)

type Repository interface {
	Create(ctx context.Context, u User) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, encoded string) (bool, error)
}

type Credentials struct {
	Email    string
	Password string
}

func (c Credentials) normalized() Credentials {
	c.Email = strings.ToLower(strings.TrimSpace(c.Email))
	return c
}

func (c Credentials) validate() error {
	var errs validation.Errors

	if address, err := mail.ParseAddress(c.Email); err != nil || address.Address != c.Email {
		errs.Add("email", "must be a valid email address")
	}

	if length := utf8.RuneCountInString(c.Password); length < minPasswordLength || length > maxPasswordLength {
		errs.Add("password", fmt.Sprintf("must be between %d and %d characters", minPasswordLength, maxPasswordLength))
	}

	return errs.OrNil()
}

type Service struct {
	repo      Repository
	hasher    PasswordHasher
	dummyHash string
}

func NewService(repo Repository, hasher PasswordHasher) (*Service, error) {
	dummyHash, err := hasher.Hash(timingEqualizer)
	if err != nil {
		return nil, fmt.Errorf("hash timing equalizer: %w", err)
	}

	return &Service{repo: repo, hasher: hasher, dummyHash: dummyHash}, nil
}

func (s *Service) Register(ctx context.Context, creds Credentials) (User, error) {
	creds = creds.normalized()
	if err := creds.validate(); err != nil {
		return User{}, err
	}

	hash, err := s.hasher.Hash(creds.Password)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return User{}, fmt.Errorf("generate user id: %w", err)
	}

	return s.repo.Create(ctx, User{ID: id, Email: creds.Email, PasswordHash: hash})
}

func (s *Service) Authenticate(ctx context.Context, creds Credentials) (User, error) {
	creds = creds.normalized()

	found, err := s.repo.GetByEmail(ctx, creds.Email)
	switch {
	case errors.Is(err, ErrNotFound):
		_, _ = s.hasher.Verify(creds.Password, s.dummyHash)
		return User{}, ErrInvalidCredentials
	case err != nil:
		return User{}, fmt.Errorf("find user: %w", err)
	}

	matches, err := s.hasher.Verify(creds.Password, found.PasswordHash)
	if err != nil {
		return User{}, fmt.Errorf("verify password: %w", err)
	}
	if !matches {
		return User{}, ErrInvalidCredentials
	}

	return found, nil
}
