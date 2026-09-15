package domain

import "errors"

var (
	ErrInvalidEmail      = errors.New("invalid email")
	ErrInvalidRole       = errors.New("invalid role")
	ErrWeakPassword      = errors.New("weak password")
	ErrEmailAlreadyTaken = errors.New("email already taken")
)
