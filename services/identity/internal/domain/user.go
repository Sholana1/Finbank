package domain

import "time"

type UserStatus string

const (
	UserStatusPendingVerification UserStatus = "PENDING_VERIFICATION"
	UserStatusActive              UserStatus = "ACTIVE"
	UserStatusSuspended           UserStatus = "SUSPENDED"
	UserStatusClosed              UserStatus = "CLOSED"
)

type User struct {
	ID           string
	Email        Email
	PhoneNumber  PhoneNumber
	PasswordHash string
	Status       UserStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewUser(id string, email Email, phone PhoneNumber, passwordHash string, now time.Time) User {
	return User{
		ID:           id,
		Email:        email,
		PhoneNumber:  phone,
		PasswordHash: passwordHash,
		Status:       UserStatusPendingVerification,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func (u User) CanSignIn() bool {
	return u.Status == UserStatusPendingVerification || u.Status == UserStatusActive
}
