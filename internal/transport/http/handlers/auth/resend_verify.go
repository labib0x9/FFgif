package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/pkg/jsonio"
)

type resendRequest struct {
	Email string `json:"email" validate:"required,email,max=50"`
}

func (h *Handler) ResendVerify(w http.ResponseWriter, r *http.Request) {
	var req resendRequest
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&req); err != nil {
		jsonio.SendError(w, "Bad request", http.StatusBadRequest)
		slog.Warn("ResendVerify: bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		jsonio.SendError(w, "field required", 422)
		slog.Warn("ResendVerify: struct validation failed", "error", err)
		return
	}

	err := h.srv.ResendVerify(r.Context(), req.Email)
	if err != nil && !errors.Is(err, auth.ErrMessageQueueFailed) {
		switch {
		case errors.Is(err, auth.ErrUserNotFound):
			jsonio.SendError(w, "user not found", http.StatusNotFound)
		case errors.Is(err, auth.ErrTokenFetchFailed):
			fallthrough
		case errors.Is(err, auth.ErrUserNotVerified):
			jsonio.SendError(w, "not verified", http.StatusForbidden)
		default:
			jsonio.SendError(w, "internal server error", http.StatusInternalServerError)
		}
		slog.Warn("srv.ResendVerify(): failed", "error", err)
		return
	}

	jsonio.SendJson(w, "check mail", http.StatusOK)
}
