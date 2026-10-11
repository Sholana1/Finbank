package domain

import "time"

type RevokeReason string

const (
	RevokeReasonLogout            RevokeReason = "LOGOUT"
	RevokeReasonRefreshTokenReuse RevokeReason = "REFRESH_TOKEN_REUSE"
)

// Session is one sign-in on one device. Every refresh token issued
// during its life belongs to it, so revoking the session kills them all.
type Session struct {
	ID        string
	UserID    string
	DeviceID  string
	IPAddress string
	UserAgent string
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

func NewSession(id, userID, deviceID, ipAddress, userAgent string, now, expiresAt time.Time) Session {
	return Session{
		ID:        id,
		UserID:    userID,
		DeviceID:  deviceID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}
}

func (s Session) IsActive(now time.Time) bool {
	return s.RevokedAt == nil && now.Before(s.ExpiresAt)
}
