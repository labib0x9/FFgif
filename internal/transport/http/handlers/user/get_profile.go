package user

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	id := httputil.GetUserId(r.Context())
	if id == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="user id not found"`)
		httputil.SendError(w, auth.AUTH_INVALID_CREDENTIALS, "user id not found", http.StatusUnauthorized)
		slog.Error("user handler - GetProfile() = user_id not found", "err", "user_id not found")
		return
	}
	found, err := h.srv.GetProfile(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrUserNotFound):
			httputil.SendError(w, auth.AUTH_USER_NOT_FOUND, "user not found", http.StatusNotFound)
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("user handler - GetProfile()", "err", err)
		return
	}

	httputil.SendJson(w, found, http.StatusOK)
}
