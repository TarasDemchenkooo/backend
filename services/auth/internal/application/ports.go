package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/TarasDemchenkooo/backend/services/auth/internal/domain"
)

type UserRepository interface {
	CreateOrReplaceUnverified(ctx context.Context, u *domain.User) error
}

type CodeStore interface {
	Save(ctx context.Context, email domain.Email, code string, ttl time.Duration) error
}

type EventPublisher interface {
	PublishUserRegistered(ctx context.Context, e UserRegisteredEvent) error
}

type PasswordHasher interface {
	Hash(password domain.RawPassword) (string, error)
}

type UserRegisteredEvent struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Code   string    `json:"code"`
}
