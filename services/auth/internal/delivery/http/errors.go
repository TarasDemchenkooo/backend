package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/TarasDemchenkooo/backend/services/auth/internal/domain"
)

func (h *AuthHandler) handleError(ctx context.Context, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidEmail):
		writeError(w, http.StatusBadRequest, "invalid_email", "email is not a valid address")
	case errors.Is(err, domain.ErrInvalidRole):
		writeError(w, http.StatusBadRequest, "invalid_role", "role is not supported")
	case errors.Is(err, domain.ErrWeakPassword):
		writeError(w, http.StatusBadRequest, "weak_password", "password is too weak")
	case errors.Is(err, domain.ErrEmailAlreadyTaken):
		writeError(w, http.StatusConflict, "email_already_taken", "email is already registered")
	default:
		h.log.ErrorContext(ctx, "unexpected error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
