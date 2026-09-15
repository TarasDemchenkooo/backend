package kafka

import (
	"context"
	"log/slog"

	"github.com/TarasDemchenkooo/backend/services/auth/internal/application"
)

type NoopPublisher struct {
	log *slog.Logger
}

func NewNoopPublisher(log *slog.Logger) *NoopPublisher {
	return &NoopPublisher{log: log}
}

func (p *NoopPublisher) PublishUserRegistered(ctx context.Context, e application.UserRegisteredEvent) error {
	p.log.InfoContext(ctx, "publish user.registered (noop)", "user_id", e.UserID)

	return nil
}
