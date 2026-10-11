package app

import (
	"context"
	"fmt"
	"time"

	"github.com/sholana1/fintechdemo/identity/internal/domain"
)

type LogoutInput struct {
	SessionID string
	UserID    string
}

type Logout struct {
	sessions SessionRepository
	revoked  RevokedSessions
	now      func() time.Time
}

func NewLogout(sessions SessionRepository, revoked RevokedSessions, now func() time.Time) *Logout {
	return &Logout{sessions: sessions, revoked: revoked, now: now}
}

// Execute ends a session. It is safe to call twice: the second call
// finds the session already revoked and changes nothing.
func (uc *Logout) Execute(ctx context.Context, in LogoutInput) error {
	session, err := uc.sessions.FindByID(ctx, in.SessionID)
	if err != nil {
		return fmt.Errorf("logout: %w", err)
	}

	// Someone else's session is reported as missing, not as forbidden,
	// so session ids cannot be probed.
	if session.UserID != in.UserID {
		return domain.ErrSessionNotFound
	}

	if err := uc.sessions.Revoke(ctx, session.ID, domain.RevokeReasonLogout, uc.now()); err != nil {
		return fmt.Errorf("logout: %w", err)
	}

	if err := uc.revoked.Add(ctx, session.ID); err != nil {
		return fmt.Errorf("logout: publish revocation: %w", err)
	}

	return nil
}
