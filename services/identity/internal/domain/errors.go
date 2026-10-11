package domain

import "errors"

var (
	ErrInvalidEmail       = errors.New("invalid email address")
	ErrInvalidPhoneNumber = errors.New("invalid mobile number")
	ErrWeakPassword       = errors.New("password must be between 10 and 128 characters")
	ErrEmailTaken         = errors.New("email is already registered")
	ErrPhoneNumberTaken   = errors.New("phone number is already registered")
)
