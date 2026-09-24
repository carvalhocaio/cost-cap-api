package user

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound           = errors.New("user not found")
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}
