package app

import (
	"context"
	"time"

	"github.com/sholana1/fintechdemo/identity/internal/domain"
)

// The interfaces in this file are the ports: everything the use cases
// need from the outside world. The adapters implement them.

type UserRepository interface {
	// Create returns domain.ErrEmailTaken or domain.ErrPhoneNumberTaken
	// on a duplicate.
	Create(ctx context.Context, user domain.User) error
	// The finders return domain.ErrUserNotFound when nobody matches.
	FindByEmail(ctx context.Context, email domain.Email) (domain.User, error)
	FindByPhoneNumber(ctx context.Context, phone domain.PhoneNumber) (domain.User, error)
}

type PasswordHasher interface {
	Hash(ctx context.Context, password string) (string, error)
	Verify(ctx context.Context, password, encoded string) (bool, error)
}

type SessionRepository interface {
	Create(ctx context.Context, session domain.Session) error
	// FindByID returns domain.ErrSessionNotFound when there is none.
	FindByID(ctx context.Context, id string) (domain.Session, error)
	// Revoke is idempotent: revoking a revoked session changes nothing.
	Revoke(ctx context.Context, id string, reason domain.RevokeReason, at time.Time) error
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token domain.RefreshToken) error
	// FindByHashForUpdate locks the row until the transaction ends, and
	// returns domain.ErrInvalidRefreshToken when there is none.
	FindByHashForUpdate(ctx context.Context, hash []byte) (domain.RefreshToken, error)
	MarkUsed(ctx context.Context, id string, at time.Time) error
}

type LoginAttemptRecorder interface {
	Record(ctx context.Context, attempt domain.LoginAttempt) error
}

// LoginLimiter counts failed sign-ins per key and locks the key out
// after too many.
type LoginLimiter interface {
	IsLocked(ctx context.Context, key string) (bool, error)
	RecordFailure(ctx context.Context, key string) error
	Reset(ctx context.Context, key string) error
}

type AccessTokenIssuer interface {
	Issue(userID, sessionID string, now time.Time) (token string, expiresAt time.Time, err error)
}

// RevokedSessions tells the rest of the platform that a session's
// access tokens must be refused from now on.
type RevokedSessions interface {
	Add(ctx context.Context, sessionID string) error
}

type Transactor interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
