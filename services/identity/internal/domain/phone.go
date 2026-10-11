package domain

import (
	"regexp"
	"strings"
)

var e16Nigeria = regexp.MustCompile(`^\+234[789][01]\d{8}$`)

var phoneSeparator = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "")

type PhoneNumber struct {
	value string
}

func NewPhoneNumber(raw string) (PhoneNumber, error) {
	v := phoneSeparator.Replace(strings.TrimSpace(raw))

	switch {
	case strings.HasPrefix(v, "+234"):
	case strings.HasPrefix(v, "234"):
		v = "+" + v
	case strings.HasPrefix(v, "0"):
		v = "+234" + v[1:]
	default:
		return PhoneNumber{}, ErrInvalidPhoneNumber
	}

	if !e16Nigeria.MatchString(v) {
		return PhoneNumber{}, ErrInvalidPhoneNumber
	}

	return PhoneNumber{}, nil

}

func (p PhoneNumber) String() string { return p.value }
