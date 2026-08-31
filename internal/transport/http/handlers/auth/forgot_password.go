package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/pkg/jsonio"
)

type reqForgot struct {
	Email string `json:"email" validate:"required,email,max=50"`
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req reqForgot
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&req); err != nil {
		jsonio.SendError(w, "Bad request", http.StatusBadRequest)
		slog.Warn("ForgotPassword: bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		jsonio.SendError(w, "field required", 422)
		slog.Warn("ForgotPassword: struct validation failed", "error", err)
		return
	}

	err := h.srv.ForgotPassword(r.Context(), req.Email)
	if err != nil && !errors.Is(err, auth.ErrMessageQueueFailed) {
		switch {
		case errors.Is(err, auth.ErrUserNotVerified):
			jsonio.SendError(w, "user is not varified", http.StatusForbidden)
		case errors.Is(err, auth.ErrUserNotFound):
			jsonio.SendError(w, "user not found", http.StatusNotFound)
		default:
			jsonio.SendError(w, "internal server error", http.StatusInternalServerError)
		}
		slog.Warn("srv.ForgotPassword(): failed", "error", err)
		return
	}

	jsonio.SendJson(w, "check mail", http.StatusOK)
}
