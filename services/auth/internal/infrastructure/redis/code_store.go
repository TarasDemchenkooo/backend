package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/TarasDemchenkooo/backend/services/auth/internal/domain"
)

const codeKeyPrefix = "auth:verify_code:"

type CodeStore struct {
	client *redis.Client
}

func NewCodeStore(client *redis.Client) *CodeStore {
	return &CodeStore{client: client}
}

func (s *CodeStore) Save(ctx context.Context, email domain.Email, code string, ttl time.Duration) error {
	if err := s.client.Set(ctx, s.key(email), code, ttl).Err(); err != nil {
		return fmt.Errorf("set verification code: %w", err)
	}

	return nil
}

func (s *CodeStore) key(email domain.Email) string {
	return codeKeyPrefix + email.String()
}
