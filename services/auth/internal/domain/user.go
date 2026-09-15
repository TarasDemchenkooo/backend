package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Email        Email
	PasswordHash string
	Role         Role
	IsVerified   bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewUser(email Email, passwordHash string, role Role) *User {
	now := time.Now().UTC()

	return &User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		IsVerified:   false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
