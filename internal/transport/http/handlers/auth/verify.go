package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		slog.Warn("auth handler - Verify() = token query param missing", "error", "token is empty")
		httputil.SendError(w, httputil.BAD_REQUEST, "token is empty", http.StatusBadRequest)
		return
	}

	if err := h.srv.Verify(r.Context(), token); err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidToken):
			httputil.SendError(w, auth.AUTH_VERIFY_TOKEN_INVALID, "token expired or invalid", http.StatusGone)
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("auth handler - Verify()", "err", err)
		return
	}

	httputil.SendJson(w, "account verified", http.StatusOK)
}
