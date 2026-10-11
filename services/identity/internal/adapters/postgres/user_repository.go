package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sholana1/fintechdemo/identity/internal/domain"
	"github.com/sholana1/fintechdemo/pkg/pgdb"
)

const userColumns = `id, email, phone_number, password_hash, status, created_at, updated_at`

type UserRepository struct {
	db *pgdb.DB
}

func NewUserRepository(db *pgdb.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user domain.User) error {
	const query = `
		INSERT INTO users (` + userColumns + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.db.Conn(ctx).Exec(ctx, query,
		user.ID,
		user.Email.String(),
		user.PhoneNumber.String(),
		user.PasswordHash,
		string(user.Status),
		user.CreatedAt,
		user.UpdatedAt,
	)
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

func (r *UserRepository) FindByEmail(ctx context.Context, email domain.Email) (domain.User, error) {
	const query = `SELECT ` + userColumns + ` FROM users WHERE email = $1`

	return r.findOne(ctx, query, email.String())
}

func (r *UserRepository) FindByPhoneNumber(ctx context.Context, phone domain.PhoneNumber) (domain.User, error) {
	const query = `SELECT ` + userColumns + ` FROM users WHERE phone_number = $1`

	return r.findOne(ctx, query, phone.String())
}

func (r *UserRepository) findOne(ctx context.Context, query string, arg string) (domain.User, error) {
	var (
		user   domain.User
		email  string
		phone  string
		status string
	)

	err := r.db.Conn(ctx).QueryRow(ctx, query, arg).Scan(
		&user.ID, &email, &phone, &user.PasswordHash, &status, &user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("postgres: find user: %w", err)
	}

	// Stored values pass through the same constructors as new input, so
	// a corrupt row is caught here and not deep inside a use case.
	if user.Email, err = domain.NewEmail(email); err != nil {
		return domain.User{}, fmt.Errorf("postgres: stored email for user %s: %w", user.ID, err)
	}
	if user.PhoneNumber, err = domain.NewPhoneNumber(phone); err != nil {
		return domain.User{}, fmt.Errorf("postgres: stored phone for user %s: %w", user.ID, err)
	}

	user.Status = domain.UserStatus(status)

	return user, nil
}
