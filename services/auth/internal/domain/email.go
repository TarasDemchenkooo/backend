package domain

import (
	"net/mail"
	"strings"
)

type Email string

func NewEmail(s string) (Email, error) {
	s = strings.ToLower(strings.TrimSpace(s))

	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s {
		return "", ErrInvalidEmail
	}

	return Email(s), nil
}

func (e Email) String() string { return string(e) }
