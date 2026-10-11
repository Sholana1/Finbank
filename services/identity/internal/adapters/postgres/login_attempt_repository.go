package postgres

import (
	"context"
	"fmt"

	"github.com/sholana1/fintechdemo/identity/internal/domain"
	"github.com/sholana1/fintechdemo/pkg/pgdb"
)

type LoginAttemptRepository struct {
	db *pgdb.DB
}

func NewLoginAttemptRepository(db *pgdb.DB) *LoginAttemptRepository {
	return &LoginAttemptRepository{db: db}
}

func (r *LoginAttemptRepository) Record(ctx context.Context, attempt domain.LoginAttempt) error {
	const query = `
		INSERT INTO login_attempts (id, user_id, outcome, device_id, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	// An empty UserID means the identifier matched nobody: store NULL.
	var userID *string
	if attempt.UserID != "" {
		userID = &attempt.UserID
	}

	_, err := r.db.Conn(ctx).Exec(ctx, query,
		attempt.ID,
		userID,
		string(attempt.Outcome),
		attempt.DeviceID,
		attempt.IPAddress,
		attempt.UserAgent,
		attempt.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("postgres: insert login attempt: %w", err)
	}
	return nil
}
