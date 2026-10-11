package domain

import (
	"net/mail"
	"strings"
)

const maxEmailLength = 254

type Email struct {
	value string
}

func NewEmail(raw string) (Email, error) {
	v := strings.ToLower(strings.TrimSpace(raw))

	if v == "" || len(v) > maxEmailLength {
		return Email{}, ErrInvalidEmail
	}

	addr, err := mail.ParseAddress(v)
	if err != nil || addr.Address != v {
		return Email{}, ErrInvalidEmail
	}

	if !strings.Contains(v[strings.LastIndex(v, "@"):], ".") {
		return Email{}, ErrInvalidEmail
	}

	return Email{value: v}, nil
}

func (e Email) String() string { return e.value }
