package domain

import "time"

// RefreshToken is the stored record of a refresh token. It holds the
// SHA-256 hash only; the token itself exists only on the user's device.
type RefreshToken struct {
	ID        string
	SessionID string
	TokenHash []byte
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
}

func (t RefreshToken) IsUsed() bool {
	return t.UsedAt != nil
}

func (t RefreshToken) IsExpired(now time.Time) bool {
	return !now.Before(t.ExpiresAt)
}
