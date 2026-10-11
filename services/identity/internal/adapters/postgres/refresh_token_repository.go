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

type RefreshTokenRepository struct {
	db *pgdb.DB
}

func NewRefreshTokenRepository(db *pgdb.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, token domain.RefreshToken) error {
	const query = `
		INSERT INTO refresh_tokens (id, session_id, token_hash, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := r.db.Conn(ctx).Exec(ctx, query,
		token.ID,
		token.SessionID,
		token.TokenHash,
		token.CreatedAt,
		token.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("postgres: insert refresh token: %w", err)
	}
	return nil
}

// FindByHashForUpdate must run inside a transaction. FOR UPDATE makes a
// second request for the same token wait here until the first one
// commits, and then see the row as already used.
func (r *RefreshTokenRepository) FindByHashForUpdate(ctx context.Context, hash []byte) (domain.RefreshToken, error) {
	const query = `
		SELECT id, session_id, token_hash, created_at, expires_at, used_at
		FROM refresh_tokens
		WHERE token_hash = $1
		FOR UPDATE`

	var token domain.RefreshToken

	err := r.db.Conn(ctx).QueryRow(ctx, query, hash).Scan(
		&token.ID,
		&token.SessionID,
		&token.TokenHash,
		&token.CreatedAt,
		&token.ExpiresAt,
		&token.UsedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RefreshToken{}, domain.ErrInvalidRefreshToken
	}
	if err != nil {
		return domain.RefreshToken{}, fmt.Errorf("postgres: find refresh token: %w", err)
	}

	return token, nil
}

func (r *RefreshTokenRepository) MarkUsed(ctx context.Context, id string, at time.Time) error {
	const query = `UPDATE refresh_tokens SET used_at = $2 WHERE id = $1`

	if _, err := r.db.Conn(ctx).Exec(ctx, query, id, at); err != nil {
		return fmt.Errorf("postgres: mark refresh token used: %w", err)
	}
	return nil
}
