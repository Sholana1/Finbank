package domain

import "unicode/utf8"

const (
	minPasswordLength = 10
	maxPasswordLength = 128
)

func ValidatePassword(password string) error {
	n := utf8.RuneCountInString(password)
	if n < minPasswordLength || n > maxPasswordLength {
		return ErrWeakPassword
	}

	return nil
}
