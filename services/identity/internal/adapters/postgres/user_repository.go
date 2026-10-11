package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sholana1/fintechdemo/identity/internal/domain"
)

const uniqueViolation = "23505"

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, user domain.User) error {
	const query = `
			INSERT INTO users (id, email, phone_number, password_hash, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.pool.Exec(ctx, query, user.ID, user.Email.String(), user.PhoneNumber.String(), user.PasswordHash, string(user.Status), user.CreatedAt, user.UpdatedAt)
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		switch pgErr.ConstraintName {
		case "users_email_key":
			return domain.ErrEmailTaken
		case "users_phone_number_key":
			return domain.ErrPhoneNumberTaken
		}
	}

	return fmt.Errorf("postgres: insert user: %w", err)
}
