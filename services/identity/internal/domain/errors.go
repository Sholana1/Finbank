package domain

import "errors"

var (
	ErrInvalidEmail       = errors.New("invalid email address")
	ErrInvalidPhoneNumber = errors.New("invalid mobile number")
	ErrWeakPassword       = errors.New("password must be between 10 and 128 characters")
	ErrEmailTaken         = errors.New("email is already registered")
	ErrPhoneNumberTaken   = errors.New("phone number is already registered")

	ErrUserNotFound        = errors.New("user not found")
	ErrMissingDeviceID     = errors.New("device id is required")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrAccountLocked       = errors.New("too many failed attempts, try again later")
	ErrAccountDisabled     = errors.New("account is not allowed to sign in")
	ErrLoginUnavailable    = errors.New("sign in is temporarily unavailable")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrRefreshTokenReuse   = errors.New("refresh token was already used, session revoked")
	ErrSessionNotFound     = errors.New("session not found")
)
