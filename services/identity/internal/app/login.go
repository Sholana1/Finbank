package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sholana1/fintechdemo/identity/internal/domain"
)

type LoginInput struct {
	Identifier string
	Password   string
	DeviceID   string
	IPAddress  string
	UserAgent  string
}

type LoginDeps struct {
	Users      UserRepository
	Hasher     PasswordHasher
	Sessions   SessionRepository
	Attempts   LoginAttemptRecorder
	Limiter    LoginLimiter
	Tokens     *SessionTokens
	Tx         Transactor
	Logger     *slog.Logger
	Now        func() time.Time
	SessionTTL time.Duration
	// DummyHash is a real password hash that matches no user. It is
	// verified when the identifier is unknown, so that request takes as
	// long as one for a real user with a wrong password.
	DummyHash string
}

type Login struct {
	deps LoginDeps
}

func NewLogin(deps LoginDeps) *Login {
	return &Login{deps: deps}
}

func (uc *Login) Execute(ctx context.Context, in LoginInput) (TokenPair, error) {
	if strings.TrimSpace(in.DeviceID) == "" {
		return TokenPair{}, domain.ErrMissingDeviceID
	}

	user, err := uc.findUser(ctx, in.Identifier)
	found := err == nil
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return TokenPair{}, fmt.Errorf("login: find user: %w", err)
	}

	limiterKey := limiterKeyFor(user, found, in.Identifier)

	locked, err := uc.deps.Limiter.IsLocked(ctx, limiterKey)
	if err != nil {
		return TokenPair{}, fmt.Errorf("login: check lockout: %w: %w", domain.ErrLoginUnavailable, err)
	}
	if locked {
		uc.audit(ctx, in, user.ID, domain.LoginOutcomeAccountLocked)
		return TokenPair{}, domain.ErrAccountLocked
	}

	passwordHash := uc.deps.DummyHash
	if found {
		passwordHash = user.PasswordHash
	}

	matches, err := uc.deps.Hasher.Verify(ctx, in.Password, passwordHash)
	if err != nil {
		return TokenPair{}, fmt.Errorf("login: verify password: %w", err)
	}

	if !found || !matches {
		if err := uc.deps.Limiter.RecordFailure(ctx, limiterKey); err != nil {
			return TokenPair{}, fmt.Errorf("login: record failure: %w: %w", domain.ErrLoginUnavailable, err)
		}
		uc.audit(ctx, in, user.ID, domain.LoginOutcomeInvalidCredentials)
		return TokenPair{}, domain.ErrInvalidCredentials
	}

	// Checked only after the password is proven, so a stranger cannot
	// learn that an account is suspended.
	if !user.CanSignIn() {
		uc.audit(ctx, in, user.ID, domain.LoginOutcomeAccountDisabled)
		return TokenPair{}, domain.ErrAccountDisabled
	}

	pair, err := uc.openSession(ctx, in, user)
	if err != nil {
		return TokenPair{}, fmt.Errorf("login: open session: %w", err)
	}

	if err := uc.deps.Limiter.Reset(ctx, limiterKey); err != nil {
		uc.deps.Logger.Warn("login: could not reset failure counter", "user_id", user.ID, "error", err)
	}

	return pair, nil
}

// openSession stores the session, its first refresh token and the
// success audit row in one transaction: all three exist, or none do.
func (uc *Login) openSession(ctx context.Context, in LoginInput, user domain.User) (TokenPair, error) {
	sessionID, err := uuid.NewV7()
	if err != nil {
		return TokenPair{}, err
	}
	attemptID, err := uuid.NewV7()
	if err != nil {
		return TokenPair{}, err
	}

	now := uc.deps.Now()
	session := domain.NewSession(sessionID.String(), user.ID, in.DeviceID, in.IPAddress, in.UserAgent, now, now.Add(uc.deps.SessionTTL))

	var pair TokenPair
	err = uc.deps.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := uc.deps.Sessions.Create(ctx, session); err != nil {
			return err
		}

		issued, err := uc.deps.Tokens.Issue(ctx, session, now)
		if err != nil {
			return err
		}
		pair = issued

		return uc.deps.Attempts.Record(ctx, domain.LoginAttempt{
			ID:        attemptID.String(),
			UserID:    user.ID,
			Outcome:   domain.LoginOutcomeSucceeded,
			DeviceID:  in.DeviceID,
			IPAddress: in.IPAddress,
			UserAgent: in.UserAgent,
			CreatedAt: now,
		})
	})
	if err != nil {
		return TokenPair{}, err
	}

	return pair, nil
}

// findUser treats anything containing "@" as an email and everything
// else as a phone number. Input that is neither is simply "not found".
func (uc *Login) findUser(ctx context.Context, identifier string) (domain.User, error) {
	if strings.Contains(identifier, "@") {
		email, err := domain.NewEmail(identifier)
		if err != nil {
			return domain.User{}, domain.ErrUserNotFound
		}
		return uc.deps.Users.FindByEmail(ctx, email)
	}

	phone, err := domain.NewPhoneNumber(identifier)
	if err != nil {
		return domain.User{}, domain.ErrUserNotFound
	}
	return uc.deps.Users.FindByPhoneNumber(ctx, phone)
}

// audit records a failed attempt. It is best effort: if the audit row
// cannot be written, the sign-in is still refused and we log why.
func (uc *Login) audit(ctx context.Context, in LoginInput, userID string, outcome domain.LoginOutcome) {
	id, err := uuid.NewV7()
	if err != nil {
		uc.deps.Logger.Error("login: could not generate audit id", "error", err)
		return
	}

	err = uc.deps.Attempts.Record(ctx, domain.LoginAttempt{
		ID:        id.String(),
		UserID:    userID,
		Outcome:   outcome,
		DeviceID:  in.DeviceID,
		IPAddress: in.IPAddress,
		UserAgent: in.UserAgent,
		CreatedAt: uc.deps.Now(),
	})
	if err != nil {
		uc.deps.Logger.Error("login: could not write audit row", "outcome", string(outcome), "error", err)
	}
}

// limiterKeyFor counts failures per user when we know who it is, so
// guessing by email and by phone share one counter. Unknown identifiers
// get a counter of their own.
func limiterKeyFor(user domain.User, found bool, identifier string) string {
	if found {
		return "user:" + user.ID
	}
	return "unknown:" + strings.ToLower(strings.TrimSpace(identifier))
}
