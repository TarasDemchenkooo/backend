package http

import (
	"log/slog"

	"github.com/TarasDemchenkooo/backend/services/auth/internal/application"
)

type AuthHandler struct {
	registerUC *application.RegisterUseCase
	log        *slog.Logger
}

func NewAuthHandler(register *application.RegisterUseCase, log *slog.Logger) *AuthHandler {
	return &AuthHandler{
		registerUC: register,
		log:        log.With(slog.String("component", "auth_handler")),
	}
}
