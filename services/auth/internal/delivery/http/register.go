package http

import (
	"encoding/json"
	"net/http"

	"github.com/TarasDemchenkooo/backend/services/auth/internal/application"
)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type registerResponse struct {
	UserID    string `json:"user_id"`
	ExpiresIn int    `json:"expires_in"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		h.log.DebugContext(r.Context(), "decode register request", "error", err)
		writeError(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}

	out, err := h.registerUC.Execute(r.Context(), req.toInput())
	if err != nil {
		h.handleError(r.Context(), w, err)
		return
	}

	writeJSON(w, http.StatusCreated, registerResponse{
		UserID:    out.UserID.String(),
		ExpiresIn: int(out.ExpiresIn.Seconds()),
	})
}

func (r registerRequest) toInput() application.RegisterInput {
	return application.RegisterInput{
		Email:    r.Email,
		Password: r.Password,
		Role:     r.Role,
	}
}
