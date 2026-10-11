package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/sholana1/fintechdemo/identity/internal/domain"
)

type RefreshSessionInput struct {
	RefreshToken string
}

type RefreshSessionDeps struct {
	RefreshTokens RefreshTokenRepository
	Sessions      SessionRepository
	Tokens        *SessionTokens
	Revoked       RevokedSessions
	Tx            Transactor
	Logger        *slog.Logger
	Now           func() time.Time
}

type RefreshSession struct {
	deps RefreshSessionDeps
}

func NewRefreshSession(deps RefreshSessionDeps) *RefreshSession {
	return &RefreshSession{deps: deps}
}

// Execute swaps a refresh token for a new pair. A refresh token works
// exactly once. Presenting one a second time means two parties hold it,
// so one of them is a thief: the whole session is revoked.
func (uc *RefreshSession) Execute(ctx context.Context, in RefreshSessionInput) (TokenPair, error) {
	hash := HashRefreshToken(in.RefreshToken)
	now := uc.deps.Now()

	var (
		pair            TokenPair
		revokedForReuse string
	)

	err := uc.deps.Tx.WithinTx(ctx, func(ctx context.Context) error {
		token, err := uc.deps.RefreshTokens.FindByHashForUpdate(ctx, hash)
		if err != nil {
			return err
		}

		if token.IsUsed() {
			if err := uc.deps.Sessions.Revoke(ctx, token.SessionID, domain.RevokeReasonRefreshTokenReuse, now); err != nil {
				return err
			}
			// Return nil so the revocation is committed.
			revokedForReuse = token.SessionID
			return nil
		}

		if token.IsExpired(now) {
			return domain.ErrInvalidRefreshToken
		}

		session, err := uc.deps.Sessions.FindByID(ctx, token.SessionID)
		if err != nil {
			return err
		}
		if !session.IsActive(now) {
			return domain.ErrInvalidRefreshToken
		}

		if err := uc.deps.RefreshTokens.MarkUsed(ctx, token.ID, now); err != nil {
			return err
		}

		issued, err := uc.deps.Tokens.Issue(ctx, session, now)
		if err != nil {
			return err
		}
		pair = issued
		return nil
	})
	if err != nil {
		return TokenPair{}, fmt.Errorf("refresh session: %w", err)
	}

	if revokedForReuse != "" {
		uc.deps.Logger.Warn("refresh token reuse detected, session revoked", "session_id", revokedForReuse)
		if err := uc.deps.Revoked.Add(ctx, revokedForReuse); err != nil {
			uc.deps.Logger.Error("refresh session: could not publish revocation", "session_id", revokedForReuse, "error", err)
		}
		return TokenPair{}, domain.ErrRefreshTokenReuse
	}

	return pair, nil
}
