package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/pkg/jsonio"
)

func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "token is empty", http.StatusBadRequest)
		slog.Warn("Verify() failed", "error", "token is empty")
		return
	}

	if err := h.srv.Verify(r.Context(), token); err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidToken):
			jsonio.SendError(w, "token expired or invalid", http.StatusGone)
		default:
			jsonio.SendError(w, "internal server error", http.StatusInternalServerError)
		}
		slog.Warn("srv.Verify() failed", "error", err)
		return
	}

	jsonio.SendJson(w, "account verified", http.StatusOK)
}
