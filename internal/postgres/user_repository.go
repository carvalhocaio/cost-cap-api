package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/carvalhocaio/cost-cap-api/internal/postgres/sqlcgen"
	"github.com/carvalhocaio/cost-cap-api/internal/user"
)

const usersEmailConstraint = "users_email_key"

type UserRepository struct {
	queries *sqlcgen.Queries
}

func NewUserRepository(db sqlcgen.DBTX) *UserRepository {
	return &UserRepository{queries: sqlcgen.New(db)}
}

func (r *UserRepository) Create(ctx context.Context, u user.User) (user.User, error) {
	row, err := r.queries.CreateUser(ctx, sqlcgen.CreateUserParams{
		ID:           u.ID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
	})
	if isUniqueViolation(err, usersEmailConstraint) {
		return user.User{}, user.ErrEmailTaken
	}
	if err != nil {
		return user.User{}, fmt.Errorf("insert user: %w", err)
	}

	return toUser(row), nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (user.User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return user.User{}, user.ErrNotFound
	}
	if err != nil {
		return user.User{}, fmt.Errorf("select user by email: %w", err)
	}

	return toUser(row), nil
}

func toUser(row sqlcgen.User) user.User {
	return user.User{
		ID:           row.ID,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		CreatedAt:    row.CreatedAt,
	}
}
