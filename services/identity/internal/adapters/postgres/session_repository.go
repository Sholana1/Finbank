package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/sholana1/fintechdemo/identity/internal/domain"
	"github.com/sholana1/fintechdemo/pkg/pgdb"
)

type SessionRepository struct {
	db *pgdb.DB
}

func NewSessionRepository(db *pgdb.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, session domain.Session) error {
	const query = `
		INSERT INTO sessions (id, user_id, device_id, ip_address, user_agent, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.db.Conn(ctx).Exec(ctx, query,
		session.ID,
		session.UserID,
		session.DeviceID,
		session.IPAddress,
		session.UserAgent,
		session.CreatedAt,
		session.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("postgres: insert session: %w", err)
	}
	return nil
}

func (r *SessionRepository) FindByID(ctx context.Context, id string) (domain.Session, error) {
	const query = `
		SELECT id, user_id, device_id, ip_address, user_agent, created_at, expires_at, revoked_at
		FROM sessions
		WHERE id = $1`

	var session domain.Session

	err := r.db.Conn(ctx).QueryRow(ctx, query, id).Scan(
		&session.ID,
		&session.UserID,
		&session.DeviceID,
		&session.IPAddress,
		&session.UserAgent,
		&session.CreatedAt,
		&session.ExpiresAt,
		&session.RevokedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return domain.Session{}, domain.ErrSessionNotFound
	}
	if err != nil {
		return domain.Session{}, fmt.Errorf("postgres: find session: %w", err)
	}

	return session, nil
}

// Revoke only touches a session that is still live, which is what makes
// it safe to call more than once.
func (r *SessionRepository) Revoke(ctx context.Context, id string, reason domain.RevokeReason, at time.Time) error {
	const query = `
		UPDATE sessions
		SET revoked_at = $2, revoke_reason = $3
		WHERE id = $1 AND revoked_at IS NULL`

	if _, err := r.db.Conn(ctx).Exec(ctx, query, id, at, string(reason)); err != nil {
		return fmt.Errorf("postgres: revoke session: %w", err)
	}
	return nil
}
