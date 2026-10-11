package domain

import "time"

type LoginOutcome string

const (
	LoginOutcomeSucceeded          LoginOutcome = "SUCCEEDED"
	LoginOutcomeInvalidCredentials LoginOutcome = "INVALID_CREDENTIALS"
	LoginOutcomeAccountLocked      LoginOutcome = "ACCOUNT_LOCKED"
	LoginOutcomeAccountDisabled    LoginOutcome = "ACCOUNT_DISABLED"
)

// LoginAttempt is one row of the authentication audit trail. UserID is
// empty when the identifier matched nobody. What the person typed is
// never stored: people sometimes type their password into the wrong box.
type LoginAttempt struct {
	ID        string
	UserID    string
	Outcome   LoginOutcome
	DeviceID  string
	IPAddress string
	UserAgent string
	CreatedAt time.Time
}
